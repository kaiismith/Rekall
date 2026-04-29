"""Read-only ORM mapping for `transcript_segments` (owned by transcript-persistence)."""

from __future__ import annotations

from datetime import datetime
from uuid import UUID

from sqlalchemy import REAL, TIMESTAMP, Integer, Text
from sqlalchemy.dialects.postgresql import UUID as PGUUID
from sqlalchemy.orm import Mapped, mapped_column

from intellikat.infrastructure.persistence.orm.base import Base


class TranscriptSegmentRow(Base):
    __tablename__ = "transcript_segments"

    id: Mapped[UUID] = mapped_column(PGUUID(as_uuid=True), primary_key=True)
    session_id: Mapped[UUID] = mapped_column(PGUUID(as_uuid=True), nullable=False)
    segment_index: Mapped[int] = mapped_column(Integer, nullable=False)
    speaker_user_id: Mapped[UUID] = mapped_column(PGUUID(as_uuid=True), nullable=False)
    text: Mapped[str] = mapped_column(Text, nullable=False)
    language: Mapped[str | None] = mapped_column(Text, nullable=True)
    confidence: Mapped[float | None] = mapped_column(REAL, nullable=True)
    start_ms: Mapped[int] = mapped_column(Integer, nullable=False)
    end_ms: Mapped[int] = mapped_column(Integer, nullable=False)
    segment_started_at: Mapped[datetime] = mapped_column(TIMESTAMP(timezone=True), nullable=False)
