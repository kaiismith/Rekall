"""Unit tests for `HFInferenceClient` using respx."""

from __future__ import annotations

from datetime import UTC, datetime
from uuid import uuid4

import httpx
import pytest
import respx
from pydantic import SecretStr

from intellikat.domain.entities.transcript_segment import TranscriptSegment
from intellikat.domain.value_objects.sentiment_label import SentimentLabel
from intellikat.infrastructure.sentiment.hf_inference_client import HFInferenceClient

BASE_URL = "https://api-inference.test.example"
MODEL = "MarieAngeA13/Sentiment-Analysis-BERT"


def _seg(text: str) -> TranscriptSegment:
    return TranscriptSegment(
        id=uuid4(),
        session_id=uuid4(),
        segment_index=0,
        speaker_user_id=uuid4(),
        text=text,
        language="en",
        confidence=0.9,
        start_ms=0,
        end_ms=500,
        segment_started_at=datetime.now(UTC),
    )


@pytest.mark.asyncio
async def test_happy_path_parses_top_label() -> None:
    payload = [
        [
            {"label": "POSITIVE", "score": 0.92},
            {"label": "NEUTRAL", "score": 0.05},
            {"label": "NEGATIVE", "score": 0.03},
        ]
    ]
    client = HFInferenceClient(
        base_url=BASE_URL, model_id=MODEL, token=SecretStr("hf_x"), timeout_s=5
    )
    try:
        with respx.mock(base_url=BASE_URL) as router:
            router.post(f"/models/{MODEL}").mock(return_value=httpx.Response(200, json=payload))
            results = await client.analyze([_seg("hello")])
        assert len(results) == 1
        assert results[0].label == SentimentLabel.POSITIVE
        assert results[0].confidence == pytest.approx(0.92)
        assert results[0].error is None
    finally:
        await client.close()


@pytest.mark.asyncio
async def test_503_retried_once_then_success() -> None:
    success_payload = [[{"label": "NEGATIVE", "score": 0.99}]]
    client = HFInferenceClient(
        base_url=BASE_URL, model_id=MODEL, token=SecretStr("hf_x"), timeout_s=5
    )
    try:
        with respx.mock(base_url=BASE_URL) as router:
            router.post(f"/models/{MODEL}").mock(
                side_effect=[
                    httpx.Response(503, json={"estimated_time": 0.01}),
                    httpx.Response(200, json=success_payload),
                ]
            )
            results = await client.analyze([_seg("hello")])
        assert results[0].label == SentimentLabel.NEGATIVE
    finally:
        await client.close()


@pytest.mark.asyncio
async def test_429_with_retry_after_then_success() -> None:
    success = [[{"label": "POSITIVE", "score": 0.7}]]
    client = HFInferenceClient(
        base_url=BASE_URL, model_id=MODEL, token=SecretStr("hf_x"), timeout_s=5
    )
    try:
        with respx.mock(base_url=BASE_URL) as router:
            router.post(f"/models/{MODEL}").mock(
                side_effect=[
                    httpx.Response(429, headers={"Retry-After": "0"}),
                    httpx.Response(200, json=success),
                ]
            )
            results = await client.analyze([_seg("hello")])
        assert results[0].label == SentimentLabel.POSITIVE
    finally:
        await client.close()


@pytest.mark.asyncio
async def test_persistent_500_returns_error_result() -> None:
    client = HFInferenceClient(
        base_url=BASE_URL, model_id=MODEL, token=SecretStr("hf_x"), timeout_s=5
    )
    try:
        with respx.mock(base_url=BASE_URL) as router:
            router.post(f"/models/{MODEL}").mock(return_value=httpx.Response(500))
            results = await client.analyze([_seg("hello")])
        assert results[0].label is None
        assert results[0].error is not None
        assert "500" in results[0].error
    finally:
        await client.close()


@pytest.mark.asyncio
async def test_unknown_label_yields_error_result() -> None:
    payload = [[{"label": "MYSTERY", "score": 0.99}]]
    client = HFInferenceClient(
        base_url=BASE_URL, model_id=MODEL, token=SecretStr("hf_x"), timeout_s=5
    )
    try:
        with respx.mock(base_url=BASE_URL) as router:
            router.post(f"/models/{MODEL}").mock(return_value=httpx.Response(200, json=payload))
            results = await client.analyze([_seg("hello")])
        assert results[0].label is None
        assert results[0].error is not None
        assert "parse error" in results[0].error
    finally:
        await client.close()


@pytest.mark.asyncio
async def test_warmup_sets_revision() -> None:
    client = HFInferenceClient(
        base_url=BASE_URL, model_id=MODEL, token=SecretStr("hf_x"), timeout_s=5
    )
    try:
        with respx.mock(base_url=BASE_URL) as router:
            router.get(f"/api/models/{MODEL}").mock(
                return_value=httpx.Response(200, json={"sha": "deadbeef"})
            )
            await client.warmup()
        assert client.model_revision == "deadbeef"
    finally:
        await client.close()
