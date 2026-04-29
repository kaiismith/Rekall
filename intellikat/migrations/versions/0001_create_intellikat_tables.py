"""create intellikat tables

Revision ID: 0001
Revises:
Create Date: 2026-04-29

"""

from __future__ import annotations

from typing import Sequence

import sqlalchemy as sa
from alembic import op
from sqlalchemy.dialects.postgresql import JSONB
from sqlalchemy.dialects.postgresql import UUID as PGUUID

revision: str = "0001"
down_revision: str | None = None
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


def upgrade() -> None:
    op.create_table(
        "transcript_segment_sentiments",
        sa.Column("id", PGUUID(as_uuid=True), primary_key=True, server_default=sa.text("gen_random_uuid()")),
        sa.Column(
            "transcript_segment_id",
            PGUUID(as_uuid=True),
            sa.ForeignKey("transcript_segments.id", ondelete="CASCADE"),
            nullable=False,
        ),
        sa.Column(
            "transcript_session_id",
            PGUUID(as_uuid=True),
            sa.ForeignKey("transcript_sessions.id", ondelete="CASCADE"),
            nullable=False,
        ),
        sa.Column("run_id", PGUUID(as_uuid=True), nullable=False),
        sa.Column("label", sa.Text(), nullable=False),
        sa.Column("confidence", sa.REAL(), nullable=False),
        sa.Column("model_id", sa.Text(), nullable=False),
        sa.Column("model_revision", sa.Text(), nullable=True),
        sa.Column("hf_mode", sa.Text(), nullable=False),
        sa.Column(
            "processed_at",
            sa.TIMESTAMP(timezone=True),
            nullable=False,
            server_default=sa.text("NOW()"),
        ),
        sa.CheckConstraint(
            "label IN ('POSITIVE','NEUTRAL','NEGATIVE')",
            name="transcript_segment_sentiments_label_chk",
        ),
        sa.CheckConstraint(
            "confidence >= 0 AND confidence <= 1",
            name="transcript_segment_sentiments_confidence_chk",
        ),
        sa.UniqueConstraint(
            "transcript_segment_id",
            "model_id",
            "model_revision",
            "run_id",
            name="transcript_segment_sentiments_unique_per_run",
        ),
    )
    op.create_index(
        "idx_segment_sentiments_session",
        "transcript_segment_sentiments",
        ["transcript_session_id"],
    )
    op.create_index(
        "idx_segment_sentiments_segment",
        "transcript_segment_sentiments",
        ["transcript_segment_id"],
    )
    op.create_index(
        "idx_segment_sentiments_label", "transcript_segment_sentiments", ["label"]
    )
    op.create_index(
        "idx_segment_sentiments_run", "transcript_segment_sentiments", ["run_id"]
    )

    op.create_table(
        "transcript_session_summaries",
        sa.Column("id", PGUUID(as_uuid=True), primary_key=True, server_default=sa.text("gen_random_uuid()")),
        sa.Column(
            "transcript_session_id",
            PGUUID(as_uuid=True),
            sa.ForeignKey("transcript_sessions.id", ondelete="CASCADE"),
            nullable=False,
        ),
        sa.Column("run_id", PGUUID(as_uuid=True), nullable=False),
        sa.Column("content", sa.Text(), nullable=False),
        sa.Column("key_points", JSONB(), nullable=True),
        sa.Column("model_id", sa.Text(), nullable=False),
        sa.Column("prompt_version", sa.Text(), nullable=False),
        sa.Column("prompt_tokens", sa.Integer(), nullable=True),
        sa.Column("completion_tokens", sa.Integer(), nullable=True),
        sa.Column(
            "input_truncated",
            sa.Boolean(),
            nullable=False,
            server_default=sa.text("FALSE"),
        ),
        sa.Column(
            "processed_at",
            sa.TIMESTAMP(timezone=True),
            nullable=False,
            server_default=sa.text("NOW()"),
        ),
        sa.UniqueConstraint(
            "transcript_session_id",
            "prompt_version",
            "run_id",
            name="transcript_session_summaries_unique_per_run",
        ),
    )
    op.create_index(
        "idx_session_summaries_session",
        "transcript_session_summaries",
        ["transcript_session_id"],
    )
    op.create_index(
        "idx_session_summaries_model", "transcript_session_summaries", ["model_id"]
    )
    op.create_index(
        "idx_session_summaries_run", "transcript_session_summaries", ["run_id"]
    )

    op.create_table(
        "intellikat_jobs",
        sa.Column("id", PGUUID(as_uuid=True), primary_key=True),
        sa.Column("message_id", sa.Text(), nullable=True),
        sa.Column("job_id", PGUUID(as_uuid=True), nullable=True),
        sa.Column("transcript_session_id", PGUUID(as_uuid=True), nullable=False),
        sa.Column("event_type", sa.Text(), nullable=False),
        sa.Column("schema_version", sa.Text(), nullable=False),
        sa.Column("correlation_id", sa.Text(), nullable=True),
        sa.Column("status", sa.Text(), nullable=False),
        sa.Column("segments_total", sa.Integer(), nullable=True),
        sa.Column("segments_processed", sa.Integer(), nullable=True),
        sa.Column("segments_failed", sa.Integer(), nullable=True),
        sa.Column(
            "summary_persisted",
            sa.Boolean(),
            nullable=False,
            server_default=sa.text("FALSE"),
        ),
        sa.Column("error_code", sa.Text(), nullable=True),
        sa.Column("error_message", sa.Text(), nullable=True),
        sa.Column("hf_mode", sa.Text(), nullable=False),
        sa.Column(
            "started_at",
            sa.TIMESTAMP(timezone=True),
            nullable=False,
            server_default=sa.text("NOW()"),
        ),
        sa.Column("finished_at", sa.TIMESTAMP(timezone=True), nullable=True),
        sa.CheckConstraint(
            "status IN ('running','completed','failed','abandoned')",
            name="intellikat_jobs_status_chk",
        ),
    )
    op.create_index(
        "idx_intellikat_jobs_session",
        "intellikat_jobs",
        ["transcript_session_id", sa.text("started_at DESC")],
    )
    op.create_index(
        "idx_intellikat_jobs_status",
        "intellikat_jobs",
        ["status", sa.text("started_at DESC")],
    )
    op.create_index(
        "idx_intellikat_jobs_started", "intellikat_jobs", [sa.text("started_at DESC")]
    )


def downgrade() -> None:
    op.drop_index("idx_intellikat_jobs_started", table_name="intellikat_jobs")
    op.drop_index("idx_intellikat_jobs_status", table_name="intellikat_jobs")
    op.drop_index("idx_intellikat_jobs_session", table_name="intellikat_jobs")
    op.drop_table("intellikat_jobs")

    op.drop_index("idx_session_summaries_run", table_name="transcript_session_summaries")
    op.drop_index("idx_session_summaries_model", table_name="transcript_session_summaries")
    op.drop_index("idx_session_summaries_session", table_name="transcript_session_summaries")
    op.drop_table("transcript_session_summaries")

    op.drop_index("idx_segment_sentiments_run", table_name="transcript_segment_sentiments")
    op.drop_index("idx_segment_sentiments_label", table_name="transcript_segment_sentiments")
    op.drop_index("idx_segment_sentiments_segment", table_name="transcript_segment_sentiments")
    op.drop_index("idx_segment_sentiments_session", table_name="transcript_segment_sentiments")
    op.drop_table("transcript_segment_sentiments")
