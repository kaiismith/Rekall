"""Write-side ORM for `transcript_session_summaries` (owned by intellikat)."""

from __future__ import annotations

from datetime import UTC, datetime
from typing import Any
from uuid import UUID, uuid4

from sqlalchemy import TIMESTAMP, Boolean, Integer, Text
from sqlalchemy.dialects.postgresql import JSONB
from sqlalchemy.dialects.postgresql import UUID as PGUUID
from sqlalchemy.orm import Mapped, mapped_column

from intellikat.domain.entities.summary import SessionSummary
from intellikat.infrastructure.persistence.orm.base import Base


class SessionSummaryRow(Base):
    __tablename__ = "transcript_session_summaries"

    id: Mapped[UUID] = mapped_column(PGUUID(as_uuid=True), primary_key=True, default=uuid4)
    transcript_session_id: Mapped[UUID] = mapped_column(PGUUID(as_uuid=True), nullable=False)
    run_id: Mapped[UUID] = mapped_column(PGUUID(as_uuid=True), nullable=False)
    content: Mapped[str] = mapped_column(Text, nullable=False)
    key_points: Mapped[Any | None] = mapped_column(JSONB, nullable=True)
    model_id: Mapped[str] = mapped_column(Text, nullable=False)
    prompt_version: Mapped[str] = mapped_column(Text, nullable=False)
    prompt_tokens: Mapped[int | None] = mapped_column(Integer, nullable=True)
    completion_tokens: Mapped[int | None] = mapped_column(Integer, nullable=True)
    input_truncated: Mapped[bool] = mapped_column(Boolean, nullable=False, default=False)
    processed_at: Mapped[datetime] = mapped_column(
        TIMESTAMP(timezone=True), nullable=False, default=lambda: datetime.now(UTC)
    )

    @classmethod
    def from_entity(cls, e: SessionSummary) -> "SessionSummaryRow":
        return cls(
            transcript_session_id=e.transcript_session_id,
            run_id=e.run_id,
            content=e.content,
            key_points=e.key_points,
            model_id=e.model_id,
            prompt_version=e.prompt_version,
            prompt_tokens=e.prompt_tokens,
            completion_tokens=e.completion_tokens,
            input_truncated=e.input_truncated,
            processed_at=e.processed_at,
        )
