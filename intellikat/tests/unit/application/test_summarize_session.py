"""Unit tests for `SummarizeSession`."""

from __future__ import annotations

from datetime import UTC, datetime
from uuid import uuid4

import pytest

from intellikat.application.dto.summary_result import SummaryResult
from intellikat.application.use_cases.summarize_session import (
    TRUNCATION_MARKER,
    SummarizeSession,
)
from intellikat.domain.entities.transcript_segment import TranscriptSegment
from intellikat.domain.errors import JobFailedError, SummaryParseError, UpstreamError
from intellikat.domain.value_objects.prompt_version import make_prompt_version


class _FakeSummarizer:
    model_id = "gpt-4o-mini"

    def __init__(self, result: SummaryResult | Exception) -> None:
        self._result = result
        self.last_input: str | None = None
        self.last_prompt_version: str | None = None

    async def health_check(self) -> bool:
        return True

    async def summarize(self, transcript: str, prompt_version: str) -> SummaryResult:
        self.last_input = transcript
        self.last_prompt_version = prompt_version
        if isinstance(self._result, Exception):
            raise self._result
        return self._result


def _seg(idx: int, text: str) -> TranscriptSegment:
    return TranscriptSegment(
        id=uuid4(),
        session_id=uuid4(),
        segment_index=idx,
        speaker_user_id=uuid4(),
        text=text,
        language="en",
        confidence=0.9,
        start_ms=idx * 1000,
        end_ms=idx * 1000 + 500,
        segment_started_at=datetime.now(UTC),
    )


@pytest.mark.asyncio
async def test_happy_path(logger) -> None:  # type: ignore[no-untyped-def]
    summarizer = _FakeSummarizer(
        SummaryResult(
            content="A short summary.",
            key_points=["one", "two"],
            prompt_tokens=120,
            completion_tokens=40,
        )
    )
    uc = SummarizeSession(
        summarizer=summarizer,
        prompt_version=make_prompt_version("sum-v1"),
        max_input_chars=10_000,
        run_id=uuid4(),
        session_id=uuid4(),
        logger=logger,
    )
    summary = await uc.execute([_seg(0, "hello"), _seg(1, "world")])
    assert summary.content == "A short summary."
    assert summary.key_points == ["one", "two"]
    assert summary.input_truncated is False
    assert summary.prompt_version == "sum-v1"
    assert summary.model_id == "gpt-4o-mini"
    assert summarizer.last_input == "hello\nworld"


@pytest.mark.asyncio
async def test_head_tail_truncation_applied(logger) -> None:  # type: ignore[no-untyped-def]
    long_text = "x" * 20_000
    summarizer = _FakeSummarizer(
        SummaryResult(content="ok", key_points=None, prompt_tokens=None, completion_tokens=None)
    )
    uc = SummarizeSession(
        summarizer=summarizer,
        prompt_version=make_prompt_version("sum-v1"),
        max_input_chars=1_000,
        run_id=uuid4(),
        session_id=uuid4(),
        logger=logger,
    )
    summary = await uc.execute([_seg(0, long_text)])
    assert summary.input_truncated is True
    assert TRUNCATION_MARKER in (summarizer.last_input or "")


@pytest.mark.asyncio
async def test_parse_error_raises_retryable(logger) -> None:  # type: ignore[no-untyped-def]
    summarizer = _FakeSummarizer(SummaryParseError("bad json"))
    uc = SummarizeSession(
        summarizer=summarizer,
        prompt_version=make_prompt_version("sum-v1"),
        max_input_chars=10_000,
        run_id=uuid4(),
        session_id=uuid4(),
        logger=logger,
    )
    with pytest.raises(JobFailedError) as exc:
        await uc.execute([_seg(0, "x")])
    assert exc.value.code == "SUMMARY_PARSE_FAILED"
    assert exc.value.retryable is True


@pytest.mark.asyncio
async def test_upstream_error_raises_retryable(logger) -> None:  # type: ignore[no-untyped-def]
    summarizer = _FakeSummarizer(UpstreamError("foundry 500"))
    uc = SummarizeSession(
        summarizer=summarizer,
        prompt_version=make_prompt_version("sum-v1"),
        max_input_chars=10_000,
        run_id=uuid4(),
        session_id=uuid4(),
        logger=logger,
    )
    with pytest.raises(JobFailedError) as exc:
        await uc.execute([_seg(0, "x")])
    assert exc.value.code == "SUMMARY_UPSTREAM_FAILED"
    assert exc.value.retryable is True
