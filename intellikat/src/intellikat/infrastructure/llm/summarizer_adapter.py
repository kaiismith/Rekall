"""Foundry-backed implementation of the `Summarizer` port.

Sends the prompt as a single user message; coerces structured JSON via
`response_format`; on parse failure retries once with a corrective system
message; on rate-limit (429) waits the suggested time; on 5xx retries with
exponential backoff up to 3 attempts.
"""

from __future__ import annotations

import asyncio
import json
from typing import TYPE_CHECKING, Any

from openai import APIStatusError, RateLimitError
from pydantic import ValidationError

from intellikat.application.dto.summary_result import SummaryResult
from intellikat.domain.errors import SummaryParseError, UpstreamError
from intellikat.infrastructure.llm.response_models import SummaryResponseModel
from intellikat.infrastructure.logging import catalog

if TYPE_CHECKING:
    from intellikat.domain.value_objects.prompt_version import PromptVersion
    from intellikat.infrastructure.llm.foundry_client import FoundryClient
    from intellikat.infrastructure.logging.logger import ContextLogger


class FoundrySummarizerAdapter:
    """Implements the `Summarizer` port against Azure OpenAI / Foundry."""

    def __init__(
        self,
        *,
        foundry: FoundryClient,
        prompt_template: str,
        deployment: str,
        prompt_version: PromptVersion,
        logger: ContextLogger,
    ) -> None:
        self._client = foundry.client
        self._template = prompt_template
        self._deployment = deployment
        self._prompt_version = prompt_version
        self._logger = logger

    @property
    def model_id(self) -> str:
        return self._deployment

    async def health_check(self) -> bool:
        try:
            # Cheap metadata call — avoids burning a completion token budget.
            await self._client.models.list()
            return True
        except Exception:  # noqa: BLE001
            return False

    async def summarize(self, transcript: str, prompt_version: PromptVersion) -> SummaryResult:
        if prompt_version != self._prompt_version:
            raise ValueError(
                f"requested prompt {prompt_version!r} but loaded {self._prompt_version!r}"
            )

        prompt = self._template.format(transcript=transcript, prompt_version=prompt_version)

        # First attempt: plain prompt. Second attempt: corrective system message.
        last_parse_error: Exception | None = None
        for parse_attempt in range(2):
            resp = await self._chat_with_backoff(prompt, corrective=parse_attempt > 0)
            try:
                content = resp.choices[0].message.content or "{}"
                parsed = SummaryResponseModel.model_validate_json(content)
                usage = resp.usage
                return SummaryResult(
                    content=parsed.summary,
                    key_points=parsed.key_points or None,
                    prompt_tokens=usage.prompt_tokens if usage else None,
                    completion_tokens=usage.completion_tokens if usage else None,
                )
            except (ValidationError, json.JSONDecodeError) as e:
                last_parse_error = e
                if parse_attempt == 0:
                    self._logger.warn(catalog.SUMMARY_PARSE_RETRY, err=str(e))
                    continue
                self._logger.error(catalog.SUMMARY_PARSE_FAILED, err=str(e))
                raise SummaryParseError(str(e)) from e

        # Defensive — loop above always returns or raises.
        raise SummaryParseError(str(last_parse_error))

    async def _chat_with_backoff(self, prompt: str, *, corrective: bool) -> Any:
        messages: list[dict[str, str]] = [{"role": "user", "content": prompt}]
        if corrective:
            messages.append(
                {
                    "role": "system",
                    "content": (
                        "Your previous response was not valid JSON. "
                        "Re-emit ONLY the JSON object."
                    ),
                }
            )

        last_exc: Exception | None = None
        for attempt in range(3):
            try:
                return await self._client.chat.completions.create(
                    model=self._deployment,
                    messages=messages,  # type: ignore[arg-type]
                    response_format={"type": "json_object"},
                    temperature=0.2,
                    max_tokens=600,
                )
            except RateLimitError as e:
                last_exc = e
                retry_after = self._retry_after(e) or float(2**attempt)
                await asyncio.sleep(retry_after)
            except APIStatusError as e:
                last_exc = e
                status = getattr(e, "status_code", None)
                if status is not None and 500 <= status < 600 and attempt < 2:
                    await asyncio.sleep(2**attempt)
                    continue
                raise UpstreamError(f"foundry status {status}: {e}") from e
            except Exception as e:  # noqa: BLE001
                raise UpstreamError(f"foundry call failed: {e}") from e

        raise UpstreamError(f"foundry call failed after retries: {last_exc}")

    @staticmethod
    def _retry_after(exc: RateLimitError) -> float | None:
        try:
            resp = getattr(exc, "response", None)
            if resp is None:
                return None
            value = resp.headers.get("Retry-After")
            return float(value) if value is not None else None
        except (TypeError, ValueError):
            return None
