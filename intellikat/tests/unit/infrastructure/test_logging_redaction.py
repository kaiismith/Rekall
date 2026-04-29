"""Unit tests for `Redactor` (Requirement 12.4)."""

from __future__ import annotations

from pydantic import SecretStr

from intellikat.infrastructure.logging.redaction import REDACTED, Redactor


def test_secret_str_value_redacted() -> None:
    r = Redactor()
    out = r.redact({"normal": "ok", "creds": SecretStr("super-secret")})
    assert out["normal"] == "ok"
    assert out["creds"] == REDACTED


def test_key_regex_redacts_token_password_etc() -> None:
    r = Redactor()
    out = r.redact(
        {
            "hf_token": "hf_abc",
            "api_key": "key",
            "Password": "p",
            "api-key": "k",
            "connection_string": "Endpoint=...",
            "credential": "x",
            "neutral": "x",
        }
    )
    assert out["hf_token"] == REDACTED
    assert out["api_key"] == REDACTED
    assert out["Password"] == REDACTED
    assert out["api-key"] == REDACTED
    assert out["connection_string"] == REDACTED
    assert out["credential"] == REDACTED
    assert out["neutral"] == "x"


def test_nested_dicts_redacted() -> None:
    r = Redactor()
    out = r.redact({"outer": {"hf_token": "secret", "ok": "yes"}})
    assert out["outer"]["hf_token"] == REDACTED
    assert out["outer"]["ok"] == "yes"


def test_non_secret_values_preserved() -> None:
    r = Redactor()
    out = r.redact({"count": 42, "ratio": 0.5, "flag": True})
    assert out == {"count": 42, "ratio": 0.5, "flag": True}
