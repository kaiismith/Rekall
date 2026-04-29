"""Admin helper: publish a `transcript.session.reprocess` message.

Usage:
    python -m intellikat.scripts.enqueue_reprocess <transcript_session_id>

Forces intellikat to re-derive sentiment + summary for the named session
(creates a new run_id; the old rows stay for provenance).
"""

from __future__ import annotations

import asyncio
import json
import os
import sys
from datetime import UTC, datetime
from uuid import UUID, uuid4

from azure.servicebus import ServiceBusMessage
from azure.servicebus.aio import ServiceBusClient


async def main(session_id: UUID) -> int:
    conn = os.environ.get("INTELLIKAT_SERVICEBUS_CONNECTION_STRING")
    fqns = os.environ.get("INTELLIKAT_SERVICEBUS_FULLY_QUALIFIED_NAMESPACE")
    topic = os.environ.get("INTELLIKAT_SERVICEBUS_TOPIC", "rekall.transcript.insights")
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

    payload = {
        "schema_version": "1",
        "job_id": str(uuid4()),
        "event_type": "transcript.session.reprocess",
        "transcript_session_id": str(session_id),
        "scope": {"kind": "call", "id": str(uuid4())},
        "speaker_user_id": str(uuid4()),
        "engine_snapshot": {"engine_mode": "openai", "model_id": "whisper-1"},
        "occurred_at": datetime.now(UTC).isoformat(),
        "correlation_id": "admin-reprocess",
    }
    msg = ServiceBusMessage(
        body=json.dumps(payload).encode("utf-8"),
        content_type="application/json",
        message_id=payload["job_id"],
        correlation_id=payload["correlation_id"],
        application_properties={"event_type": payload["event_type"], "schema_version": "1"},
    )
    async with client:
        sender = client.get_topic_sender(topic_name=topic)
        async with sender:
            await sender.send_messages(msg)
    print(f"queued reprocess for {session_id} (job_id={payload['job_id']})")
    return 0


if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("usage: enqueue_reprocess.py <transcript_session_id>", file=sys.stderr)
        sys.exit(2)
    sys.exit(asyncio.run(main(UUID(sys.argv[1]))))
