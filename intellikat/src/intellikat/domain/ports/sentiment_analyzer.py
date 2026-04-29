"""Port: classify each segment's text into a sentiment label + confidence.

Implemented by the hosted (HF Inference API) and local (transformers pipeline)
adapters. The use-case layer never knows which is in use.
"""

from __future__ import annotations

from typing import Protocol, Sequence, runtime_checkable

from intellikat.application.dto.sentiment_result import SentimentResult
from intellikat.domain.entities.transcript_segment import TranscriptSegment


@runtime_checkable
class SentimentAnalyzer(Protocol):
    @property
    def model_id(self) -> str: ...

    @property
    def model_revision(self) -> str | None: ...

    @property
    def hf_mode(self) -> str: ...

    async def warmup(self) -> None:
        """Optional one-shot prep at startup (load weights, fetch revision).

        Adapters that don't need it can leave this as a no-op.
        """
        ...

    async def health_check(self) -> bool: ...

    async def analyze(
        self, segments: Sequence[TranscriptSegment]
    ) -> Sequence[SentimentResult]:
        """Returns one SentimentResult per input segment, in the same order."""
        ...
