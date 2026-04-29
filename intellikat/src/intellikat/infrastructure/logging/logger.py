"""Structured logging.

Wraps stdlib `logging` with a JSON formatter and a `ContextLogger` that
forces every emit to use an `Event` constant from the catalog. No metrics,
no APM, no OTel — logs only by user constraint (Requirement 12).
"""

from __future__ import annotations

import logging
import sys
from datetime import UTC, datetime
from typing import TYPE_CHECKING, Any

from intellikat.infrastructure.logging.redaction import Redactor

if TYPE_CHECKING:
    from intellikat.infrastructure.config.settings import Settings
    from intellikat.infrastructure.logging.catalog import Event


class JsonFormatter(logging.Formatter):
    """Emits one JSON object per log record."""

    def __init__(
        self,
        redactor: Redactor,
        static_fields: dict[str, Any] | None = None,
    ) -> None:
        super().__init__()
        self._redactor = redactor
        self._static = static_fields or {}

    def format(self, record: logging.LogRecord) -> str:
        import json

        fields: dict[str, Any] = {
            "timestamp": datetime.now(UTC).isoformat(timespec="milliseconds").replace("+00:00", "Z"),
            "level": record.levelname.lower(),
            **self._static,
            "logger": record.name,
            "message": record.getMessage(),
        }
        extras = getattr(record, "_intellikat_fields", None)
        if isinstance(extras, dict):
            fields.update(extras)
        if record.exc_info:
            fields["exc"] = self.formatException(record.exc_info)
        return json.dumps(self._redactor.redact(fields), default=str)


class ContextLogger:
    """Thin wrapper over a stdlib logger that requires an `Event` for every emit."""

    def __init__(self, logger: logging.Logger) -> None:
        self._logger = logger

    def debug(self, event: Event, **fields: Any) -> None:
        self._emit(logging.DEBUG, event, fields)

    def info(self, event: Event, **fields: Any) -> None:
        self._emit(logging.INFO, event, fields)

    def warn(self, event: Event, **fields: Any) -> None:
        self._emit(logging.WARNING, event, fields)

    def warning(self, event: Event, **fields: Any) -> None:
        self._emit(logging.WARNING, event, fields)

    def error(self, event: Event, exc_info: bool = False, **fields: Any) -> None:
        self._emit(logging.ERROR, event, fields, exc_info=exc_info)

    def _emit(
        self,
        level: int,
        event: Event,
        fields: dict[str, Any],
        exc_info: bool = False,
    ) -> None:
        if not self._logger.isEnabledFor(level):
            return
        merged = {"event_code": event.code, **fields}
        self._logger.log(
            level,
            event.message,
            extra={"_intellikat_fields": merged},
            exc_info=exc_info,
        )


def configure_logging(settings: Settings) -> ContextLogger:
    """Configure root logging once at startup; return the wrapped service logger."""
    handler = logging.StreamHandler(sys.stdout)
    if settings.log_format == "json":
        handler.setFormatter(
            JsonFormatter(
                redactor=Redactor(),
                static_fields={"service": "intellikat", "service_env": settings.service_env},
            )
        )
    else:
        handler.setFormatter(
            logging.Formatter("%(asctime)s %(levelname)s %(name)s %(message)s")
        )

    root = logging.getLogger()
    root.handlers = [handler]
    root.setLevel(settings.log_level)

    for noisy in ("sqlalchemy.engine", "azure", "httpx", "httpcore", "urllib3", "uamqp"):
        logging.getLogger(noisy).setLevel(logging.WARNING)

    return ContextLogger(logging.getLogger("intellikat"))
