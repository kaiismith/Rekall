"""Unit tests for `AnalyzeSentiment` (Requirement 18.2)."""

from __future__ import annotations

from datetime import UTC, datetime
from typing import Sequence
from uuid import uuid4

import pytest

from intellikat.application.dto.sentiment_result import SentimentResult
from intellikat.application.use_cases.analyze_sentiment import AnalyzeSentiment
from intellikat.domain.entities.transcript_segment import TranscriptSegment
from intellikat.domain.errors import JobFailedError, UpstreamError
from intellikat.domain.value_objects.sentiment_label import SentimentLabel


class _FakeAnalyzer:
    model_id = "MarieAngeA13/Sentiment-Analysis-BERT"
    model_revision = "deadbeef"
    hf_mode = "hosted"

    def __init__(self, results: Sequence[SentimentResult] | Exception) -> None:
        self._results = results

    async def warmup(self) -> None:
        return None

    async def health_check(self) -> bool:
        return True

    async def analyze(
        self, segments: Sequence[TranscriptSegment]
    ) -> Sequence[SentimentResult]:
        if isinstance(self._results, Exception):
            raise self._results
        return self._results


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
    segs = [_seg(0, "hello"), _seg(1, "world")]
    analyzer = _FakeAnalyzer(
        [
            SentimentResult(label=SentimentLabel.POSITIVE, confidence=0.9),
            SentimentResult(label=SentimentLabel.NEGATIVE, confidence=0.7),
        ]
    )
    uc = AnalyzeSentiment(
        analyzer=analyzer,
        failure_ratio_threshold=0.5,
        run_id=uuid4(),
        session_id=uuid4(),
        logger=logger,
    )
    sentiments, failed = await uc.execute(segs)
    assert len(sentiments) == 2
    assert failed == 0
    assert sentiments[0].label == SentimentLabel.POSITIVE
    assert sentiments[0].model_id == "MarieAngeA13/Sentiment-Analysis-BERT"
    assert sentiments[0].model_revision == "deadbeef"
    assert sentiments[0].hf_mode == "hosted"


@pytest.mark.asyncio
async def test_empty_text_skipped(logger) -> None:  # type: ignore[no-untyped-def]
    segs = [_seg(0, ""), _seg(1, "  "), _seg(2, "real")]
    analyzer = _FakeAnalyzer(
        [SentimentResult(label=SentimentLabel.NEUTRAL, confidence=0.5)]
    )
    uc = AnalyzeSentiment(
        analyzer=analyzer,
        failure_ratio_threshold=0.5,
        run_id=uuid4(),
        session_id=uuid4(),
        logger=logger,
    )
    sentiments, failed = await uc.execute(segs)
    assert len(sentiments) == 1
    assert failed == 0


@pytest.mark.asyncio
async def test_per_segment_failure_tolerated(logger) -> None:  # type: ignore[no-untyped-def]
    segs = [_seg(0, "good"), _seg(1, "bad"), _seg(2, "ok")]
    analyzer = _FakeAnalyzer(
        [
            SentimentResult(label=SentimentLabel.POSITIVE, confidence=0.8),
            SentimentResult(label=None, confidence=0.0, error="upstream 500"),
            SentimentResult(label=SentimentLabel.NEUTRAL, confidence=0.6),
        ]
    )
    uc = AnalyzeSentiment(
        analyzer=analyzer,
        failure_ratio_threshold=0.5,
        run_id=uuid4(),
        session_id=uuid4(),
        logger=logger,
    )
    sentiments, failed = await uc.execute(segs)
    assert len(sentiments) == 2
    assert failed == 1


@pytest.mark.asyncio
async def test_threshold_exceeded_raises(logger) -> None:  # type: ignore[no-untyped-def]
    segs = [_seg(i, "x") for i in range(4)]
    analyzer = _FakeAnalyzer(
        [
            SentimentResult(label=None, error="boom"),
            SentimentResult(label=None, error="boom"),
            SentimentResult(label=None, error="boom"),
            SentimentResult(label=SentimentLabel.POSITIVE, confidence=0.9),
        ]
    )
    uc = AnalyzeSentiment(
        analyzer=analyzer,
        failure_ratio_threshold=0.25,
        run_id=uuid4(),
        session_id=uuid4(),
        logger=logger,
    )
    with pytest.raises(JobFailedError) as exc:
        await uc.execute(segs)
    assert exc.value.code == "SENTIMENT_FAILURE_RATIO_EXCEEDED"
    assert exc.value.retryable is True


@pytest.mark.asyncio
async def test_upstream_error_raises_retryable(logger) -> None:  # type: ignore[no-untyped-def]
    analyzer = _FakeAnalyzer(UpstreamError("hf 503"))
    uc = AnalyzeSentiment(
        analyzer=analyzer,
        failure_ratio_threshold=0.25,
        run_id=uuid4(),
        session_id=uuid4(),
        logger=logger,
    )
    with pytest.raises(JobFailedError) as exc:
        await uc.execute([_seg(0, "x")])
    assert exc.value.code == "SENTIMENT_BATCH_FAILED"
    assert exc.value.retryable is True


@pytest.mark.asyncio
async def test_all_segments_empty_returns_zero(logger) -> None:  # type: ignore[no-untyped-def]
    analyzer = _FakeAnalyzer([])
    uc = AnalyzeSentiment(
        analyzer=analyzer,
        failure_ratio_threshold=0.25,
        run_id=uuid4(),
        session_id=uuid4(),
        logger=logger,
    )
    sentiments, failed = await uc.execute([_seg(0, ""), _seg(1, "  ")])
    assert sentiments == []
    assert failed == 0


@pytest.mark.asyncio
async def test_result_length_mismatch_raises(logger) -> None:  # type: ignore[no-untyped-def]
    segs = [_seg(0, "a"), _seg(1, "b")]
    analyzer = _FakeAnalyzer([SentimentResult(label=SentimentLabel.POSITIVE, confidence=0.9)])
    uc = AnalyzeSentiment(
        analyzer=analyzer,
        failure_ratio_threshold=0.25,
        run_id=uuid4(),
        session_id=uuid4(),
        logger=logger,
    )
    with pytest.raises(JobFailedError) as exc:
        await uc.execute(segs)
    assert exc.value.code == "SENTIMENT_RESULT_LENGTH_MISMATCH"
