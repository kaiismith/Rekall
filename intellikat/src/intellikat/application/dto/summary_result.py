"""DTO returned by `Summarizer.summarize`."""

from __future__ import annotations

from pydantic import BaseModel, ConfigDict


class SummaryResult(BaseModel):
    model_config = ConfigDict(frozen=True, extra="forbid")

    content: str
    key_points: list[str] | None
    prompt_tokens: int | None
    completion_tokens: int | None
