"""Sentiment classification label."""

from __future__ import annotations

from enum import Enum


class SentimentLabel(str, Enum):
    POSITIVE = "POSITIVE"
    NEUTRAL = "NEUTRAL"
    NEGATIVE = "NEGATIVE"
