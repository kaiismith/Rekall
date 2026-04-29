"""Local HF transformers pipeline adapter.

Loaded only when `HF_MODE=local` AND the `local` extras are installed:
`uv sync --extra local`. The `transformers` import is deferred to the
constructor so the hosted-mode image doesn't pull torch.
"""

from __future__ import annotations

import asyncio
from pathlib import Path
from typing import TYPE_CHECKING, Any, Sequence

from intellikat.application.dto.sentiment_result import SentimentResult
from intellikat.infrastructure.sentiment import label_map

if TYPE_CHECKING:
    from intellikat.domain.entities.transcript_segment import TranscriptSegment


class HFLocalPipeline:
    """Implements the `SentimentAnalyzer` port via in-process inference."""

    hf_mode = "local"

    def __init__(
        self,
        *,
        model_id: str,
        cache_dir: Path,
        device: str,
        batch_size: int,
        max_input_tokens: int,
    ) -> None:
        try:
            from transformers import pipeline  # noqa: PLC0415
        except ImportError as e:  # pragma: no cover — covered by factory test
            raise ImportError(
                "intellikat: HF_MODE=local requires `pip install intellikat[local]`"
            ) from e

        actual_device = self._resolve_device(device)
        cache_dir.mkdir(parents=True, exist_ok=True)

        self._pipe: Any = pipeline(
            task="sentiment-analysis",
            model=model_id,
            tokenizer=model_id,
            device=actual_device,
            return_all_scores=False,
            model_kwargs={"cache_dir": str(cache_dir)},
        )
        self._model_id = model_id
        self._batch_size = batch_size
        self._max_input_tokens = max_input_tokens
        self._revision: str | None = getattr(
            getattr(self._pipe, "model", None).config if getattr(self._pipe, "model", None) else None,
            "_commit_hash",
            None,
        )

    @property
    def model_id(self) -> str:
        return self._model_id

    @property
    def model_revision(self) -> str | None:
        return self._revision

    async def warmup(self) -> None:
        # Model already loaded in __init__; one tiny inference to JIT any lazy state.
        await asyncio.to_thread(self._pipe, "ok")

    async def health_check(self) -> bool:
        return self._pipe is not None

    async def analyze(self, segments: Sequence[TranscriptSegment]) -> Sequence[SentimentResult]:
        return await asyncio.to_thread(self._infer_batched, list(segments))

    def _infer_batched(self, segments: list[TranscriptSegment]) -> list[SentimentResult]:
        if not segments:
            return []
        texts = [s.text for s in segments]
        out = self._pipe(
            texts,
            batch_size=self._batch_size,
            truncation=True,
            max_length=self._max_input_tokens,
        )
        results: list[SentimentResult] = []
        for entry in out:
            try:
                results.append(
                    SentimentResult(
                        label=label_map.from_hf(str(entry["label"])),
                        confidence=float(entry["score"]),
                    )
                )
            except (KeyError, ValueError, TypeError) as e:
                results.append(
                    SentimentResult(label=None, confidence=0.0, error=f"parse error: {e}")
                )
        return results

    @staticmethod
    def _resolve_device(setting: str) -> int | str:
        if setting != "auto":
            return setting
        try:
            import torch  # noqa: PLC0415

            if torch.cuda.is_available():
                return 0  # transformers convention: int = CUDA device index
            if getattr(torch.backends, "mps", None) and torch.backends.mps.is_available():
                return "mps"
        except ImportError:
            pass
        return "cpu"
