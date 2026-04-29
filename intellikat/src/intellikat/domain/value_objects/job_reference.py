"""Service Bus message body — parsed.

The wire format is documented in [requirements.md §2.1] and [design.md].
The message carries IDENTIFIERS ONLY; never transcript text.
"""

from __future__ import annotations

from datetime import datetime
from typing import Literal
from uuid import UUID

from pydantic import BaseModel, ConfigDict, Field, field_validator

from intellikat.domain.value_objects.engine_snapshot import EngineSnapshot

EventType = Literal["transcript.session.closed", "transcript.session.reprocess"]
ScopeKind = Literal["call", "meeting"]

SCHEMA_VERSION = "1"


class Scope(BaseModel):
    model_config = ConfigDict(frozen=True, extra="forbid")

    kind: ScopeKind
    id: UUID


class SegmentIndexRange(BaseModel):
    """Inclusive `from_`, exclusive `to` (or both inclusive — consumer reads them all)."""

    model_config = ConfigDict(frozen=True, extra="forbid", populate_by_name=True)

    from_: int = Field(alias="from", ge=0)
    to: int = Field(ge=0)


class JobReference(BaseModel):
    model_config = ConfigDict(frozen=True, extra="forbid", populate_by_name=True)

    schema_version: str
    job_id: UUID
    event_type: EventType
    transcript_session_id: UUID
    scope: Scope
    segment_index_range: SegmentIndexRange | None = None
    speaker_user_id: UUID
    engine_snapshot: EngineSnapshot
    occurred_at: datetime
    correlation_id: str | None = None

    @field_validator("schema_version")
    @classmethod
    def _check_schema_version(cls, v: str) -> str:
        if v != SCHEMA_VERSION:
            raise ValueError(
                f"unsupported schema_version: {v!r} (expected {SCHEMA_VERSION!r})"
            )
        return v
