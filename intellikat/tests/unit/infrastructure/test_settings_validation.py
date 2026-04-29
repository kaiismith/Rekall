"""Unit tests for `Settings` cross-field validation (Requirement 8.3)."""

from __future__ import annotations

import pytest

from intellikat.domain.errors import ConfigInvalidError
from intellikat.infrastructure.config.settings import Settings


def _base_env(**overrides: str) -> dict[str, str]:
    env = {
        "INTELLIKAT_DATABASE_URL": "postgresql+psycopg://intellikat:pw@db.internal:5432/rekall",
        "INTELLIKAT_SERVICEBUS_NAMESPACE": "rekall-dev",
        "INTELLIKAT_SERVICEBUS_FULLY_QUALIFIED_NAMESPACE": "ns.servicebus.windows.net",
        "INTELLIKAT_HF_MODE": "hosted",
        "INTELLIKAT_HF_TOKEN": "hf_test",
        "INTELLIKAT_FOUNDRY_ENDPOINT": "https://foundry.example.com/",
        "INTELLIKAT_FOUNDRY_API_KEY": "key_test",
    }
    env.update(overrides)
    return env


def _build(monkeypatch: pytest.MonkeyPatch, env: dict[str, str]) -> Settings:
    for k in list(env.keys()):
        monkeypatch.setenv(k, env[k])
    return Settings(_env_file=None)  # type: ignore[call-arg]


def test_happy_path(monkeypatch: pytest.MonkeyPatch) -> None:
    s = _build(monkeypatch, _base_env())
    assert s.hf_mode == "hosted"
    assert s.foundry_summary_deployment == "gpt-4o-mini"


def test_hf_hosted_requires_token(monkeypatch: pytest.MonkeyPatch) -> None:
    env = _base_env()
    env.pop("INTELLIKAT_HF_TOKEN")
    with pytest.raises(Exception) as exc:  # pydantic wraps the inner ConfigInvalidError
        _build(monkeypatch, env)
    assert "HF_TOKEN" in str(exc.value)


def test_foundry_requires_key_when_not_managed_identity(monkeypatch: pytest.MonkeyPatch) -> None:
    env = _base_env()
    env.pop("INTELLIKAT_FOUNDRY_API_KEY")
    with pytest.raises(Exception) as exc:
        _build(monkeypatch, env)
    assert "FOUNDRY_API_KEY" in str(exc.value)


def test_foundry_managed_identity_skips_key_check(monkeypatch: pytest.MonkeyPatch) -> None:
    env = _base_env()
    env.pop("INTELLIKAT_FOUNDRY_API_KEY")
    env["INTELLIKAT_FOUNDRY_USE_MANAGED_IDENTITY"] = "true"
    s = _build(monkeypatch, env)
    assert s.foundry_use_managed_identity is True


def test_production_rejects_localhost_db(monkeypatch: pytest.MonkeyPatch) -> None:
    env = _base_env(
        INTELLIKAT_SERVICE_ENV="production",
        INTELLIKAT_DATABASE_URL="postgresql+psycopg://u:p@localhost:5432/rekall",
    )
    with pytest.raises(Exception) as exc:
        _build(monkeypatch, env)
    assert "localhost" in str(exc.value)


def test_servicebus_requires_either_fqns_or_connection_string(monkeypatch: pytest.MonkeyPatch) -> None:
    env = _base_env()
    env.pop("INTELLIKAT_SERVICEBUS_FULLY_QUALIFIED_NAMESPACE")
    with pytest.raises(Exception) as exc:
        _build(monkeypatch, env)
    assert "SERVICEBUS" in str(exc.value)


def test_production_forbids_connection_string(monkeypatch: pytest.MonkeyPatch) -> None:
    env = _base_env(
        INTELLIKAT_SERVICE_ENV="production",
        INTELLIKAT_SERVICEBUS_CONNECTION_STRING="Endpoint=sb://x;SharedAccessKeyName=k;SharedAccessKey=v",
    )
    with pytest.raises(Exception) as exc:
        _build(monkeypatch, env)
    assert "managed identity" in str(exc.value).lower()


def test_prompt_version_must_match_shape(monkeypatch: pytest.MonkeyPatch) -> None:
    env = _base_env(INTELLIKAT_PROMPT_VERSION="bad-name")
    with pytest.raises(Exception) as exc:
        _build(monkeypatch, env)
    assert "prompt_version" in str(exc.value)


def test_prompt_version_sum_v2_ok(monkeypatch: pytest.MonkeyPatch) -> None:
    s = _build(monkeypatch, _base_env(INTELLIKAT_PROMPT_VERSION="sum-v2"))
    assert s.prompt_version == "sum-v2"


def test_config_invalid_error_is_domain_error() -> None:
    from intellikat.domain.errors import IntellikatError

    assert issubclass(ConfigInvalidError, IntellikatError)
