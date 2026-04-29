"""Port: read-only access to the shared transcript tables."""

from __future__ import annotations

from typing import Protocol, runtime_checkable
from uuid import UUID

from intellikat.domain.entities.transcript_segment import TranscriptSegment
from intellikat.domain.entities.transcript_session import TranscriptSession


@runtime_checkable
class TranscriptReader(Protocol):
    async def get_session(self, session_id: UUID) -> TranscriptSession | None: ...

    async def list_segments(
        self,
        session_id: UUID,
        segment_index_from: int | None = None,
        segment_index_to: int | None = None,
    ) -> list[TranscriptSegment]: ...
