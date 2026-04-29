"""Read-model mirror of the `transcript_segments` row.

Owned by transcript-persistence; intellikat reads only.
"""

from __future__ import annotations

from datetime import datetime
from uuid import UUID

from pydantic import BaseModel, ConfigDict, Field


class TranscriptSegment(BaseModel):
    model_config = ConfigDict(frozen=True, extra="ignore")

    id: UUID
    session_id: UUID
    segment_index: int = Field(ge=0)
    speaker_user_id: UUID
    text: str
    language: str | None
    confidence: float | None
    start_ms: int = Field(ge=0)
    end_ms: int = Field(gt=0)
    segment_started_at: datetime
