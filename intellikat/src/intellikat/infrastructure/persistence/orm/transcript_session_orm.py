"""Read-only ORM mapping for `transcript_sessions` (owned by transcript-persistence).

Intellikat NEVER writes to this table. The mapping exists so we can SELECT
from it via SQLAlchemy.
"""

from __future__ import annotations

from datetime import datetime
from uuid import UUID

from sqlalchemy import TIMESTAMP, Text
from sqlalchemy.dialects.postgresql import UUID as PGUUID
from sqlalchemy.orm import Mapped, mapped_column

from intellikat.infrastructure.persistence.orm.base import Base


class TranscriptSessionRow(Base):
    __tablename__ = "transcript_sessions"

    id: Mapped[UUID] = mapped_column(PGUUID(as_uuid=True), primary_key=True)
    speaker_user_id: Mapped[UUID] = mapped_column(PGUUID(as_uuid=True), nullable=False)
    call_id: Mapped[UUID | None] = mapped_column(PGUUID(as_uuid=True), nullable=True)
    meeting_id: Mapped[UUID | None] = mapped_column(PGUUID(as_uuid=True), nullable=True)
    engine_mode: Mapped[str] = mapped_column(Text, nullable=False)
    model_id: Mapped[str] = mapped_column(Text, nullable=False)
    language_requested: Mapped[str | None] = mapped_column(Text, nullable=True)
    status: Mapped[str] = mapped_column(Text, nullable=False)
    started_at: Mapped[datetime] = mapped_column(TIMESTAMP(timezone=True), nullable=False)
    ended_at: Mapped[datetime | None] = mapped_column(TIMESTAMP(timezone=True), nullable=True)
    expires_at: Mapped[datetime] = mapped_column(TIMESTAMP(timezone=True), nullable=False)
    correlation_id: Mapped[str | None] = mapped_column(Text, nullable=True)
