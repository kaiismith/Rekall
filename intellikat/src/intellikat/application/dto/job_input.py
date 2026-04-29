"""Application-layer view of a job's inputs after the orchestrator
resolves the JobReference into the read-side DB rows."""

from __future__ import annotations

from uuid import UUID

from pydantic import BaseModel, ConfigDict


class JobInput(BaseModel):
    model_config = ConfigDict(frozen=True, extra="forbid")

    transcript_session_id: UUID
    correlation_id: str | None
    segment_index_from: int | None
    segment_index_to: int | None
