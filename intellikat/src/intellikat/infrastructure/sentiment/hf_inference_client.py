"""Hugging Face Inference API adapter.

One POST per segment to `{base}/models/{model_id}`. Bearer auth via `HF_TOKEN`.
Handles 503 (model loading) with a one-shot backoff and 429 with `Retry-After`.
"""

from __future__ import annotations

import asyncio
from typing import TYPE_CHECKING, Any, Sequence

import httpx
from pydantic import SecretStr

from intellikat.application.dto.sentiment_result import SentimentResult
from intellikat.infrastructure.sentiment import label_map

if TYPE_CHECKING:
    from intellikat.domain.entities.transcript_segment import TranscriptSegment

_MAX_RETRIES = 3
_DEFAULT_LOAD_BACKOFF = 20.0
_HEALTH_PROBE_TEXT = "ok"


class HFInferenceClient:
    """Implements the `SentimentAnalyzer` port for the hosted HF Inference API."""

    hf_mode = "hosted"

    def __init__(
        self,
        *,
        base_url: str,
        model_id: str,
        token: SecretStr,
        timeout_s: int,
    ) -> None:
        self._base_url = base_url.rstrip("/")
        self._model_id = model_id
        self._infer_path = f"/models/{model_id}"
        self._client = httpx.AsyncClient(
            base_url=self._base_url,
            timeout=timeout_s,
            headers={"Authorization": f"Bearer {token.get_secret_value()}"},
        )
        self._revision: str | None = None

    @property
    def model_id(self) -> str:
        return self._model_id

    @property
    def model_revision(self) -> str | None:
        return self._revision

    async def warmup(self) -> None:
        """Fetch the commit SHA + prime the model so it's loaded on first inference."""
        resp = await self._client.get(f"/api/models/{self._model_id}")
        resp.raise_for_status()
        body = resp.json() if resp.content else {}
        self._revision = body.get("sha") if isinstance(body, dict) else None

    async def health_check(self) -> bool:
        try:
            resp = await self._client.post(
                self._infer_path,
                json={"inputs": _HEALTH_PROBE_TEXT, "options": {"wait_for_model": False}},
            )
            return resp.status_code in (200, 503)
        except httpx.HTTPError:
            return False

    async def analyze(self, segments: Sequence[TranscriptSegment]) -> Sequence[SentimentResult]:
        results: list[SentimentResult] = []
        for seg in segments:
            results.append(await self._analyze_one(seg.text))
        return results

    async def _analyze_one(self, text: str) -> SentimentResult:
        last_error: str | None = None
        load_retried = False
        rate_retries = 0

        for _ in range(_MAX_RETRIES + 1):
            try:
                resp = await self._client.post(
                    self._infer_path,
                    json={"inputs": text, "options": {"wait_for_model": True}},
                )
            except httpx.HTTPError as e:
                last_error = f"http error: {e}"
                continue

            if resp.status_code == 503 and not load_retried:
                load_retried = True
                wait = _DEFAULT_LOAD_BACKOFF
                try:
                    body = resp.json()
                    if isinstance(body, dict) and "estimated_time" in body:
                        wait = min(_DEFAULT_LOAD_BACKOFF, float(body["estimated_time"]))
                except Exception:  # noqa: BLE001
                    pass
                await asyncio.sleep(wait)
                continue

            if resp.status_code == 429 and rate_retries < _MAX_RETRIES:
                retry_after = float(resp.headers.get("Retry-After", 2 ** rate_retries))
                rate_retries += 1
                await asyncio.sleep(retry_after)
                continue

            if resp.status_code != 200:
                return SentimentResult(
                    label=None,
                    confidence=0.0,
                    error=f"HF status {resp.status_code}",
                )

            return self._parse_payload(resp.json())

        return SentimentResult(label=None, confidence=0.0, error=last_error or "max retries")

    @staticmethod
    def _parse_payload(payload: Any) -> SentimentResult:
        # Expected: [[{"label": "POSITIVE", "score": 0.98}, ...]]
        # Some endpoints return the inner list directly.
        try:
            inner = payload[0] if (isinstance(payload, list) and payload and isinstance(payload[0], list)) else payload
            if not isinstance(inner, list) or not inner:
                return SentimentResult(label=None, confidence=0.0, error="empty payload")
            top = max(inner, key=lambda x: float(x.get("score", 0)))
            return SentimentResult(
                label=label_map.from_hf(str(top["label"])),
                confidence=float(top["score"]),
            )
        except (KeyError, ValueError, TypeError) as e:
            return SentimentResult(label=None, confidence=0.0, error=f"parse error: {e}")

    async def close(self) -> None:
        await self._client.aclose()
