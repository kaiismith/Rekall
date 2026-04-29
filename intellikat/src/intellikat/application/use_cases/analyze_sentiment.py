"""Use case: classify each segment's text into a sentiment label + confidence."""

from __future__ import annotations

from datetime import UTC, datetime
from typing import TYPE_CHECKING
from uuid import UUID

from intellikat.domain.entities.sentiment import SegmentSentiment
from intellikat.domain.entities.transcript_segment import TranscriptSegment
from intellikat.domain.errors import JobFailedError, UpstreamError
from intellikat.infrastructure.logging import catalog

if TYPE_CHECKING:
    from intellikat.domain.ports.sentiment_analyzer import SentimentAnalyzer
    from intellikat.infrastructure.logging.logger import ContextLogger


class AnalyzeSentiment:
    def __init__(
        self,
        *,
        analyzer: SentimentAnalyzer,
        failure_ratio_threshold: float,
        run_id: UUID,
        session_id: UUID,
        logger: ContextLogger,
    ) -> None:
        self._analyzer = analyzer
        self._threshold = failure_ratio_threshold
        self._run_id = run_id
        self._session_id = session_id
        self._logger = logger

    async def execute(
        self, segments: list[TranscriptSegment]
    ) -> tuple[list[SegmentSentiment], int]:
        non_empty = [s for s in segments if s.text and s.text.strip()]
        skipped = len(segments) - len(non_empty)
        if skipped:
            self._logger.info(
                catalog.SENTIMENT_SKIP_EMPTY,
                run_id=str(self._run_id),
                transcript_session_id=str(self._session_id),
                skipped=skipped,
            )

        if not non_empty:
            return [], 0

        self._logger.info(
            catalog.SENTIMENT_BATCH_STARTED,
            run_id=str(self._run_id),
            transcript_session_id=str(self._session_id),
            batch_size=len(non_empty),
        )

        try:
            results = await self._analyzer.analyze(non_empty)
        except UpstreamError as e:
            raise JobFailedError(
                code="SENTIMENT_BATCH_FAILED", retryable=True, message=str(e)
            ) from e

        if len(results) != len(non_empty):
            raise JobFailedError(
                code="SENTIMENT_RESULT_LENGTH_MISMATCH",
                retryable=True,
                message=f"analyzer returned {len(results)} results for {len(non_empty)} segments",
            )

        sentiments: list[SegmentSentiment] = []
        failed = 0
        now = datetime.now(UTC)
        for seg, res in zip(non_empty, results, strict=True):
            if res.error is not None or res.label is None:
                failed += 1
                self._logger.warn(
                    catalog.SENTIMENT_SEGMENT_FAILED,
                    run_id=str(self._run_id),
                    transcript_session_id=str(self._session_id),
                    segment_index=seg.segment_index,
                    reason=res.error or "no label",
                )
                continue
            sentiments.append(
                SegmentSentiment(
                    transcript_segment_id=seg.id,
                    transcript_session_id=self._session_id,
                    run_id=self._run_id,
                    label=res.label,
                    confidence=res.confidence,
                    model_id=self._analyzer.model_id,
                    model_revision=self._analyzer.model_revision,
                    hf_mode=self._analyzer.hf_mode,
                    processed_at=now,
                )
            )

        if non_empty and (failed / len(non_empty)) > self._threshold:
            raise JobFailedError(
                code="SENTIMENT_FAILURE_RATIO_EXCEEDED",
                retryable=True,
                message=f"{failed}/{len(non_empty)} segments failed (threshold {self._threshold})",
            )

        self._logger.info(
            catalog.SENTIMENT_BATCH_DONE,
            run_id=str(self._run_id),
            transcript_session_id=str(self._session_id),
            persisted=len(sentiments),
            failed=failed,
        )
        return sentiments, failed
