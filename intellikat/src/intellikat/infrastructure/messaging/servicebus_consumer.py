"""Azure Service Bus consumer.

Pulls messages from `topic/subscription`, dispatches to the handler with
bounded concurrency, auto-renews message locks, and translates handler
outcomes into complete / abandon / dead-letter on the broker.
"""

from __future__ import annotations

import asyncio
import json
from typing import TYPE_CHECKING, Any

from azure.servicebus.aio import AutoLockRenewer, ServiceBusClient
from pydantic import ValidationError

from intellikat.domain.errors import JobFailedError
from intellikat.domain.value_objects.job_reference import JobReference
from intellikat.infrastructure.logging import catalog

if TYPE_CHECKING:
    from azure.servicebus.aio import ServiceBusReceiver
    from azure.servicebus.aio._servicebus_receiver_async import (
        ServiceBusReceivedMessage,
    )

    from intellikat.domain.ports.job_consumer import JobHandler
    from intellikat.infrastructure.config.settings import Settings
    from intellikat.infrastructure.logging.logger import ContextLogger


class ServiceBusJobConsumer:
    """Implements the `JobConsumer` port over Azure Service Bus."""

    def __init__(self, settings: Settings, logger: ContextLogger) -> None:
        self._settings = settings
        self._logger = logger
        self._client = self._build_client(settings)
        self._sem = asyncio.Semaphore(settings.servicebus_max_concurrent_messages)
        self._inflight: set[asyncio.Task[None]] = set()

    async def run(self, handler: JobHandler, stop_event: asyncio.Event) -> None:
        async with self._client:
            self._logger.info(
                catalog.SERVICEBUS_CONNECTED,
                topic=self._settings.servicebus_topic,
                subscription=self._settings.servicebus_subscription,
            )
            receiver = self._client.get_subscription_receiver(
                topic_name=self._settings.servicebus_topic,
                subscription_name=self._settings.servicebus_subscription,
                max_wait_time=5,
            )
            async with receiver:
                while not stop_event.is_set():
                    msgs = await receiver.receive_messages(
                        max_message_count=self._settings.servicebus_max_concurrent_messages,
                        max_wait_time=5,
                    )
                    for msg in msgs:
                        task = asyncio.create_task(self._dispatch(receiver, msg, handler))
                        self._inflight.add(task)
                        task.add_done_callback(self._inflight.discard)

                # Wait for in-flight tasks to drain (capped by caller's drain budget).
                if self._inflight:
                    await asyncio.gather(*self._inflight, return_exceptions=True)
            self._logger.info(catalog.SERVICEBUS_DISCONNECTED)

    async def close(self) -> None:
        if self._client is not None:
            await self._client.close()

    async def _dispatch(
        self,
        receiver: ServiceBusReceiver,
        msg: ServiceBusReceivedMessage,
        handler: JobHandler,
    ) -> None:
        async with self._sem:
            renewer = AutoLockRenewer(
                max_lock_renewal_duration=self._settings.servicebus_message_lock_renewal_seconds * 4,
            )
            renewer.register(receiver, msg)
            try:
                try:
                    ref = self._parse(msg)
                except (ValidationError, ValueError, json.JSONDecodeError) as e:
                    self._logger.error(
                        catalog.JOB_PARSE_FAILED,
                        message_id=msg.message_id,
                        err=str(e),
                    )
                    await receiver.dead_letter_message(
                        msg,
                        reason="JOB_REFERENCE_INVALID",
                        error_description=str(e)[:1000],
                    )
                    return

                self._logger.info(
                    catalog.JOB_RECEIVED,
                    message_id=msg.message_id,
                    transcript_session_id=str(ref.transcript_session_id),
                    correlation_id=ref.correlation_id,
                    event_type=ref.event_type,
                )

                try:
                    await handler(ref)
                    await receiver.complete_message(msg)
                except JobFailedError as e:
                    if e.retryable:
                        self._logger.warn(
                            catalog.JOB_ABANDONED,
                            message_id=msg.message_id,
                            code=e.code,
                            err=str(e),
                        )
                        await receiver.abandon_message(msg)
                    else:
                        self._logger.warn(
                            catalog.JOB_DEAD_LETTERED,
                            message_id=msg.message_id,
                            code=e.code,
                            err=str(e),
                        )
                        await receiver.dead_letter_message(
                            msg,
                            reason=e.code,
                            error_description=str(e)[:1000],
                        )
                except Exception as e:  # noqa: BLE001
                    self._logger.error(
                        catalog.JOB_ABANDONED,
                        message_id=msg.message_id,
                        code="UNEXPECTED",
                        err=str(e),
                    )
                    await receiver.abandon_message(msg)
            finally:
                try:
                    await renewer.close()
                except Exception:  # noqa: BLE001
                    self._logger.warn(catalog.LOCK_RENEW_FAILED, message_id=msg.message_id)

    @staticmethod
    def _parse(msg: ServiceBusReceivedMessage) -> JobReference:
        body = msg.body
        if hasattr(body, "__iter__") and not isinstance(body, (bytes, bytearray, str)):
            body = b"".join(body)  # type: ignore[arg-type]
        if isinstance(body, str):
            body = body.encode("utf-8")
        return JobReference.model_validate_json(body)  # type: ignore[arg-type]

    @staticmethod
    def _build_client(settings: Settings) -> Any:
        if settings.servicebus_fully_qualified_namespace:
            from azure.identity.aio import DefaultAzureCredential  # noqa: PLC0415

            return ServiceBusClient(
                fully_qualified_namespace=settings.servicebus_fully_qualified_namespace,
                credential=DefaultAzureCredential(),
            )
        if settings.servicebus_connection_string is None:
            raise RuntimeError("settings validation should have caught this")
        return ServiceBusClient.from_connection_string(
            settings.servicebus_connection_string.get_secret_value()
        )
