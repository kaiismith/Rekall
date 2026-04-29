"""Pydantic shape of the LLM's expected JSON output.

Validated by the summarizer adapter; on mismatch the adapter retries once
with a corrective system message, then raises `SummaryParseError`.
"""

from __future__ import annotations

from pydantic import BaseModel, ConfigDict, Field


class SummaryResponseModel(BaseModel):
    model_config = ConfigDict(extra="ignore")

    summary: str = Field(min_length=1)
    key_points: list[str] = Field(default_factory=list)
