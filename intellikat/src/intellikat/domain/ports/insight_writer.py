"""Port: transactional write of all insight results for one job.

One UoW per job: enter on job start, save_*, commit on success, rollback on
any exception. Prevents partial states like "summary persisted but no
sentiments" (Requirement 13.4).
"""

from __future__ import annotations

from types import TracebackType
from typing import Protocol, Sequence, runtime_checkable

from intellikat.domain.entities.insight_job import InsightJob
from intellikat.domain.entities.sentiment import SegmentSentiment
from intellikat.domain.entities.summary import SessionSummary


@runtime_checkable
class InsightWriter(Protocol):
    async def __aenter__(self) -> "InsightWriter": ...

    async def __aexit__(
        self,
        exc_type: type[BaseException] | None,
        exc: BaseException | None,
        tb: TracebackType | None,
    ) -> None: ...

    async def save_job(self, job: InsightJob) -> None: ...

    async def save_sentiments(
        self, sentiments: Sequence[SegmentSentiment]
    ) -> None: ...

    async def save_summary(self, summary: SessionSummary) -> None: ...
