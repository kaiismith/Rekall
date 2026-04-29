"""Port: produce a structured summary of a stitched transcript.

Implemented by the Foundry adapter; future Anthropic/Bedrock/etc adapters
slot in without touching the use case.
"""

from __future__ import annotations

from typing import Protocol, runtime_checkable

from intellikat.application.dto.summary_result import SummaryResult
from intellikat.domain.value_objects.prompt_version import PromptVersion


@runtime_checkable
class Summarizer(Protocol):
    @property
    def model_id(self) -> str: ...

    async def health_check(self) -> bool: ...

    async def summarize(
        self, transcript: str, prompt_version: PromptVersion
    ) -> SummaryResult: ...
