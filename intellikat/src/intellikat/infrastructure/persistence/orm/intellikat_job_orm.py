"""Write-side ORM for `intellikat_jobs` (audit table; owned by intellikat).

Note: `transcript_session_id` intentionally has NO foreign key — the audit
row outlives the parent session if it's deleted, so operators can still
answer "was this ever processed?".
"""

from __future__ import annotations

from datetime import UTC, datetime
from uuid import UUID

from sqlalchemy import TIMESTAMP, Boolean, Integer, Text
from sqlalchemy.dialects.postgresql import UUID as PGUUID
from sqlalchemy.orm import Mapped, mapped_column

from intellikat.domain.entities.insight_job import InsightJob
from intellikat.infrastructure.persistence.orm.base import Base


class IntellikatJobRow(Base):
    __tablename__ = "intellikat_jobs"

    id: Mapped[UUID] = mapped_column(PGUUID(as_uuid=True), primary_key=True)
    message_id: Mapped[str | None] = mapped_column(Text, nullable=True)
    job_id: Mapped[UUID | None] = mapped_column(PGUUID(as_uuid=True), nullable=True)
    transcript_session_id: Mapped[UUID] = mapped_column(PGUUID(as_uuid=True), nullable=False)
    event_type: Mapped[str] = mapped_column(Text, nullable=False)
    schema_version: Mapped[str] = mapped_column(Text, nullable=False)
    correlation_id: Mapped[str | None] = mapped_column(Text, nullable=True)
    status: Mapped[str] = mapped_column(Text, nullable=False)
    segments_total: Mapped[int | None] = mapped_column(Integer, nullable=True)
    segments_processed: Mapped[int | None] = mapped_column(Integer, nullable=True)
    segments_failed: Mapped[int | None] = mapped_column(Integer, nullable=True)
    summary_persisted: Mapped[bool] = mapped_column(Boolean, nullable=False, default=False)
    error_code: Mapped[str | None] = mapped_column(Text, nullable=True)
    error_message: Mapped[str | None] = mapped_column(Text, nullable=True)
    hf_mode: Mapped[str] = mapped_column(Text, nullable=False)
    started_at: Mapped[datetime] = mapped_column(
        TIMESTAMP(timezone=True), nullable=False, default=lambda: datetime.now(UTC)
    )
    finished_at: Mapped[datetime | None] = mapped_column(TIMESTAMP(timezone=True), nullable=True)

    @classmethod
    def from_entity(cls, j: InsightJob, *, hf_mode: str) -> "IntellikatJobRow":
        return cls(
            id=j.run_id,
            message_id=j.message_id,
            job_id=j.reference.job_id,
            transcript_session_id=j.reference.transcript_session_id,
            event_type=j.reference.event_type,
            schema_version=j.reference.schema_version,
            correlation_id=j.reference.correlation_id,
            status=j.status,
            segments_total=j.segments_total,
            segments_processed=j.segments_processed,
            segments_failed=j.segments_failed,
            summary_persisted=j.summary_persisted,
            error_code=j.error_code,
            error_message=j.error_message,
            hf_mode=hf_mode,
            started_at=j.started_at,
            finished_at=j.finished_at,
        )
