"""Unit tests for `ProcessInsightJob` orchestrator."""

from __future__ import annotations

from datetime import UTC, datetime
from typing import Any, Sequence
from uuid import UUID, uuid4

import pytest

from intellikat.application.dto.sentiment_result import SentimentResult
from intellikat.application.dto.summary_result import SummaryResult
from intellikat.application.use_cases.process_insight_job import ProcessInsightJob
from intellikat.domain.entities.insight_job import InsightJob
from intellikat.domain.entities.sentiment import SegmentSentiment
from intellikat.domain.entities.summary import SessionSummary
from intellikat.domain.entities.transcript_segment import TranscriptSegment
from intellikat.domain.entities.transcript_session import TranscriptSession
from intellikat.domain.errors import JobFailedError
from intellikat.domain.value_objects.engine_snapshot import EngineSnapshot
from intellikat.domain.value_objects.job_reference import (
    JobReference,
    Scope,
    SegmentIndexRange,
)
from intellikat.domain.value_objects.prompt_version import make_prompt_version
from intellikat.domain.value_objects.sentiment_label import SentimentLabel


# ── fakes ───────────────────────────────────────────────────────────────────


class _FakeReader:
    def __init__(
        self,
        session: TranscriptSession | None,
        segments: list[TranscriptSegment],
    ) -> None:
        self._session = session
        self._segments = segments

    async def get_session(self, session_id: UUID) -> TranscriptSession | None:
        return self._session

    async def list_segments(self, session_id, segment_index_from=None, segment_index_to=None):  # type: ignore[no-untyped-def]
        return list(self._segments)


class _FakeAnalyzer:
    model_id = "MarieAngeA13/Sentiment-Analysis-BERT"
    model_revision = "abc"
    hf_mode = "hosted"

    async def warmup(self) -> None:
        return None

    async def health_check(self) -> bool:
        return True

    async def analyze(self, segments: Sequence[TranscriptSegment]) -> Sequence[SentimentResult]:
        return [
            SentimentResult(label=SentimentLabel.POSITIVE, confidence=0.9)
            for _ in segments
        ]


class _FakeSummarizer:
    model_id = "gpt-4o-mini"

    async def health_check(self) -> bool:
        return True

    async def summarize(self, transcript: str, prompt_version: str) -> SummaryResult:
        return SummaryResult(
            content="summary",
            key_points=["a"],
            prompt_tokens=100,
            completion_tokens=20,
        )


class _FakeWriter:
    def __init__(self, fail_on_save_summary: bool = False) -> None:
        self.saved_sentiments: list[SegmentSentiment] = []
        self.saved_summaries: list[SessionSummary] = []
        self.saved_jobs: list[InsightJob] = []
        self.committed = False
        self.rolled_back = False
        self._fail_on_save_summary = fail_on_save_summary

    async def __aenter__(self) -> "_FakeWriter":
        return self

    async def __aexit__(self, exc_type, exc, tb) -> None:  # type: ignore[no-untyped-def]
        if exc_type is None:
            self.committed = True
        else:
            self.rolled_back = True

    async def save_job(self, job: InsightJob) -> None:
        self.saved_jobs.append(job)

    async def save_sentiments(self, sentiments: Sequence[SegmentSentiment]) -> None:
        self.saved_sentiments.extend(sentiments)

    async def save_summary(self, summary: SessionSummary) -> None:
        if self._fail_on_save_summary:
            raise RuntimeError("simulated DB error")
        self.saved_summaries.append(summary)


# ── helpers ─────────────────────────────────────────────────────────────────


def _job_ref(session_id: UUID) -> JobReference:
    return JobReference(
        schema_version="1",
        job_id=uuid4(),
        event_type="transcript.session.closed",
        transcript_session_id=session_id,
        scope=Scope(kind="call", id=uuid4()),
        segment_index_range=SegmentIndexRange.model_validate({"from": 0, "to": 5}),
        speaker_user_id=uuid4(),
        engine_snapshot=EngineSnapshot(engine_mode="openai", model_id="whisper-1"),
        occurred_at=datetime.now(UTC),
        correlation_id="cid-1",
    )


