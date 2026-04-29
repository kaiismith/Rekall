"""Loads the summarization prompt template from disk.

One file per `prompt_version` (e.g. `summarization_v1.txt`) so editing the
prompt is a deploy event with auditable provenance.
"""

from __future__ import annotations

from pathlib import Path

from intellikat.domain.errors import ConfigInvalidError


class PromptLoader:
    def __init__(self, prompt_dir: Path) -> None:
        self._dir = prompt_dir

    def load(self, prompt_version: str) -> str:
        version_num = prompt_version.removeprefix("sum-v")
        if not version_num:
            raise ConfigInvalidError(f"invalid prompt_version: {prompt_version!r}")
        path = self._dir / f"summarization_v{version_num}.txt"
        if not path.exists():
            raise ConfigInvalidError(f"prompt template not found: {path}")
        return path.read_text(encoding="utf-8")
