"""FastAPI app factory + Uvicorn lifespan wiring.

V1 exposes ONLY /healthz and /readyz. Docs are suppressed in production.
"""

from __future__ import annotations

import asyncio
from typing import TYPE_CHECKING, Any

import uvicorn
from fastapi import FastAPI

from intellikat.interfaces.http.handlers.health import router as health_router

if TYPE_CHECKING:
    from intellikat.infrastructure.di.container import Container


def build_app(container: Container) -> FastAPI:
    is_prod = container.settings.service_env == "production"
    app = FastAPI(
        title="Intellikat",
        version="0.1.0",
        docs_url=None if is_prod else "/docs",
        redoc_url=None if is_prod else "/redoc",
        openapi_url=None if is_prod else "/openapi.json",
    )
    app.state.container = container
    app.include_router(health_router)
    return app


class _UvicornHandle:
    def __init__(self, server: uvicorn.Server, task: asyncio.Task[Any]) -> None:
        self._server = server
        self._task = task

    async def shutdown(self) -> None:
        self._server.should_exit = True
        try:
            await self._task
        except asyncio.CancelledError:
            pass


async def start_http_server(container: Container) -> _UvicornHandle:
    """Start the FastAPI app on a background asyncio task; return a handle."""
    host, _, port_s = container.settings.http_listen.partition(":")
    port = int(port_s) if port_s else 8090
    cfg = uvicorn.Config(
        build_app(container),
        host=host,
        port=port,
        log_level=container.settings.log_level.lower(),
        access_log=False,
    )
    server = uvicorn.Server(cfg)
    task = asyncio.create_task(server.serve())
    # Give the server a moment to start before returning.
    for _ in range(20):
        if server.started:
            break
        await asyncio.sleep(0.05)
    return _UvicornHandle(server, task)
