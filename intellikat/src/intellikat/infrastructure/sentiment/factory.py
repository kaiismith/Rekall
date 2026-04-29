"""Build the appropriate `SentimentAnalyzer` for the selected mode."""

from __future__ import annotations

from typing import TYPE_CHECKING

from intellikat.domain.errors import ConfigInvalidError

if TYPE_CHECKING:
    from intellikat.domain.ports.sentiment_analyzer import SentimentAnalyzer
    from intellikat.infrastructure.config.settings import Settings


def build_sentiment_analyzer(settings: Settings) -> SentimentAnalyzer:
    """Returns a `SentimentAnalyzer` per `settings.hf_mode`."""
    if settings.hf_mode == "hosted":
        if settings.hf_token is None:
            raise ConfigInvalidError("HF_MODE=hosted requires INTELLIKAT_HF_TOKEN")
        from intellikat.infrastructure.sentiment.hf_inference_client import (  # noqa: PLC0415
            HFInferenceClient,
        )

        return HFInferenceClient(
            base_url=str(settings.hf_inference_base_url),
            model_id=settings.hf_model_id,
            token=settings.hf_token,
            timeout_s=settings.hf_request_timeout_seconds,
        )

    if settings.hf_mode == "local":
        try:
            from intellikat.infrastructure.sentiment.hf_local_pipeline import (  # noqa: PLC0415
                HFLocalPipeline,
            )
        except ImportError as e:
            raise ConfigInvalidError(
                "HF_MODE=local requires `pip install intellikat[local]`"
            ) from e

        return HFLocalPipeline(
            model_id=settings.hf_model_id,
            cache_dir=settings.hf_local_cache_dir,
            device=settings.hf_local_device,
            batch_size=settings.hf_local_batch_size,
            max_input_tokens=settings.max_input_tokens,
        )

    raise ConfigInvalidError(f"unknown HF_MODE: {settings.hf_mode}")
