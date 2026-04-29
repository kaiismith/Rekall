"""Foundry AI / Azure OpenAI client wiring.

Auth: managed identity preferred (`DefaultAzureCredential`), API key fallback.
The wrapped `openai.AsyncAzureOpenAI` is exposed via `.client`.
"""

from __future__ import annotations

from typing import TYPE_CHECKING

from openai import AsyncAzureOpenAI

if TYPE_CHECKING:
    from intellikat.infrastructure.config.settings import Settings

VERSION = "0.1.0"


class FoundryClient:
    def __init__(self, settings: Settings) -> None:
        endpoint = str(settings.foundry_endpoint)
        common: dict[str, object] = {
            "azure_endpoint": endpoint,
            "api_version": settings.foundry_api_version,
            "timeout": settings.foundry_request_timeout_seconds,
            "default_headers": {"User-Agent": f"rekall-intellikat/{VERSION}"},
        }

        if settings.foundry_use_managed_identity:
            from azure.identity import DefaultAzureCredential, get_bearer_token_provider  # noqa: PLC0415

            credential = DefaultAzureCredential()
            token_provider = get_bearer_token_provider(
                credential, "https://cognitiveservices.azure.com/.default"
            )
            self._client = AsyncAzureOpenAI(
                azure_ad_token_provider=token_provider,
                **common,  # type: ignore[arg-type]
            )
        else:
            assert settings.foundry_api_key is not None  # validated in settings
            self._client = AsyncAzureOpenAI(
                api_key=settings.foundry_api_key.get_secret_value(),
                **common,  # type: ignore[arg-type]
            )

    @property
    def client(self) -> AsyncAzureOpenAI:
        return self._client

    async def close(self) -> None:
        await self._client.close()
