"""Shared test fixtures."""

from __future__ import annotations

import logging

import pytest

from intellikat.infrastructure.logging.logger import ContextLogger


@pytest.fixture
def logger() -> ContextLogger:
    """A real ContextLogger that emits to a NullHandler — silent in tests."""
    base = logging.getLogger("intellikat.tests")
    base.handlers = [logging.NullHandler()]
    base.propagate = False
    return ContextLogger(base)
