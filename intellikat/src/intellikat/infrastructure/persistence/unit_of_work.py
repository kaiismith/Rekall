"""Unit-of-work — one transaction per job.

Implements the `InsightWriter` port. Sync SQLAlchemy work runs in a thread
so the asyncio orchestrator stays unblocked.
"""

from __future__ import annotations

import asyncio
from types import TracebackType
from typing import TYPE_CHECKING, Sequence

from sqlalchemy.dialects.postgresql import insert as pg_insert

from intellikat.infrastructure.persistence.orm.intellikat_job_orm import IntellikatJobRow
from intellikat.infrastructure.persistence.orm.segment_sentiment_orm import (
    SegmentSentimentRow,
)
from intellikat.infrastructure.persistence.orm.session_summary_orm import SessionSummaryRow

if TYPE_CHECKING:
    from sqlalchemy.orm import Session, sessionmaker

    from intellikat.domain.entities.insight_job import InsightJob
    from intellikat.domain.entities.sentiment import SegmentSentiment
    from intellikat.domain.entities.summary import SessionSummary


class SqlAlchemyUnitOfWork:
    """Implements the `InsightWriter` port.

    `hf_mode` is captured here (rather than on each entity) because it's a
    process-wide property of the analyzer adapter, not a per-job decision.
    """

    def __init__(self, session_factory: sessionmaker[Session], hf_mode: str) -> None:
        self._sf = session_factory
        self._hf_mode = hf_mode
        self._session: Session | None = None

    async def __aenter__(self) -> "SqlAlchemyUnitOfWork":
        self._session = await asyncio.to_thread(self._sf)
        return self

    async def __aexit__(
        self,
        exc_type: type[BaseException] | None,
        exc: BaseException | None,
        tb: TracebackType | None,
    ) -> None:
        sess = self._session
        if sess is None:
            return
        try:
            if exc_type is None:
                await asyncio.to_thread(sess.commit)
            else:
                await asyncio.to_thread(sess.rollback)
        finally:
            await asyncio.to_thread(sess.close)
            self._session = None

    async def save_sentiments(self, sentiments: Sequence[SegmentSentiment]) -> None:
        if not sentiments:
            return
        sess = self._require_session()
        rows = [SegmentSentimentRow.from_entity(s) for s in sentiments]
        await asyncio.to_thread(sess.add_all, rows)
        await asyncio.to_thread(sess.flush)

    async def save_summary(self, summary: SessionSummary) -> None:
        sess = self._require_session()
        row = SessionSummaryRow.from_entity(summary)
        await asyncio.to_thread(sess.add, row)
        await asyncio.to_thread(sess.flush)

    async def save_job(self, job: InsightJob) -> None:
        sess = self._require_session()
        row = IntellikatJobRow.from_entity(job, hf_mode=self._hf_mode)

        # UPSERT — the orchestrator may write the same audit row twice
        # (once on the happy path inside the UoW, once via the best-effort
        # audit on failure).
        def _upsert() -> None:
            stmt = pg_insert(IntellikatJobRow).values(
                {
                    "id": row.id,
                    "message_id": row.message_id,
                    "job_id": row.job_id,
                    "transcript_session_id": row.transcript_session_id,
                    "event_type": row.event_type,
                    "schema_version": row.schema_version,
                    "correlation_id": row.correlation_id,
                    "status": row.status,
                    "segments_total": row.segments_total,
                    "segments_processed": row.segments_processed,
                    "segments_failed": row.segments_failed,
                    "summary_persisted": row.summary_persisted,
                    "error_code": row.error_code,
                    "error_message": row.error_message,
                    "hf_mode": row.hf_mode,
                    "started_at": row.started_at,
                    "finished_at": row.finished_at,
                }
            )
            stmt = stmt.on_conflict_do_update(
                index_elements=["id"],
                set_={
                    "status": stmt.excluded.status,
                    "segments_total": stmt.excluded.segments_total,
                    "segments_processed": stmt.excluded.segments_processed,
                    "segments_failed": stmt.excluded.segments_failed,
                    "summary_persisted": stmt.excluded.summary_persisted,
                    "error_code": stmt.excluded.error_code,
                    "error_message": stmt.excluded.error_message,
                    "finished_at": stmt.excluded.finished_at,
                },
            )
            sess.execute(stmt)
            sess.flush()

        await asyncio.to_thread(_upsert)

    def _require_session(self) -> Session:
        if self._session is None:
            raise RuntimeError("UnitOfWork used outside `async with`")
        return self._session
