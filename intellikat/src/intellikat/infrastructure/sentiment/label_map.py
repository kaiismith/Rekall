"""Map raw HF sentiment labels to the canonical `SentimentLabel` enum."""

from __future__ import annotations

from intellikat.domain.value_objects.sentiment_label import SentimentLabel

_HF_TO_CANONICAL: dict[str, SentimentLabel] = {
    "POSITIVE": SentimentLabel.POSITIVE,
    "NEUTRAL": SentimentLabel.NEUTRAL,
    "NEGATIVE": SentimentLabel.NEGATIVE,
    # Some upstream variants — defensive
    "LABEL_0": SentimentLabel.NEGATIVE,
    "LABEL_1": SentimentLabel.NEUTRAL,
    "LABEL_2": SentimentLabel.POSITIVE,
}


def from_hf(raw: str) -> SentimentLabel:
    """Translate an HF label to the canonical enum; raise on unknown values."""
    upper = (raw or "").strip().upper()
    label = _HF_TO_CANONICAL.get(upper)
    if label is None:
        raise ValueError(f"unknown HF sentiment label: {raw!r}")
    return label
