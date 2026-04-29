"""Postgres implementation of the `TranscriptReader` port.

Read-only queries against the shared `transcript_sessions` /
`transcript_segments` tables (owned by transcript-persistence).
"""

from __future__ import annotations

import asyncio
from typing import TYPE_CHECKING
from uuid import UUID

from sqlalchemy import select

from intellikat.domain.entities.transcript_segment import TranscriptSegment
from intellikat.domain.entities.transcript_session import TranscriptSession
from intellikat.infrastructure.persistence.orm.transcript_segment_orm import (
    TranscriptSegmentRow,
)
from intellikat.infrastructure.persistence.orm.transcript_session_orm import (
    TranscriptSessionRow,
)

if TYPE_CHECKING:
    from sqlalchemy.orm import Session, sessionmaker


class TranscriptReaderPg:
    def __init__(self, session_factory: sessionmaker[Session]) -> None:
        self._sf = session_factory

    async def get_session(self, session_id: UUID) -> TranscriptSession | None:
        return await asyncio.to_thread(self._get_session_sync, session_id)

    async def list_segments(
        self,
        session_id: UUID,
        segment_index_from: int | None = None,
        segment_index_to: int | None = None,
    ) -> list[TranscriptSegment]:
        return await asyncio.to_thread(
            self._list_segments_sync, session_id, segment_index_from, segment_index_to
        )

    def _get_session_sync(self, session_id: UUID) -> TranscriptSession | None:
        with self._sf() as sess:
            row = sess.execute(
                select(TranscriptSessionRow).where(TranscriptSessionRow.id == session_id)
            ).scalar_one_or_none()
            if row is None:
                return None
            return TranscriptSession.model_validate(row, from_attributes=True)

    def _list_segments_sync(
        self,
        session_id: UUID,
        idx_from: int | None,
        idx_to: int | None,
    ) -> list[TranscriptSegment]:
        with self._sf() as sess:
            stmt = (
                select(TranscriptSegmentRow)
                .where(TranscriptSegmentRow.session_id == session_id)
                .order_by(
                    TranscriptSegmentRow.segment_started_at.asc(),
                    TranscriptSegmentRow.segment_index.asc(),
                )
            )
            if idx_from is not None:
                stmt = stmt.where(TranscriptSegmentRow.segment_index >= idx_from)
            if idx_to is not None:
                stmt = stmt.where(TranscriptSegmentRow.segment_index <= idx_to)
            rows = sess.execute(stmt).scalars().all()
            return [
                TranscriptSegment.model_validate(r, from_attributes=True) for r in rows
            ]
