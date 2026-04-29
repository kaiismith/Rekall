"""Unit tests for `ServiceBusJobConsumer._parse` and `_dispatch` outcomes."""

from __future__ import annotations

import json
from types import SimpleNamespace
from typing import Any
from uuid import uuid4

import pytest

from intellikat.domain.errors import JobFailedError
from intellikat.domain.value_objects.job_reference import JobReference
from intellikat.infrastructure.messaging.servicebus_consumer import (
    ServiceBusJobConsumer,
)


def _valid_payload() -> dict[str, Any]:
    return {
        "schema_version": "1",
        "job_id": str(uuid4()),
        "event_type": "transcript.session.closed",
        "transcript_session_id": str(uuid4()),
        "scope": {"kind": "call", "id": str(uuid4())},
        "segment_index_range": {"from": 0, "to": 5},
        "speaker_user_id": str(uuid4()),
        "engine_snapshot": {"engine_mode": "openai", "model_id": "whisper-1"},
        "occurred_at": "2026-04-29T08:42:11.420Z",
        "correlation_id": "cid",
    }


class _FakeMsg:
    def __init__(self, body: bytes, message_id: str = "msg-1") -> None:
        self.body = body
        self.message_id = message_id


def test_parse_valid_payload() -> None:
    body = json.dumps(_valid_payload()).encode("utf-8")
    msg = _FakeMsg(body)
    ref = ServiceBusJobConsumer._parse(msg)  # type: ignore[arg-type]
    assert isinstance(ref, JobReference)
    assert ref.event_type == "transcript.session.closed"


def test_parse_iterable_body() -> None:
    chunks = [json.dumps(_valid_payload()).encode("utf-8")]
    msg = _FakeMsg(iter(chunks))  # type: ignore[arg-type]
    ref = ServiceBusJobConsumer._parse(msg)  # type: ignore[arg-type]
    assert isinstance(ref, JobReference)


def test_parse_invalid_payload_raises() -> None:
    msg = _FakeMsg(b'{"schema_version": "999"}')
    with pytest.raises(Exception):
        ServiceBusJobConsumer._parse(msg)  # type: ignore[arg-type]


# ── dispatch outcome tests ──────────────────────────────────────────────────


class _FakeReceiver:
    def __init__(self) -> None:
        self.completed: list[Any] = []
        self.abandoned: list[Any] = []
        self.dead_lettered: list[tuple[Any, str]] = []

    async def complete_message(self, msg) -> None:  # type: ignore[no-untyped-def]
        self.completed.append(msg)

    async def abandon_message(self, msg) -> None:  # type: ignore[no-untyped-def]
        self.abandoned.append(msg)

    async def dead_letter_message(self, msg, reason: str = "", error_description: str = "") -> None:  # type: ignore[no-untyped-def]
        self.dead_lettered.append((msg, reason))


class _FakeAutoLockRenewer:
    def __init__(self, *args, **kwargs) -> None:  # type: ignore[no-untyped-def]
        pass

    def register(self, *args, **kwargs) -> None:  # type: ignore[no-untyped-def]
        pass

    async def close(self) -> None:
        return None


@pytest.fixture
def consumer(monkeypatch, logger):  # type: ignore[no-untyped-def]
    # Patch AutoLockRenewer + bypass __init__ network work
    import intellikat.infrastructure.messaging.servicebus_consumer as mod

    monkeypatch.setattr(mod, "AutoLockRenewer", _FakeAutoLockRenewer)
    obj = ServiceBusJobConsumer.__new__(ServiceBusJobConsumer)
    obj._settings = SimpleNamespace(servicebus_message_lock_renewal_seconds=240)  # type: ignore[attr-defined]
    obj._logger = logger  # type: ignore[attr-defined]
    import asyncio as _aio
    obj._sem = _aio.Semaphore(4)  # type: ignore[attr-defined]
    obj._inflight = set()  # type: ignore[attr-defined]
    return obj


@pytest.mark.asyncio
async def test_dispatch_happy_path_completes(consumer) -> None:  # type: ignore[no-untyped-def]
    msg = _FakeMsg(json.dumps(_valid_payload()).encode("utf-8"))
    receiver = _FakeReceiver()

    async def handler(_ref):  # type: ignore[no-untyped-def]
        return None

    await consumer._dispatch(receiver, msg, handler)
    assert len(receiver.completed) == 1
    assert receiver.abandoned == []
    assert receiver.dead_lettered == []


@pytest.mark.asyncio
async def test_dispatch_retryable_failure_abandons(consumer) -> None:  # type: ignore[no-untyped-def]
    msg = _FakeMsg(json.dumps(_valid_payload()).encode("utf-8"))
    receiver = _FakeReceiver()

    async def handler(_ref):  # type: ignore[no-untyped-def]
        raise JobFailedError(code="UPSTREAM_500", retryable=True)

    await consumer._dispatch(receiver, msg, handler)
    assert receiver.completed == []
    assert len(receiver.abandoned) == 1
    assert receiver.dead_lettered == []


@pytest.mark.asyncio
async def test_dispatch_non_retryable_failure_dead_letters(consumer) -> None:  # type: ignore[no-untyped-def]
    msg = _FakeMsg(json.dumps(_valid_payload()).encode("utf-8"))
    receiver = _FakeReceiver()

    async def handler(_ref):  # type: ignore[no-untyped-def]
        raise JobFailedError(code="SESSION_NOT_FOUND", retryable=False)

    await consumer._dispatch(receiver, msg, handler)
    assert receiver.completed == []
    assert receiver.abandoned == []
    assert len(receiver.dead_lettered) == 1
    assert receiver.dead_lettered[0][1] == "SESSION_NOT_FOUND"


@pytest.mark.asyncio
async def test_dispatch_unexpected_exception_abandons(consumer) -> None:  # type: ignore[no-untyped-def]
    msg = _FakeMsg(json.dumps(_valid_payload()).encode("utf-8"))
    receiver = _FakeReceiver()

    async def handler(_ref):  # type: ignore[no-untyped-def]
        raise RuntimeError("boom")

    await consumer._dispatch(receiver, msg, handler)
    assert receiver.completed == []
    assert len(receiver.abandoned) == 1


@pytest.mark.asyncio
async def test_dispatch_parse_failure_dead_letters(consumer) -> None:  # type: ignore[no-untyped-def]
    msg = _FakeMsg(b"not json")
    receiver = _FakeReceiver()

    async def handler(_ref):  # type: ignore[no-untyped-def]
        raise AssertionError("handler should not be called on parse failure")

    await consumer._dispatch(receiver, msg, handler)
    assert receiver.completed == []
    assert len(receiver.dead_lettered) == 1
    assert receiver.dead_lettered[0][1] == "JOB_REFERENCE_INVALID"
