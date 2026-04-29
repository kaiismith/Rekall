"""Write-side aggregate: one row in `transcript_session_summaries`."""

from __future__ import annotations

from datetime import datetime
from uuid import UUID

from pydantic import BaseModel, ConfigDict


class SessionSummary(BaseModel):
    model_config = ConfigDict(frozen=True, extra="forbid")

    transcript_session_id: UUID
    run_id: UUID
    content: str
    key_points: list[str] | None
    model_id: str
    prompt_version: str
    prompt_tokens: int | None
    completion_tokens: int | None
    input_truncated: bool
    processed_at: datetime
