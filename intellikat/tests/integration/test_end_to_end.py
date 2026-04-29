"""End-to-end integration test (no Service Bus broker).

Exercises the full vertical slice: real Postgres (testcontainers) +
fake-but-realistic in-memory consumer + respx-stubbed HF Inference API +
hand-rolled fake Foundry client.

Skips if testcontainers / Postgres aren't available.
"""

from __future__ import annotations

import json
import os
from datetime import UTC, datetime, timedelta
from types import SimpleNamespace
from typing import Any
from uuid import UUID, uuid4

import httpx
import pytest
import respx
from sqlalchemy import create_engine, text
from sqlalchemy.orm import sessionmaker

from intellikat.application.use_cases.process_insight_job import ProcessInsightJob
from intellikat.domain.value_objects.engine_snapshot import EngineSnapshot
from intellikat.domain.value_objects.job_reference import (
    JobReference,
    Scope,
    SegmentIndexRange,
)
from intellikat.domain.value_objects.prompt_version import make_prompt_version
from intellikat.infrastructure.persistence.repositories.transcript_reader_pg import (
    TranscriptReaderPg,
)
from intellikat.infrastructure.persistence.unit_of_work import SqlAlchemyUnitOfWork
from intellikat.infrastructure.sentiment.hf_inference_client import HFInferenceClient

pytestmark = pytest.mark.integration

try:
    from testcontainers.postgres import PostgresContainer
except ImportError:
    PostgresContainer = None  # type: ignore[assignment,misc]


# Reuse the schema seeding helper from the repository test.
from tests.integration.test_repositories_pg import (  # noqa: E402
    SHARED_TABLES_DDL,
    _seed_session_with_segments,
)


HF_BASE = "https://hf-test.example"
HF_MODEL = "MarieAngeA13/Sentiment-Analysis-BERT"


@pytest.fixture(scope="module")
def pg_engine():  # type: ignore[no-untyped-def]
    if PostgresContainer is None:
        pytest.skip("testcontainers not installed")
    if os.environ.get("INTELLIKAT_SKIP_INTEGRATION") == "1":
        pytest.skip("integration tests disabled by env")

    with PostgresContainer("postgres:16-alpine") as pg:
        url = pg.get_connection_url().replace(
            "postgresql+psycopg2://", "postgresql+psycopg://"
        )
        engine = create_engine(url, future=True)
        with engine.begin() as conn:
            for stmt in SHARED_TABLES_DDL.split(";"):
                if stmt.strip():
                    conn.execute(text(stmt))

        from alembic import command
        from alembic.config import Config as AlembicConfig

        ini_path = os.path.join(
            os.path.dirname(__file__), "..", "..", "migrations", "alembic.ini"
        )
        cfg = AlembicConfig(os.path.abspath(ini_path))
        cfg.set_main_option("sqlalchemy.url", url)
        cfg.set_main_option("script_location", os.path.abspath(os.path.dirname(ini_path)))

        os.environ["INTELLIKAT_DATABASE_URL"] = url
        os.environ.setdefault("INTELLIKAT_SERVICEBUS_NAMESPACE", "test")
        os.environ.setdefault(
            "INTELLIKAT_SERVICEBUS_CONNECTION_STRING",
            "Endpoint=sb://x;SharedAccessKeyName=k;SharedAccessKey=v",
        )
        os.environ.setdefault("INTELLIKAT_HF_TOKEN", "hf_test")
        os.environ.setdefault("INTELLIKAT_FOUNDRY_ENDPOINT", "https://x.example.com/")
        os.environ.setdefault("INTELLIKAT_FOUNDRY_API_KEY", "key")

        command.upgrade(cfg, "head")
        yield engine


@pytest.fixture
def session_factory(pg_engine):  # type: ignore[no-untyped-def]
    return sessionmaker(bind=pg_engine, expire_on_commit=False)


# ── fake summarizer (no Foundry mock framework needed) ──────────────────────


class _StubSummarizer:
    model_id = "gpt-4o-mini"

    def __init__(self, content: str, key_points: list[str]) -> None:
        self._content = content
        self._key_points = key_points

    async def health_check(self) -> bool:
        return True

    async def summarize(self, transcript: str, prompt_version: str):  # type: ignore[no-untyped-def]
        from intellikat.application.dto.summary_result import SummaryResult

        return SummaryResult(
            content=self._content,
            key_points=self._key_points,
            prompt_tokens=len(transcript) // 4,
            completion_tokens=20,
        )


