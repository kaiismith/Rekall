"""Unit tests for `build_sentiment_analyzer`."""

from __future__ import annotations

from types import SimpleNamespace

import pytest
from pydantic import HttpUrl, SecretStr

from intellikat.domain.errors import ConfigInvalidError
from intellikat.infrastructure.sentiment.factory import build_sentiment_analyzer


def _settings(**overrides):  # type: ignore[no-untyped-def]
    base = SimpleNamespace(
        hf_mode="hosted",
        hf_token=SecretStr("hf_x"),
        hf_model_id="MarieAngeA13/Sentiment-Analysis-BERT",
        hf_inference_base_url=HttpUrl("https://api.example.com/"),
        hf_request_timeout_seconds=10,
        hf_local_cache_dir="/tmp/intellikat-hf",
        hf_local_device="cpu",
        hf_local_batch_size=4,
        max_input_tokens=128,
    )
    for k, v in overrides.items():
        setattr(base, k, v)
    return base


def test_hosted_returns_hf_inference_client() -> None:
    from intellikat.infrastructure.sentiment.hf_inference_client import HFInferenceClient

    a = build_sentiment_analyzer(_settings())
    assert isinstance(a, HFInferenceClient)
    assert a.hf_mode == "hosted"
    assert a.model_id == "MarieAngeA13/Sentiment-Analysis-BERT"


def test_hosted_requires_token() -> None:
    with pytest.raises(ConfigInvalidError) as exc:
        build_sentiment_analyzer(_settings(hf_token=None))
    assert "HF_TOKEN" in str(exc.value)


def test_local_without_extras_raises_config_invalid() -> None:
    """When HF_MODE=local but transformers is not installed, factory raises ConfigInvalidError."""
    pytest.importorskip
    try:
        import transformers  # noqa: F401
    except ImportError:
        with pytest.raises(ConfigInvalidError) as exc:
            build_sentiment_analyzer(_settings(hf_mode="local"))
        assert "local" in str(exc.value).lower()
    else:
        pytest.skip("transformers is installed; cannot test the missing-extras path")


def test_unknown_mode_raises() -> None:
    with pytest.raises(ConfigInvalidError):
        build_sentiment_analyzer(_settings(hf_mode="sideways"))
