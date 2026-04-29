"""Integration tests for the Postgres repository + UoW.

Spins up Postgres via testcontainers and runs Alembic migrations + a
minimal seed of the read-side tables (which intellikat does NOT own — they
are normally provisioned by the backend's own migrations).
"""

from __future__ import annotations

import os
from datetime import UTC, datetime, timedelta
from uuid import uuid4

import pytest
from sqlalchemy import create_engine, text
from sqlalchemy.orm import sessionmaker

from intellikat.domain.entities.insight_job import InsightJob
from intellikat.domain.entities.sentiment import SegmentSentiment
from intellikat.domain.entities.summary import SessionSummary
from intellikat.domain.value_objects.engine_snapshot import EngineSnapshot
from intellikat.domain.value_objects.job_reference import (
    JobReference,
    Scope,
    SegmentIndexRange,
)
from intellikat.domain.value_objects.sentiment_label import SentimentLabel
from intellikat.infrastructure.persistence.repositories.transcript_reader_pg import (
    TranscriptReaderPg,
)
from intellikat.infrastructure.persistence.unit_of_work import SqlAlchemyUnitOfWork

pytestmark = pytest.mark.integration

try:
    from testcontainers.postgres import PostgresContainer
except ImportError:
    PostgresContainer = None  # type: ignore[assignment,misc]


SHARED_TABLES_DDL = """
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS calls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid()
);

CREATE TABLE IF NOT EXISTS meetings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid()
);

CREATE TABLE IF NOT EXISTS transcript_sessions (
    id                     UUID PRIMARY KEY,
    speaker_user_id        UUID NOT NULL REFERENCES users(id),
    call_id                UUID NULL REFERENCES calls(id) ON DELETE CASCADE,
    meeting_id             UUID NULL REFERENCES meetings(id) ON DELETE CASCADE,
    engine_mode            TEXT NOT NULL,
    model_id               TEXT NOT NULL,
    language_requested     TEXT NULL,
    sample_rate            INTEGER NOT NULL DEFAULT 16000,
    frame_format           TEXT NOT NULL DEFAULT 'pcm_s16le_mono',
    correlation_id         TEXT NULL,
    status                 TEXT NOT NULL DEFAULT 'active',
    started_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at               TIMESTAMPTZ NULL,
    expires_at             TIMESTAMPTZ NOT NULL,
    finalized_segment_count INTEGER NOT NULL DEFAULT 0,
    audio_seconds_total    NUMERIC(10,3) NOT NULL DEFAULT 0,
    error_code             TEXT NULL,
    error_message          TEXT NULL,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS transcript_segments (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id          UUID NOT NULL REFERENCES transcript_sessions(id) ON DELETE CASCADE,
    segment_index       INTEGER NOT NULL,
    speaker_user_id     UUID NOT NULL REFERENCES users(id),
    call_id             UUID NULL REFERENCES calls(id) ON DELETE CASCADE,
    meeting_id          UUID NULL REFERENCES meetings(id) ON DELETE CASCADE,
    text                TEXT NOT NULL,
    language            TEXT NULL,
    confidence          REAL NULL,
    start_ms            INTEGER NOT NULL,
    end_ms              INTEGER NOT NULL,
    words               JSONB NULL,
    engine_mode         TEXT NOT NULL,
    model_id            TEXT NOT NULL,
    segment_started_at  TIMESTAMPTZ NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
"""


@pytest.fixture(scope="module")
def pg_engine():  # type: ignore[no-untyped-def]
    if PostgresContainer is None:
        pytest.skip("testcontainers not installed")
    if os.environ.get("INTELLIKAT_SKIP_INTEGRATION") == "1":
        pytest.skip("integration tests disabled by env")

    with PostgresContainer("postgres:16-alpine") as pg:
        url = pg.get_connection_url().replace("postgresql+psycopg2://", "postgresql+psycopg://")
        engine = create_engine(url, future=True)

        # Seed the shared tables that intellikat doesn't own.
        with engine.begin() as conn:
            for stmt in SHARED_TABLES_DDL.split(";"):
                if stmt.strip():
                    conn.execute(text(stmt))

        # Apply intellikat migrations.
        from alembic import command
        from alembic.config import Config as AlembicConfig

        ini_path = os.path.join(os.path.dirname(__file__), "..", "..", "migrations", "alembic.ini")
        cfg = AlembicConfig(os.path.abspath(ini_path))
        cfg.set_main_option("sqlalchemy.url", url)
        cfg.set_main_option("script_location", os.path.abspath(os.path.dirname(ini_path)))

        # env.py also reads INTELLIKAT_DATABASE_URL — set it for the same value.
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


