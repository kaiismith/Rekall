"""Prompt version — typed string `sum-vN`.

Persisted with each summary so a future prompt rewrite doesn't muddle history.
"""

from __future__ import annotations

import re
from typing import NewType

PromptVersion = NewType("PromptVersion", str)

_SHAPE = re.compile(r"^sum-v\d+$")


def make_prompt_version(value: str) -> PromptVersion:
    """Validate the shape; raises `ValueError` on bad input."""
    if not _SHAPE.match(value):
        raise ValueError(f"prompt_version must look like 'sum-vN' (got {value!r})")
    return PromptVersion(value)