def _session(session_id: UUID, status: str = "ended") -> TranscriptSession:
    return TranscriptSession(
        id=session_id,
        speaker_user_id=uuid4(),
        call_id=uuid4(),
        meeting_id=None,
        engine_mode="openai",
        model_id="whisper-1",
        language_requested="en",
        status=status,  # type: ignore[arg-type]
        started_at=datetime.now(UTC),
        ended_at=datetime.now(UTC),
        expires_at=datetime.now(UTC),
        correlation_id="cid-1",
    )


def _segment(idx: int, session_id: UUID) -> TranscriptSegment:
    return TranscriptSegment(
        id=uuid4(),
        session_id=session_id,
        segment_index=idx,
        speaker_user_id=uuid4(),
        text=f"segment {idx}",
        language="en",
        confidence=0.9,
        start_ms=idx * 1000,
        end_ms=idx * 1000 + 500,
        segment_started_at=datetime.now(UTC),
    )


def _build(reader, writer, *, summarizer=None, analyzer=None, logger=None):  # type: ignore[no-untyped-def]
    return ProcessInsightJob(
        reader=reader,
        analyzer=analyzer or _FakeAnalyzer(),
        summarizer=summarizer or _FakeSummarizer(),
        writer_factory=lambda: writer,
        prompt_version=make_prompt_version("sum-v1"),
        failure_ratio_threshold=0.5,
        max_summary_input_chars=10_000,
        logger=logger,
    )


# ── tests ───────────────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_happy_path(logger) -> None:  # type: ignore[no-untyped-def]
    sid = uuid4()
    reader = _FakeReader(_session(sid), [_segment(i, sid) for i in range(3)])
    writer = _FakeWriter()
    uc = _build(reader, writer, logger=logger)
    await uc.execute(_job_ref(sid))
    assert len(writer.saved_sentiments) == 3
    assert len(writer.saved_summaries) == 1
    assert len(writer.saved_jobs) == 1
    assert writer.saved_jobs[0].status == "completed"
    assert writer.saved_jobs[0].summary_persisted is True
    assert writer.committed is True


@pytest.mark.asyncio
async def test_session_not_found_raises_non_retryable(logger) -> None:  # type: ignore[no-untyped-def]
    sid = uuid4()
    reader = _FakeReader(None, [])
    writer = _FakeWriter()
    uc = _build(reader, writer, logger=logger)
    with pytest.raises(JobFailedError) as exc:
        await uc.execute(_job_ref(sid))
    assert exc.value.code == "SESSION_NOT_FOUND"
    assert exc.value.retryable is False
    # audit row still attempted
    assert len(writer.saved_jobs) == 1
    assert writer.saved_jobs[0].status == "failed"


@pytest.mark.asyncio
async def test_session_not_closed_raises_retryable(logger) -> None:  # type: ignore[no-untyped-def]
    sid = uuid4()
    reader = _FakeReader(_session(sid, status="active"), [_segment(0, sid)])
    writer = _FakeWriter()
    uc = _build(reader, writer, logger=logger)
    with pytest.raises(JobFailedError) as exc:
        await uc.execute(_job_ref(sid))
    assert exc.value.code == "SESSION_NOT_CLOSED"
    assert exc.value.retryable is True


@pytest.mark.asyncio
async def test_session_with_no_segments_raises(logger) -> None:  # type: ignore[no-untyped-def]
    sid = uuid4()
    reader = _FakeReader(_session(sid), [])
    writer = _FakeWriter()
    uc = _build(reader, writer, logger=logger)
    with pytest.raises(JobFailedError) as exc:
        await uc.execute(_job_ref(sid))
    assert exc.value.code == "SESSION_HAS_NO_SEGMENTS"
    assert exc.value.retryable is False


@pytest.mark.asyncio
async def test_writer_failure_rolls_back(logger) -> None:  # type: ignore[no-untyped-def]
    sid = uuid4()
    reader = _FakeReader(_session(sid), [_segment(0, sid)])
    writer = _FakeWriter(fail_on_save_summary=True)
    uc = _build(reader, writer, logger=logger)
    with pytest.raises(Exception):
        await uc.execute(_job_ref(sid))
    assert writer.rolled_back is True
    assert writer.committed is False
