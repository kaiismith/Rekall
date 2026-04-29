"""Unit tests for `FoundrySummarizerAdapter`.

Uses a hand-rolled fake of the Azure OpenAI client (the openai SDK doesn't
play well with respx because it wraps the request layer in its own retry).
"""

from __future__ import annotations

import json
from types import SimpleNamespace
from typing import Any

import httpx
import pytest
from openai import APIStatusError, RateLimitError

from intellikat.domain.errors import SummaryParseError, UpstreamError
from intellikat.domain.value_objects.prompt_version import make_prompt_version
from intellikat.infrastructure.llm.summarizer_adapter import FoundrySummarizerAdapter


# ── fake Azure OpenAI client ────────────────────────────────────────────────


class _FakeChatCompletions:
    def __init__(self, responses: list[Any]) -> None:
        self._responses = list(responses)
        self.calls: list[dict[str, Any]] = []

    async def create(self, **kwargs: Any) -> Any:
        self.calls.append(kwargs)
        if not self._responses:
            raise AssertionError("no more fake responses queued")
        item = self._responses.pop(0)
        if isinstance(item, Exception):
            raise item
        return item


class _FakeChat:
    def __init__(self, completions: _FakeChatCompletions) -> None:
        self.completions = completions


class _FakeAsyncAzureOpenAI:
    def __init__(self, completions: _FakeChatCompletions) -> None:
        self.chat = _FakeChat(completions)
        self.models = SimpleNamespace(list=self._models_list)

    async def _models_list(self) -> list[Any]:
        return []


class _FakeFoundry:
    def __init__(self, completions: _FakeChatCompletions) -> None:
        self._client = _FakeAsyncAzureOpenAI(completions)

    @property
    def client(self) -> _FakeAsyncAzureOpenAI:
        return self._client


def _resp(content: str, prompt_tokens: int = 100, completion_tokens: int = 30) -> Any:
    return SimpleNamespace(
        choices=[SimpleNamespace(message=SimpleNamespace(content=content))],
        usage=SimpleNamespace(
            prompt_tokens=prompt_tokens,
            completion_tokens=completion_tokens,
        ),
    )


def _http_response(status: int, headers: dict[str, str] | None = None) -> httpx.Response:
    return httpx.Response(
        status, request=httpx.Request("POST", "https://x.example/"), headers=headers or {}
    )


def _build(
    completions: _FakeChatCompletions, logger
):  # type: ignore[no-untyped-def]
    return FoundrySummarizerAdapter(
        foundry=_FakeFoundry(completions),  # type: ignore[arg-type]
        prompt_template="prompt {prompt_version}: {transcript}",
        deployment="gpt-4o-mini",
        prompt_version=make_prompt_version("sum-v1"),
        logger=logger,
    )


# ── tests ───────────────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_happy_path_parses_response(logger) -> None:  # type: ignore[no-untyped-def]
    body = json.dumps({"summary": "all good", "key_points": ["a", "b"]})
    cc = _FakeChatCompletions([_resp(body)])
    summarizer = _build(cc, logger)

    out = await summarizer.summarize("transcript text", make_prompt_version("sum-v1"))
    assert out.content == "all good"
    assert out.key_points == ["a", "b"]
    assert out.prompt_tokens == 100
    assert out.completion_tokens == 30


@pytest.mark.asyncio
async def test_parse_failure_then_success(logger) -> None:  # type: ignore[no-untyped-def]
    bad = "not json"
    good = json.dumps({"summary": "rescued", "key_points": ["x"]})
    cc = _FakeChatCompletions([_resp(bad), _resp(good)])
    summarizer = _build(cc, logger)

    out = await summarizer.summarize("t", make_prompt_version("sum-v1"))
    assert out.content == "rescued"
    # second call was corrective
    assert any(
        m["role"] == "system" and "valid JSON" in m["content"]
        for m in cc.calls[1]["messages"]
    )


@pytest.mark.asyncio
async def test_parse_failure_twice_raises(logger) -> None:  # type: ignore[no-untyped-def]
    cc = _FakeChatCompletions([_resp("bad1"), _resp("bad2")])
    summarizer = _build(cc, logger)
    with pytest.raises(SummaryParseError):
        await summarizer.summarize("t", make_prompt_version("sum-v1"))


@pytest.mark.asyncio
async def test_429_retries_once_then_succeeds(logger) -> None:  # type: ignore[no-untyped-def]
    err = RateLimitError(
        message="rate limited",
        response=_http_response(429, {"Retry-After": "0"}),
        body=None,
    )
    body = json.dumps({"summary": "ok", "key_points": []})
    cc = _FakeChatCompletions([err, _resp(body)])
    summarizer = _build(cc, logger)

    out = await summarizer.summarize("t", make_prompt_version("sum-v1"))
    assert out.content == "ok"
    assert out.key_points is None  # empty list normalised


@pytest.mark.asyncio
async def test_5xx_retries_then_raises_upstream(logger) -> None:  # type: ignore[no-untyped-def]
    err500 = APIStatusError(
        message="server boom", response=_http_response(500), body=None
    )
    cc = _FakeChatCompletions([err500, err500, err500])
    summarizer = _build(cc, logger)
    with pytest.raises(UpstreamError):
        await summarizer.summarize("t", make_prompt_version("sum-v1"))


@pytest.mark.asyncio
async def test_prompt_version_mismatch_raises(logger) -> None:  # type: ignore[no-untyped-def]
    cc = _FakeChatCompletions([])
    summarizer = _build(cc, logger)
    with pytest.raises(ValueError):
        await summarizer.summarize("t", make_prompt_version("sum-v2"))
