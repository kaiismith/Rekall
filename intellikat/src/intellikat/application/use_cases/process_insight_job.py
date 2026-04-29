"""Orchestrator: receive a JobReference, run sentiment + summarization,
persist results in one transaction, write the audit row.

The only place that knows the full graph of ports.
"""

from __future__ import annotations

from datetime import UTC, datetime
from typing import TYPE_CHECKING, Callable
from uuid import uuid4

from intellikat.application.use_cases.analyze_sentiment import AnalyzeSentiment
from intellikat.application.use_cases.summarize_session import SummarizeSession
from intellikat.domain.entities.insight_job import InsightJob
from intellikat.domain.errors import IntellikatError, JobFailedError
from intellikat.infrastructure.logging import catalog

if TYPE_CHECKING:
    from intellikat.domain.ports.insight_writer import InsightWriter
    from intellikat.domain.ports.sentiment_analyzer import SentimentAnalyzer
    from intellikat.domain.ports.summarizer import Summarizer
    from intellikat.domain.ports.transcript_reader import TranscriptReader
    from intellikat.domain.value_objects.job_reference import JobReference
    from intellikat.domain.value_objects.prompt_version import PromptVersion
    from intellikat.infrastructure.logging.logger import ContextLogger


class ProcessInsightJob:
    def __init__(
        self,
        *,
        reader: TranscriptReader,
        analyzer: SentimentAnalyzer,
        summarizer: Summarizer,
        writer_factory: Callable[[], "InsightWriter"],
        prompt_version: PromptVersion,
        failure_ratio_threshold: float,
        max_summary_input_chars: int,
        logger: ContextLogger,
    ) -> None:
        self._reader = reader
        self._analyzer = analyzer
        self._summarizer = summarizer
        self._writer_factory = writer_factory
        self._prompt_version = prompt_version
        self._failure_ratio_threshold = failure_ratio_threshold
        self._max_summary_input_chars = max_summary_input_chars
        self._logger = logger

    async def execute(self, ref: JobReference) -> None:
        run_id = uuid4()
        started_at = datetime.now(UTC)
        job = InsightJob(run_id=run_id, reference=ref, started_at=started_at)

        self._logger.info(
            catalog.JOB_STARTED,
            run_id=str(run_id),
            transcript_session_id=str(ref.transcript_session_id),
            correlation_id=ref.correlation_id,
            event_type=ref.event_type,
            hf_mode=self._analyzer.hf_mode,
        )

        try:
            session = await self._reader.get_session(ref.transcript_session_id)
            if session is None:
                raise JobFailedError(
                    code="SESSION_NOT_FOUND",
                    retryable=False,
                    message=f"no transcript_sessions row for {ref.transcript_session_id}",
                )
            if session.status not in ("ended", "expired"):
                raise JobFailedError(
                    code="SESSION_NOT_CLOSED",
                    retryable=True,
                    message=f"session status is {session.status!r}; expected ended/expired",
                )

            seg_from = ref.segment_index_range.from_ if ref.segment_index_range else None
            seg_to = ref.segment_index_range.to if ref.segment_index_range else None
            segments = await self._reader.list_segments(
                session.id, segment_index_from=seg_from, segment_index_to=seg_to
            )
            job.segments_total = len(segments)
            if not segments:
                raise JobFailedError(
                    code="SESSION_HAS_NO_SEGMENTS", retryable=False
                )

            sentiment_uc = AnalyzeSentiment(
                analyzer=self._analyzer,
                failure_ratio_threshold=self._failure_ratio_threshold,
                run_id=run_id,
                session_id=session.id,
                logger=self._logger,
            )
            sentiments, segments_failed = await sentiment_uc.execute(segments)
            job.segments_processed = len(sentiments)
            job.segments_failed = segments_failed

            summary_uc = SummarizeSession(
                summarizer=self._summarizer,
                prompt_version=self._prompt_version,
                max_input_chars=self._max_summary_input_chars,
                run_id=run_id,
                session_id=session.id,
                logger=self._logger,
            )
            summary = await summary_uc.execute(segments)

            async with self._writer_factory() as uow:
                if sentiments:
                    await uow.save_sentiments(sentiments)
                await uow.save_summary(summary)
                job.summary_persisted = True
                job.status = "completed"
                job.finished_at = datetime.now(UTC)
                await uow.save_job(job)

            self._logger.info(
                catalog.JOB_COMPLETED,
                run_id=str(run_id),
                transcript_session_id=str(ref.transcript_session_id),
                segments_total=job.segments_total,
                segments_processed=job.segments_processed,
                segments_failed=job.segments_failed,
                summary_persisted=job.summary_persisted,
                duration_ms=int(
                    (job.finished_at - job.started_at).total_seconds() * 1000
                )
                if job.finished_at
                else None,
            )

        except JobFailedError as e:
            job.status = "failed"
            job.error_code = e.code
            job.error_message = str(e)
            job.finished_at = datetime.now(UTC)
            await self._best_effort_audit(job)
            raise
        except IntellikatError as e:
            job.status = "failed"
            job.error_code = "UNEXPECTED_DOMAIN_ERROR"
            job.error_message = str(e)
            job.finished_at = datetime.now(UTC)
            await self._best_effort_audit(job)
            raise JobFailedError(
                code=job.error_code, retryable=True, message=str(e)
            ) from e

    async def _best_effort_audit(self, job: InsightJob) -> None:
        """Persist the audit row even on failure; never propagate further."""
        try:
            async with self._writer_factory() as uow:
                await uow.save_job(job)
        except Exception as e:  # noqa: BLE001
            self._logger.error(
                catalog.DB_WRITE_FAILED,
                run_id=str(job.run_id),
                reason="audit row write failed",
                err=str(e),
            )