def _seed_session_with_segments(engine, *, segment_count: int = 3) -> tuple:  # type: ignore[no-untyped-def]
    user_id = uuid4()
    call_id = uuid4()
    session_id = uuid4()
    seg_ids = [uuid4() for _ in range(segment_count)]
    started = datetime.now(UTC)

    with engine.begin() as conn:
        conn.execute(text("INSERT INTO users(id, email) VALUES (:id, 'x@example.com')"), {"id": user_id})
        conn.execute(text("INSERT INTO calls(id) VALUES (:id)"), {"id": call_id})
        conn.execute(
            text(
                """INSERT INTO transcript_sessions
                   (id, speaker_user_id, call_id, engine_mode, model_id, status,
                    started_at, ended_at, expires_at)
                   VALUES (:id, :uid, :cid, 'openai', 'whisper-1', 'ended',
                           :st, :et, :exp)"""
            ),
            {
                "id": session_id,
                "uid": user_id,
                "cid": call_id,
                "st": started,
                "et": started + timedelta(seconds=30),
                "exp": started + timedelta(minutes=10),
            },
        )
        for i, sid in enumerate(seg_ids):
            conn.execute(
                text(
                    """INSERT INTO transcript_segments
                       (id, session_id, segment_index, speaker_user_id, call_id, text,
                        language, confidence, start_ms, end_ms, engine_mode, model_id,
                        segment_started_at)
                       VALUES (:id, :sid, :idx, :uid, :cid, :text, 'en', 0.9,
                               :start_ms, :end_ms, 'openai', 'whisper-1', :ts)"""
                ),
                {
                    "id": sid,
                    "sid": session_id,
                    "idx": i,
                    "uid": user_id,
                    "cid": call_id,
                    "text": f"segment {i}",
                    "start_ms": i * 1000,
                    "end_ms": i * 1000 + 500,
                    "ts": started + timedelta(milliseconds=i * 1000),
                },
            )
    return user_id, session_id, seg_ids


@pytest.mark.asyncio
async def test_reader_round_trip(pg_engine, session_factory):  # type: ignore[no-untyped-def]
    _, session_id, _ = _seed_session_with_segments(pg_engine, segment_count=2)
    reader = TranscriptReaderPg(session_factory)

    session = await reader.get_session(session_id)
    assert session is not None
    assert session.status == "ended"

    segments = await reader.list_segments(session_id)
    assert len(segments) == 2
    assert [s.segment_index for s in segments] == [0, 1]
    assert segments[0].text == "segment 0"


