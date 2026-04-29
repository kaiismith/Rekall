"""Use case: stitch all segments and produce a Foundry-AI summary."""

from __future__ import annotations

from datetime import UTC, datetime
from typing import TYPE_CHECKING
from uuid import UUID

from intellikat.domain.entities.summary import SessionSummary
from intellikat.domain.entities.transcript_segment import TranscriptSegment
from intellikat.domain.errors import JobFailedError, SummaryParseError, UpstreamError
from intellikat.infrastructure.logging import catalog

if TYPE_CHECKING:
    from intellikat.domain.ports.summarizer import Summarizer
    from intellikat.domain.value_objects.prompt_version import PromptVersion
    from intellikat.infrastructure.logging.logger import ContextLogger

TRUNCATION_MARKER = "\n\n[…transcript truncated for length…]\n\n"


class SummarizeSession:
    def __init__(
        self,
        *,
        summarizer: Summarizer,
        prompt_version: PromptVersion,
        max_input_chars: int,
        run_id: UUID,
        session_id: UUID,
        logger: ContextLogger,
    ) -> None:
        self._summarizer = summarizer
        self._prompt_version = prompt_version
        self._max_chars = max_input_chars
        self._run_id = run_id
        self._session_id = session_id
        self._logger = logger

    async def execute(self, segments: list[TranscriptSegment]) -> SessionSummary:
        full_text = "\n".join(seg.text for seg in segments if seg.text)

        truncated = False
        if len(full_text) > self._max_chars:
            head = full_text[: self._max_chars // 2]
            tail = full_text[-(self._max_chars // 2) :]
            full_text = f"{head}{TRUNCATION_MARKER}{tail}"
            truncated = True
            self._logger.info(
                catalog.SUMMARY_INPUT_TRUNCATED,
                run_id=str(self._run_id),
                transcript_session_id=str(self._session_id),
                max_chars=self._max_chars,
            )

        self._logger.info(
            catalog.SUMMARY_REQUESTED,
            run_id=str(self._run_id),
            transcript_session_id=str(self._session_id),
            input_chars=len(full_text),
            prompt_version=self._prompt_version,
        )

        try:
            result = await self._summarizer.summarize(full_text, self._prompt_version)
        except SummaryParseError as e:
            raise JobFailedError(
                code="SUMMARY_PARSE_FAILED", retryable=True, message=str(e)
            ) from e
        except UpstreamError as e:
            raise JobFailedError(
                code="SUMMARY_UPSTREAM_FAILED", retryable=True, message=str(e)
            ) from e

        self._logger.info(
            catalog.SUMMARY_DONE,
            run_id=str(self._run_id),
            transcript_session_id=str(self._session_id),
            content_chars=len(result.content),
            key_points_count=len(result.key_points or []),
            prompt_tokens=result.prompt_tokens,
            completion_tokens=result.completion_tokens,
        )

        return SessionSummary(
            transcript_session_id=self._session_id,
            run_id=self._run_id,
            content=result.content,
            key_points=result.key_points,
            model_id=self._summarizer.model_id,
            prompt_version=self._prompt_version,
            prompt_tokens=result.prompt_tokens,
            completion_tokens=result.completion_tokens,
            input_truncated=truncated,
            processed_at=datetime.now(UTC),
        )
