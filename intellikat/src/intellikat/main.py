"""Binary entrypoint.

Three modes:
- worker (default) — run the Service Bus consumer loop
- migrate          — apply Alembic migrations and exit (Kubernetes init-container)
- health-check     — run readiness checks once and exit 0/1 (cluster smoke test)
"""

from __future__ import annotations

import argparse
import asyncio
import os
import signal
import sys
from typing import Any

from intellikat.domain.errors import ConfigInvalidError
from intellikat.infrastructure.config.settings import Settings
from intellikat.infrastructure.di.container import Container, build_container
from intellikat.infrastructure.logging import catalog
from intellikat.infrastructure.logging.logger import ContextLogger, configure_logging


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    p = argparse.ArgumentParser(prog="intellikat")
    p.add_argument("--mode", choices=("worker", "migrate", "health-check"), default="worker")
    p.add_argument("--config", default=os.getenv("INTELLIKAT_CONFIG_PATH"))
    return p.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv)

    try:
        settings = Settings()  # type: ignore[call-arg]
    except ConfigInvalidError as e:
        # Logger isn't configured yet — write straight to stderr.
        sys.stderr.write(f"INTELLIKAT_CONFIG_INVALID: {e}\n")
        return 1
    except Exception as e:  # noqa: BLE001 — pydantic wraps validation errors
        sys.stderr.write(f"INTELLIKAT_CONFIG_INVALID: {e}\n")
        return 1

    logger = configure_logging(settings)

    if args.mode == "migrate":
        return _run_alembic_upgrade(settings, logger)
    if args.mode == "health-check":
        return asyncio.run(_run_oneshot_readiness(settings, logger))

    try:
        container = build_container(settings, logger)
    except ConfigInvalidError as e:
        logger.error(catalog.STARTUP_FAILED, err=str(e))
        return 1
    except Exception as e:  # noqa: BLE001
        logger.error(catalog.STARTUP_FAILED, err=str(e))
        return 1

    logger.info(
        catalog.HF_MODE_SELECTED,
        hf_mode=settings.hf_mode,
        hf_model_id=settings.hf_model_id,
    )

    try:
        return asyncio.run(_run_worker(container))
    except Exception as e:  # noqa: BLE001
        logger.error(catalog.FATAL, err=str(e), exc_info=True)
        return 2


async def _run_worker(container: Container) -> int:
    stop_event = asyncio.Event()
    loop = asyncio.get_running_loop()

    def _handle_signal(*_: Any) -> None:
        container.logger.info(catalog.GRACEFUL_DRAIN_BEGIN)
        stop_event.set()

    for sig_name in ("SIGTERM", "SIGINT"):
        sig = getattr(signal, sig_name, None)
        if sig is None:
            continue
        try:
            loop.add_signal_handler(sig, _handle_signal)
        except NotImplementedError:
            # Windows event loop doesn't support add_signal_handler.
            signal.signal(sig, lambda *_a: _handle_signal())

    # Warm up the analyzer (fetches model_revision in hosted mode; loads weights in local).
    await container.sentiment_analyzer.warmup()
    container.logger.info(catalog.HF_LOAD_OK)

    # Lazy import: avoids circular dep with FastAPI app at module load.
    from intellikat.interfaces.http.app import start_http_server  # noqa: PLC0415

    http_server = await start_http_server(container)
    container.logger.info(catalog.STARTUP_OK)

    try:
        await container.consumer.run(container.process_use_case.execute, stop_event)
    finally:
        await asyncio.wait_for(
            http_server.shutdown(),
            timeout=container.settings.graceful_drain_seconds,
        )
        await container.aclose()
        container.logger.info(catalog.GRACEFUL_DRAIN_END)
    return 0


def _run_alembic_upgrade(settings: Settings, logger: ContextLogger) -> int:
    try:
        from alembic import command  # noqa: PLC0415
        from alembic.config import Config as AlembicConfig  # noqa: PLC0415

        ini_path = os.path.join(
            os.path.dirname(os.path.dirname(os.path.dirname(__file__))),
            "migrations",
            "alembic.ini",
        )
        cfg = AlembicConfig(ini_path)
        cfg.set_main_option("sqlalchemy.url", str(settings.database_url))
        command.upgrade(cfg, "head")
        return 0
    except Exception as e:  # noqa: BLE001
        logger.error(catalog.STARTUP_FAILED, err=str(e), exc_info=True)
        return 1


async def _run_oneshot_readiness(settings: Settings, logger: ContextLogger) -> int:
    try:
        container = build_container(settings, logger)
    except Exception as e:  # noqa: BLE001
        logger.error(catalog.STARTUP_FAILED, err=str(e))
        return 1
    try:
        from intellikat.interfaces.http.handlers.health import run_readiness_checks  # noqa: PLC0415

        result = await run_readiness_checks(container)
        return 0 if result["ready"] else 1
    finally:
        await container.aclose()


if __name__ == "__main__":
    sys.exit(main())
