"""Redaction helpers for log records.

Drops `SecretStr` values and any field whose key matches a secret-like regex.
The regex catches common credential field names defensively; explicit
`SecretStr` typing in settings is the primary defence.
"""

from __future__ import annotations

import re
from typing import Any

from pydantic import SecretStr

DEFAULT_SECRET_REGEX = re.compile(
    r"(?i)(token|secret|password|api[_-]?key|connection[_-]?string|credential)"
)

REDACTED = "<redacted>"


class Redactor:
    """Stateless redactor; safe to share across log records."""

    def __init__(self, secret_regex: re.Pattern[str] = DEFAULT_SECRET_REGEX) -> None:
        self._regex = secret_regex

    def redact(self, fields: dict[str, Any]) -> dict[str, Any]:
        """Return a shallow copy of `fields` with secrets replaced by `<redacted>`."""
        out: dict[str, Any] = {}
        for key, value in fields.items():
            if self._is_secret_key(key) or self._is_secret_value(value):
                out[key] = REDACTED
            elif isinstance(value, dict):
                out[key] = self.redact(value)  # type: ignore[arg-type]
            else:
                out[key] = value
        return out

    def _is_secret_key(self, key: str) -> bool:
        return bool(self._regex.search(key))

    @staticmethod
    def _is_secret_value(value: Any) -> bool:
        return isinstance(value, SecretStr)
