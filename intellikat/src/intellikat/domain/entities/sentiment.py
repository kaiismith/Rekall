"""Write-side aggregate: one row in `transcript_segment_sentiments`."""

from __future__ import annotations

from datetime import datetime
from uuid import UUID

from pydantic import BaseModel, ConfigDict, Field

from intellikat.domain.value_objects.sentiment_label import SentimentLabel


class SegmentSentiment(BaseModel):
    model_config = ConfigDict(frozen=True, extra="forbid")

    transcript_segment_id: UUID
    transcript_session_id: UUID
    run_id: UUID
    label: SentimentLabel
    confidence: float = Field(ge=0.0, le=1.0)
    model_id: str
    model_revision: str | None
    hf_mode: str
    processed_at: datetime
