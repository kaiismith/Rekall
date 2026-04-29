"""Port: long-running message consumer.

Service Bus is the v1 implementation; an in-memory consumer is used in tests.
The consumer owns message lifecycle (complete / abandon / dead-letter); the
handler signals outcome by returning normally or raising `JobFailedError`.
"""

from __future__ import annotations

import asyncio
from typing import Awaitable, Callable, Protocol, runtime_checkable

from intellikat.domain.value_objects.job_reference import JobReference

JobHandler = Callable[[JobReference], Awaitable[None]]


@runtime_checkable
class JobConsumer(Protocol):
    async def run(self, handler: JobHandler, stop_event: asyncio.Event) -> None: ...

    async def close(self) -> None: ...
