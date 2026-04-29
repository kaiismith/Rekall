"""Write-side ORM for `transcript_segment_sentiments` (owned by intellikat)."""

from __future__ import annotations

from datetime import UTC, datetime
from uuid import UUID, uuid4

from sqlalchemy import REAL, TIMESTAMP, Text
from sqlalchemy.dialects.postgresql import UUID as PGUUID
from sqlalchemy.orm import Mapped, mapped_column

from intellikat.domain.entities.sentiment import SegmentSentiment
from intellikat.infrastructure.persistence.orm.base import Base


class SegmentSentimentRow(Base):
    __tablename__ = "transcript_segment_sentiments"

    id: Mapped[UUID] = mapped_column(PGUUID(as_uuid=True), primary_key=True, default=uuid4)
    transcript_segment_id: Mapped[UUID] = mapped_column(PGUUID(as_uuid=True), nullable=False)
    transcript_session_id: Mapped[UUID] = mapped_column(PGUUID(as_uuid=True), nullable=False)
    run_id: Mapped[UUID] = mapped_column(PGUUID(as_uuid=True), nullable=False)
    label: Mapped[str] = mapped_column(Text, nullable=False)
    confidence: Mapped[float] = mapped_column(REAL, nullable=False)
    model_id: Mapped[str] = mapped_column(Text, nullable=False)
    model_revision: Mapped[str | None] = mapped_column(Text, nullable=True)
    hf_mode: Mapped[str] = mapped_column(Text, nullable=False)
    processed_at: Mapped[datetime] = mapped_column(
        TIMESTAMP(timezone=True), nullable=False, default=lambda: datetime.now(UTC)
    )

    @classmethod
    def from_entity(cls, e: SegmentSentiment) -> "SegmentSentimentRow":
        return cls(
            transcript_segment_id=e.transcript_segment_id,
            transcript_session_id=e.transcript_session_id,
            run_id=e.run_id,
            label=e.label.value,
            confidence=e.confidence,
            model_id=e.model_id,
            model_revision=e.model_revision,
            hf_mode=e.hf_mode,
            processed_at=e.processed_at,
        )
