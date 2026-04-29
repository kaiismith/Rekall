"""Application settings.

All knobs come from environment variables prefixed `INTELLIKAT_`. A `.env`
file in the working directory is loaded for local dev convenience. Cross-field
validation runs at instantiation time; failures raise `ConfigInvalidError`.
"""

from __future__ import annotations

from pathlib import Path
from typing import Literal

from pydantic import (
    Field,
    HttpUrl,
    PostgresDsn,
    SecretStr,
    field_validator,
    model_validator,
)
from pydantic_settings import BaseSettings, SettingsConfigDict

from intellikat.domain.errors import ConfigInvalidError

HFMode = Literal["hosted", "local"]
HFLocalDevice = Literal["cpu", "cuda", "mps", "auto"]
LogLevel = Literal["DEBUG", "INFO", "WARNING", "ERROR"]
LogFormat = Literal["json", "text"]
ServiceEnv = Literal["development", "staging", "production"]


class Settings(BaseSettings):
    """Runtime configuration sourced from env vars (`INTELLIKAT_*`)."""

    model_config = SettingsConfigDict(
        env_prefix="INTELLIKAT_",
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
        case_sensitive=False,
    )

    # ── database ────────────────────────────────────────────────────────────
    database_url: PostgresDsn
    db_pool_size: int = Field(default=5, ge=1, le=64)
    db_max_overflow: int = Field(default=5, ge=0, le=64)
    db_application_name: str = "intellikat"

    # ── service bus ─────────────────────────────────────────────────────────
    servicebus_namespace: str
    servicebus_fully_qualified_namespace: str | None = None
    servicebus_connection_string: SecretStr | None = None
    servicebus_topic: str = "rekall.transcript.insights"
    servicebus_subscription: str = "intellikat"
    servicebus_max_concurrent_messages: int = Field(default=4, ge=1, le=64)
    servicebus_message_lock_renewal_seconds: int = Field(default=240, ge=30, le=3600)
    servicebus_max_delivery_count: int = Field(default=5, ge=1, le=20)

    # ── hugging face ────────────────────────────────────────────────────────
    hf_mode: HFMode = "hosted"
    hf_token: SecretStr | None = None
    hf_model_id: str = "MarieAngeA13/Sentiment-Analysis-BERT"
    hf_inference_base_url: HttpUrl = HttpUrl("https://api-inference.huggingface.co")
    hf_local_cache_dir: Path = Path("/var/cache/intellikat/hf")
    hf_local_device: HFLocalDevice = "auto"
    hf_local_batch_size: int = Field(default=16, ge=1, le=256)
    hf_request_timeout_seconds: int = Field(default=30, ge=1, le=600)

    # ── foundry ai ──────────────────────────────────────────────────────────
    foundry_endpoint: HttpUrl
    foundry_api_key: SecretStr | None = None
    foundry_use_managed_identity: bool = False
    foundry_summary_deployment: str = "gpt-4o-mini"
    foundry_api_version: str = "2024-10-21"
    foundry_request_timeout_seconds: int = Field(default=60, ge=1, le=600)

    # ── processing knobs ────────────────────────────────────────────────────
    max_input_tokens: int = Field(default=512, ge=16, le=8192)
    max_summary_input_chars: int = Field(default=60_000, ge=500, le=1_000_000)
    sentiment_failure_ratio_threshold: float = Field(default=0.25, ge=0.0, le=1.0)
    prompt_version: str = "sum-v1"
    prompt_dir: Path = Path("config/prompts")

    # ── http ────────────────────────────────────────────────────────────────
    http_listen: str = "0.0.0.0:8090"

    # ── logging ─────────────────────────────────────────────────────────────
    log_level: LogLevel = "INFO"
    log_format: LogFormat = "json"

    # ── generic ─────────────────────────────────────────────────────────────
    service_env: ServiceEnv = "development"
    max_concurrent_jobs: int = Field(default=4, ge=1, le=64)
    graceful_drain_seconds: int = Field(default=60, ge=1, le=600)

    # ── cross-field validation ──────────────────────────────────────────────

    @field_validator("prompt_version")
    @classmethod
    def _prompt_version_shape(cls, v: str) -> str:
        if not v or not v.startswith("sum-v"):
            raise ConfigInvalidError(
                f"prompt_version must look like 'sum-vN' (got {v!r})"
            )
        return v

    @model_validator(mode="after")
    def _validate_combinations(self) -> Settings:
        if self.hf_mode == "hosted" and self.hf_token is None:
            raise ConfigInvalidError("INTELLIKAT_HF_TOKEN is required when HF_MODE=hosted")

        if not self.foundry_use_managed_identity and self.foundry_api_key is None:
            raise ConfigInvalidError(
                "INTELLIKAT_FOUNDRY_API_KEY is required unless FOUNDRY_USE_MANAGED_IDENTITY=true"
            )

        if self.service_env == "production":
            host = self.database_url.host or ""
            if host in {"localhost", "127.0.0.1", "::1"}:
                raise ConfigInvalidError(
                    "INTELLIKAT_DATABASE_URL must not point at localhost in production"
                )

        if (
            self.servicebus_connection_string is None
            and self.servicebus_fully_qualified_namespace is None
        ):
            raise ConfigInvalidError(
                "either INTELLIKAT_SERVICEBUS_CONNECTION_STRING or "
                "INTELLIKAT_SERVICEBUS_FULLY_QUALIFIED_NAMESPACE must be set"
            )

        if (
            self.service_env == "production"
            and self.servicebus_connection_string is not None
        ):
            raise ConfigInvalidError(
                "INTELLIKAT_SERVICEBUS_CONNECTION_STRING is not permitted in production "
                "(use managed identity via FULLY_QUALIFIED_NAMESPACE)"
            )

        return self
