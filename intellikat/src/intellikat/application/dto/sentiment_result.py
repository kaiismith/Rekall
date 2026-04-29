"""DTO returned by `SentimentAnalyzer.analyze` for one segment.

Per-segment failure is encoded in `error` instead of an exception so a single
failed segment doesn't abandon the whole batch (Requirement 3.8).
"""

from __future__ import annotations

from pydantic import BaseModel, ConfigDict, Field

from intellikat.domain.value_objects.sentiment_label import SentimentLabel


class SentimentResult(BaseModel):
    model_config = ConfigDict(frozen=True, extra="forbid")

    label: SentimentLabel | None = None
    confidence: float = Field(default=0.0, ge=0.0, le=1.0)
    error: str | None = None
