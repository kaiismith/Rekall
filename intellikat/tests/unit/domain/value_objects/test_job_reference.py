"""Unit tests for `JobReference` parsing (Requirement 18.1)."""

from __future__ import annotations

import json
from typing import Any
from uuid import uuid4

import pytest
from pydantic import ValidationError

from intellikat.domain.value_objects.job_reference import JobReference


def _valid_payload(**overrides: Any) -> dict[str, Any]:
    payload: dict[str, Any] = {
        "schema_version": "1",
        "job_id": str(uuid4()),
        "event_type": "transcript.session.closed",
        "transcript_session_id": str(uuid4()),
        "scope": {"kind": "call", "id": str(uuid4())},
        "segment_index_range": {"from": 0, "to": 47},
        "speaker_user_id": str(uuid4()),
        "engine_snapshot": {"engine_mode": "openai", "model_id": "whisper-1"},
        "occurred_at": "2026-04-29T08:42:11.420Z",
        "correlation_id": "req-abc",
    }
    payload.update(overrides)
    return payload


def test_happy_path_parses() -> None:
    ref = JobReference.model_validate(_valid_payload())
    assert ref.event_type == "transcript.session.closed"
    assert ref.scope.kind == "call"
    assert ref.segment_index_range is not None
    assert ref.segment_index_range.from_ == 0


def test_meeting_scope_ok() -> None:
    payload = _valid_payload(scope={"kind": "meeting", "id": str(uuid4())})
    ref = JobReference.model_validate(payload)
    assert ref.scope.kind == "meeting"


def test_reprocess_event_type_ok() -> None:
    ref = JobReference.model_validate(
        _valid_payload(event_type="transcript.session.reprocess")
    )
    assert ref.event_type == "transcript.session.reprocess"


def test_unknown_event_type_rejected() -> None:
    with pytest.raises(ValidationError):
        JobReference.model_validate(_valid_payload(event_type="something.else"))


def test_unknown_schema_version_rejected() -> None:
    with pytest.raises(ValidationError) as exc:
        JobReference.model_validate(_valid_payload(schema_version="2"))
    assert "schema_version" in str(exc.value)


def test_bad_uuid_rejected() -> None:
    with pytest.raises(ValidationError):
        JobReference.model_validate(_valid_payload(transcript_session_id="not-a-uuid"))


def test_extra_fields_forbidden() -> None:
    payload = _valid_payload()
    payload["extra_field"] = "boom"
    with pytest.raises(ValidationError):
        JobReference.model_validate(payload)


def test_missing_required_field_rejected() -> None:
    payload = _valid_payload()
    del payload["transcript_session_id"]
    with pytest.raises(ValidationError):
        JobReference.model_validate(payload)


def test_segment_index_range_optional() -> None:
    payload = _valid_payload()
    del payload["segment_index_range"]
    ref = JobReference.model_validate(payload)
    assert ref.segment_index_range is None


def test_negative_segment_index_rejected() -> None:
    payload = _valid_payload(segment_index_range={"from": -1, "to": 5})
    with pytest.raises(ValidationError):
        JobReference.model_validate(payload)


def test_parse_from_json_bytes() -> None:
    body = json.dumps(_valid_payload()).encode("utf-8")
    ref = JobReference.model_validate_json(body)
    assert ref.schema_version == "1"