def _job_ref(session_id: UUID, speaker_user_id: UUID) -> JobReference:
    return JobReference(
        schema_version="1",
        job_id=uuid4(),
        event_type="transcript.session.closed",
        transcript_session_id=session_id,
        scope=Scope(kind="call", id=uuid4()),
        segment_index_range=SegmentIndexRange.model_validate({"from": 0, "to": 100}),
        speaker_user_id=speaker_user_id,
        engine_snapshot=EngineSnapshot(engine_mode="openai", model_id="whisper-1"),
        occurred_at=datetime.now(UTC),
        correlation_id="cid-e2e",
    )


@pytest.mark.asyncio
async def test_happy_path_writes_sentiments_summary_and_audit(
    pg_engine, session_factory, logger
):  # type: ignore[no-untyped-def]
    user_id, session_id, _ = _seed_session_with_segments(pg_engine, segment_count=3)
    from pydantic import SecretStr

    analyzer = HFInferenceClient(
        base_url=HF_BASE, model_id=HF_MODEL, token=SecretStr("x"), timeout_s=5
    )
    try:
        with respx.mock(base_url=HF_BASE) as router:
            router.post(f"/models/{HF_MODEL}").mock(
                return_value=httpx.Response(
                    200,
                    json=[[{"label": "POSITIVE", "score": 0.91}]],
                )
            )

            uc = ProcessInsightJob(
                reader=TranscriptReaderPg(session_factory),
                analyzer=analyzer,
                summarizer=_StubSummarizer("ok", ["a", "b"]),
                writer_factory=lambda: SqlAlchemyUnitOfWork(
                    session_factory, hf_mode="hosted"
                ),
                prompt_version=make_prompt_version("sum-v1"),
                failure_ratio_threshold=0.5,
                max_summary_input_chars=10_000,
                logger=logger,
            )
            await uc.execute(_job_ref(session_id, user_id))
    finally:
        await analyzer.close()

    with pg_engine.connect() as conn:
        sentiments = conn.execute(
            text(
                "SELECT COUNT(*) FROM transcript_segment_sentiments WHERE transcript_session_id = :sid"
            ),
            {"sid": session_id},
        ).scalar_one()
        summaries = conn.execute(
            text(
                "SELECT COUNT(*) FROM transcript_session_summaries WHERE transcript_session_id = :sid"
            ),
            {"sid": session_id},
        ).scalar_one()
        audits = conn.execute(
            text(
                "SELECT status, segments_processed FROM intellikat_jobs WHERE transcript_session_id = :sid"
            ),
            {"sid": session_id},
        ).fetchall()
    assert sentiments == 3
    assert summaries == 1
    assert len(audits) == 1
    assert audits[0][0] == "completed"
    assert audits[0][1] == 3


@pytest.mark.asyncio
async def test_hf_failures_above_threshold_abandons_no_persisted_rows(
    pg_engine, session_factory, logger
):  # type: ignore[no-untyped-def]
    from pydantic import SecretStr

    user_id, session_id, _ = _seed_session_with_segments(pg_engine, segment_count=2)
    analyzer = HFInferenceClient(
        base_url=HF_BASE, model_id=HF_MODEL, token=SecretStr("x"), timeout_s=2
    )
    try:
        with respx.mock(base_url=HF_BASE) as router:
            router.post(f"/models/{HF_MODEL}").mock(
                return_value=httpx.Response(500)
            )

            uc = ProcessInsightJob(
                reader=TranscriptReaderPg(session_factory),
                analyzer=analyzer,
                summarizer=_StubSummarizer("ok", []),
                writer_factory=lambda: SqlAlchemyUnitOfWork(
                    session_factory, hf_mode="hosted"
                ),
                prompt_version=make_prompt_version("sum-v1"),
                failure_ratio_threshold=0.0,
                max_summary_input_chars=10_000,
                logger=logger,
            )
            with pytest.raises(Exception):
                await uc.execute(_job_ref(session_id, user_id))
    finally:
        await analyzer.close()

    with pg_engine.connect() as conn:
        sentiments = conn.execute(
            text(
                "SELECT COUNT(*) FROM transcript_segment_sentiments WHERE transcript_session_id = :sid"
            ),
            {"sid": session_id},
        ).scalar_one()
        summaries = conn.execute(
            text(
                "SELECT COUNT(*) FROM transcript_session_summaries WHERE transcript_session_id = :sid"
            ),
            {"sid": session_id},
        ).scalar_one()
        audit = conn.execute(
            text("SELECT status FROM intellikat_jobs WHERE transcript_session_id = :sid"),
            {"sid": session_id},
        ).fetchall()
    # No rows persisted; audit row marks failure
    assert sentiments == 0
    assert summaries == 0
    assert len(audit) == 1
    assert audit[0][0] == "failed"
