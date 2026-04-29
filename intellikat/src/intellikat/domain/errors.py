"""Domain-level error hierarchy.

Errors thrown by the application layer that surface specific recovery
behaviour (retryable vs not) at the Service Bus boundary.
"""

from __future__ import annotations


class IntellikatError(Exception):
    """Root exception type for everything intellikat-domain."""


class ConfigInvalidError(IntellikatError):
    """Raised at startup when settings cross-field validation fails."""


class JobFailedError(IntellikatError):
    """Raised by use cases when a job cannot complete.

    `retryable=True`  → consumer abandons the message; broker re-delivers.
    `retryable=False` → consumer dead-letters the message immediately.
    """

    def __init__(self, *, code: str, retryable: bool, message: str | None = None) -> None:
        super().__init__(message or code)
        self.code = code
        self.retryable = retryable


class SummaryParseError(IntellikatError):
    """Raised when the LLM response can't be parsed as the expected JSON shape
    even after the corrective retry."""


class UpstreamError(IntellikatError):
    """Raised by adapters wrapping a transient upstream failure (HF, Foundry).

    Wrapped into `JobFailedError(retryable=True)` by use cases when appropriate.
    """