@pytest.mark.asyncio
async def test_writer_persists_sentiment_summary_and_audit(
    pg_engine, session_factory
):  # type: ignore[no-untyped-def]
    user_id, session_id, seg_ids = _seed_session_with_segments(pg_engine, segment_count=2)
    run_id = uuid4()

    sentiments = [
        SegmentSentiment(
            transcript_segment_id=seg_ids[0],
            transcript_session_id=session_id,
            run_id=run_id,
            label=SentimentLabel.POSITIVE,
            confidence=0.9,
            model_id="MarieAngeA13/Sentiment-Analysis-BERT",
            model_revision="abc",
            hf_mode="hosted",
            processed_at=datetime.now(UTC),
        ),
        SegmentSentiment(
            transcript_segment_id=seg_ids[1],
            transcript_session_id=session_id,
            run_id=run_id,
            label=SentimentLabel.NEGATIVE,
            confidence=0.7,
            model_id="MarieAngeA13/Sentiment-Analysis-BERT",
            model_revision="abc",
            hf_mode="hosted",
            processed_at=datetime.now(UTC),
        ),
    ]
    summary = SessionSummary(
        transcript_session_id=session_id,
        run_id=run_id,
        content="A short summary.",
        key_points=["one", "two"],
        model_id="gpt-4o-mini",
        prompt_version="sum-v1",
        prompt_tokens=120,
        completion_tokens=40,
        input_truncated=False,
        processed_at=datetime.now(UTC),
    )
    job = InsightJob(
        run_id=run_id,
        reference=JobReference(
            schema_version="1",
            job_id=uuid4(),
            event_type="transcript.session.closed",
            transcript_session_id=session_id,
            scope=Scope(kind="call", id=uuid4()),
            segment_index_range=SegmentIndexRange.model_validate({"from": 0, "to": 1}),
            speaker_user_id=user_id,
            engine_snapshot=EngineSnapshot(engine_mode="openai", model_id="whisper-1"),
            occurred_at=datetime.now(UTC),
            correlation_id="cid",
        ),
        message_id="msg-1",
        status="completed",
        segments_total=2,
        segments_processed=2,
        segments_failed=0,
        summary_persisted=True,
        started_at=datetime.now(UTC),
        finished_at=datetime.now(UTC),
    )

    uow = SqlAlchemyUnitOfWork(session_factory, hf_mode="hosted")
    async with uow:
        await uow.save_sentiments(sentiments)
        await uow.save_summary(summary)
        await uow.save_job(job)

    with pg_engine.connect() as conn:
        sentiment_count = conn.execute(
            text(
                "SELECT COUNT(*) FROM transcript_segment_sentiments WHERE run_id = :rid"
            ),
            {"rid": run_id},
        ).scalar_one()
        summary_count = conn.execute(
            text("SELECT COUNT(*) FROM transcript_session_summaries WHERE run_id = :rid"),
            {"rid": run_id},
        ).scalar_one()
        audit_status = conn.execute(
            text("SELECT status FROM intellikat_jobs WHERE id = :rid"), {"rid": run_id}
        ).scalar_one()

    assert sentiment_count == 2
    assert summary_count == 1
    assert audit_status == "completed"


@pytest.mark.asyncio
async def test_cascade_delete_on_session(pg_engine, session_factory):  # type: ignore[no-untyped-def]
    _, session_id, seg_ids = _seed_session_with_segments(pg_engine)
    run_id = uuid4()

    sentiments = [
        SegmentSentiment(
            transcript_segment_id=seg_ids[0],
            transcript_session_id=session_id,
            run_id=run_id,
            label=SentimentLabel.NEUTRAL,
            confidence=0.5,
            model_id="MarieAngeA13/Sentiment-Analysis-BERT",
            model_revision=None,
            hf_mode="hosted",
            processed_at=datetime.now(UTC),
        )
    ]
    uow = SqlAlchemyUnitOfWork(session_factory, hf_mode="hosted")
    async with uow:
        await uow.save_sentiments(sentiments)

    with pg_engine.begin() as conn:
        conn.execute(
            text("DELETE FROM transcript_sessions WHERE id = :id"), {"id": session_id}
        )
        remaining = conn.execute(
            text("SELECT COUNT(*) FROM transcript_segment_sentiments WHERE run_id = :rid"),
            {"rid": run_id},
        ).scalar_one()
    assert remaining == 0


@pytest.mark.asyncio
async def test_unique_constraint_enforced(pg_engine, session_factory):  # type: ignore[no-untyped-def]
    _, session_id, seg_ids = _seed_session_with_segments(pg_engine)
    run_id = uuid4()
    base = SegmentSentiment(
        transcript_segment_id=seg_ids[0],
        transcript_session_id=session_id,
        run_id=run_id,
        label=SentimentLabel.POSITIVE,
        confidence=0.8,
        model_id="MarieAngeA13/Sentiment-Analysis-BERT",
        model_revision="rev-1",
        hf_mode="hosted",
        processed_at=datetime.now(UTC),
    )
    uow = SqlAlchemyUnitOfWork(session_factory, hf_mode="hosted")
    async with uow:
        await uow.save_sentiments([base])

    with pytest.raises(Exception):
        async with SqlAlchemyUnitOfWork(session_factory, hf_mode="hosted") as uow2:
            await uow2.save_sentiments([base])
