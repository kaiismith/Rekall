"""Engine snapshot — mirrors `(engine_mode, model_id)` from transcript-persistence."""

from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, ConfigDict

EngineMode = Literal["local", "openai", "legacy"]


class EngineSnapshot(BaseModel):
    model_config = ConfigDict(frozen=True, extra="forbid")

    engine_mode: EngineMode
    model_id: str
