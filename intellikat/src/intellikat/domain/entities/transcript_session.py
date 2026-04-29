"""Read-model mirror of the `transcript_sessions` row.

Owned by transcript-persistence; intellikat NEVER mutates this. No behaviour,
just shape — entities here serve as the type-safe boundary between the read
side of the DB and the application layer.
"""

from __future__ import annotations

from datetime import datetime
from typing import Literal
from uuid import UUID

from pydantic import BaseModel, ConfigDict

TranscriptSessionStatus = Literal["active", "ended", "errored", "expired"]


class TranscriptSession(BaseModel):
    model_config = ConfigDict(frozen=True, extra="ignore")

    id: UUID
    speaker_user_id: UUID
    call_id: UUID | None
    meeting_id: UUID | None
    engine_mode: str
    model_id: str
    language_requested: str | None
    status: TranscriptSessionStatus
    started_at: datetime
    ended_at: datetime | None
    expires_at: datetime
    correlation_id: str | None
