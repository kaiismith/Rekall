"""Unit tests for the health endpoints."""

from __future__ import annotations

from types import SimpleNamespace

import pytest
from fastapi.testclient import TestClient

from intellikat.interfaces.http.app import build_app


def _container(*, db=True, sentiment=True, foundry=True):  # type: ignore[no-untyped-def]
    class _Sess:
        def __enter__(self):
            return self

        def __exit__(self, *_):
            return None

        def execute(self, *_a, **_kw):
            if not db:
                raise RuntimeError("db down")
            return None

    sentiment_mock = SimpleNamespace(health_check=lambda: _Async(sentiment))
    summarizer_mock = SimpleNamespace(health_check=lambda: _Async(foundry))

    return SimpleNamespace(
        settings=SimpleNamespace(service_env="development", log_level="INFO", http_listen="0.0.0.0:8090"),
        session_factory=lambda: _Sess(),
        consumer=object(),
        sentiment_analyzer=sentiment_mock,
        summarizer=summarizer_mock,
    )


class _Async:
    def __init__(self, value: bool) -> None:
        self._value = value

    def __await__(self):
        async def _impl():
            return self._value

        return _impl().__await__()


def test_healthz_always_200() -> None:
    app = build_app(_container())  # type: ignore[arg-type]
    client = TestClient(app)
    resp = client.get("/healthz")
    assert resp.status_code == 200
    assert resp.json() == {"status": "ok"}


def test_readyz_returns_200_when_all_pass() -> None:
    app = build_app(_container())  # type: ignore[arg-type]
    client = TestClient(app)
    resp = client.get("/readyz")
    assert resp.status_code == 200
    body = resp.json()
    assert body["status"] == "ready"
    assert all(body["checks"].values())


@pytest.mark.parametrize(
    "broken",
    [
        {"db": False},
        {"sentiment": False},
        {"foundry": False},
    ],
)
def test_readyz_returns_503_when_any_fails(broken: dict[str, bool]) -> None:
    app = build_app(_container(**broken))  # type: ignore[arg-type]
    client = TestClient(app)
    resp = client.get("/readyz")
    assert resp.status_code == 503
    assert resp.json()["status"] == "not_ready"


def test_docs_hidden_in_production() -> None:
    c = _container()
    c.settings = SimpleNamespace(service_env="production", log_level="INFO", http_listen="0.0.0.0:8090")
    app = build_app(c)  # type: ignore[arg-type]
    client = TestClient(app)
    assert client.get("/docs").status_code == 404
    assert client.get("/openapi.json").status_code == 404
