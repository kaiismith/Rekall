"""Audit aggregate written to `intellikat_jobs`.

Lifecycle: running → (completed | failed | abandoned).

Mutated by the orchestrator as the job progresses; persisted at completion.
"""

from __future__ import annotations

from datetime import datetime
from typing import Literal
from uuid import UUID

from pydantic import BaseModel, ConfigDict

from intellikat.domain.value_objects.job_reference import JobReference

InsightJobStatus = Literal["running", "completed", "failed", "abandoned"]


class InsightJob(BaseModel):
    """Mutable — the orchestrator updates counters / status as it runs."""

    model_config = ConfigDict(frozen=False, extra="forbid")

    run_id: UUID
    reference: JobReference
    message_id: str | None = None
    status: InsightJobStatus = "running"
    segments_total: int = 0
    segments_processed: int = 0
    segments_failed: int = 0
    summary_persisted: bool = False
    error_code: str | None = None
    error_message: str | None = None
    started_at: datetime
    finished_at: datetime | None = None
