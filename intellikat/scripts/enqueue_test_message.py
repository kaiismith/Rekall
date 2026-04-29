"""Dev helper: publish a synthetic `transcript.session.closed` message.

Usage:
    python -m intellikat.scripts.enqueue_test_message <transcript_session_id>

Reads INTELLIKAT_SERVICEBUS_* from the environment for connection info.
Useful for local end-to-end smoke tests of the consumer without going
through the real backend.
"""

from __future__ import annotations

import argparse
import asyncio
import json
import os
import sys
from datetime import UTC, datetime
from uuid import UUID, uuid4

from azure.servicebus import ServiceBusMessage
from azure.servicebus.aio import ServiceBusClient


def _build_payload(session_id: UUID) -> dict:
    return {
        "schema_version": "1",
        "job_id": str(uuid4()),
        "event_type": "transcript.session.closed",
        "transcript_session_id": str(session_id),
        "scope": {"kind": "call", "id": str(uuid4())},
        "segment_index_range": {"from": 0, "to": 99},
        "speaker_user_id": str(uuid4()),
        "engine_snapshot": {"engine_mode": "openai", "model_id": "whisper-1"},
        "occurred_at": datetime.now(UTC).isoformat(),
        "correlation_id": "dev-enqueue",
    }


async def main() -> int:
    p = argparse.ArgumentParser()
    p.add_argument("transcript_session_id", type=UUID)
    p.add_argument("--topic", default=os.getenv("INTELLIKAT_SERVICEBUS_TOPIC", "rekall.transcript.insights"))
    args = p.parse_args()

    conn = os.environ.get("INTELLIKAT_SERVICEBUS_CONNECTION_STRING")
    fqns = os.environ.get("INTELLIKAT_SERVICEBUS_FULLY_QUALIFIED_NAMESPACE")
    if not (conn or fqns):
        print("set INTELLIKAT_SERVICEBUS_CONNECTION_STRING or _FULLY_QUALIFIED_NAMESPACE", file=sys.stderr)
        return 1

    if conn:
        client = ServiceBusClient.from_connection_string(conn)
    else:
        from azure.identity.aio import DefaultAzureCredential

        client = ServiceBusClient(
            fully_qualified_namespace=fqns, credential=DefaultAzureCredential()
        )

    payload = _build_payload(args.transcript_session_id)
    body = json.dumps(payload).encode("utf-8")
    msg = ServiceBusMessage(
        body=body,
        content_type="application/json",
        message_id=payload["job_id"],
        correlation_id=payload["correlation_id"],
        application_properties={"event_type": payload["event_type"], "schema_version": "1"},
    )
    async with client:
        sender = client.get_topic_sender(topic_name=args.topic)
        async with sender:
            await sender.send_messages(msg)
    print(f"sent message {payload['job_id']} to topic {args.topic}")
    return 0


if __name__ == "__main__":
    sys.exit(asyncio.run(main()))
