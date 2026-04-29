"""Liveness + readiness endpoints.

`/healthz` is a constant 200 — used by Kubernetes liveness probes.
`/readyz` runs all dependency checks — used by readiness probes.
"""

from __future__ import annotations

import asyncio
from typing import TYPE_CHECKING, Any

from fastapi import APIRouter, Request, Response
from sqlalchemy import text

if TYPE_CHECKING:
    from intellikat.infrastructure.di.container import Container

router = APIRouter()


@router.get("/healthz")
async def healthz() -> dict[str, str]:
    return {"status": "ok"}


@router.get("/readyz")
async def readyz(request: Request, response: Response) -> dict[str, Any]:
    container: Container = request.app.state.container
    result = await run_readiness_checks(container)
    if not result["ready"]:
        response.status_code = 503
        return {"status": "not_ready", "checks": result["checks"]}
    return {"status": "ready", "checks": result["checks"]}


async def run_readiness_checks(container: Container) -> dict[str, Any]:
    db_ok, sb_ok, hf_ok, foundry_ok = await asyncio.gather(
        _check_db(container),
        _check_servicebus(container),
        _check_sentiment(container),
        _check_foundry(container),
        return_exceptions=False,
    )
    checks = {"db": db_ok, "servicebus": sb_ok, "sentiment": hf_ok, "foundry": foundry_ok}
    return {"ready": all(checks.values()), "checks": checks}


async def _check_db(container: Container) -> bool:
    try:
        def _ping() -> None:
            with container.session_factory() as sess:
                sess.execute(text("SELECT 1"))

        await asyncio.wait_for(asyncio.to_thread(_ping), timeout=2.0)
        return True
    except Exception:  # noqa: BLE001
        return False


async def _check_servicebus(container: Container) -> bool:
    # We don't pull a message — that would alter delivery counts.
    # Healthy = the ServiceBusClient was constructed without raising.
    return container.consumer is not None


async def _check_sentiment(container: Container) -> bool:
    try:
        return await container.sentiment_analyzer.health_check()
    except Exception:  # noqa: BLE001
        return False


async def _check_foundry(container: Container) -> bool:
    try:
        return await container.summarizer.health_check()
    except Exception:  # noqa: BLE001
        return False
