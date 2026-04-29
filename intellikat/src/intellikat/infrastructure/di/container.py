"""Composition root.

The ONLY place that knows the full graph of ports, adapters, and use cases.
Every other module receives its dependencies via constructor injection.
"""

from __future__ import annotations

from dataclasses import dataclass
from typing import TYPE_CHECKING

from sqlalchemy import Engine
from sqlalchemy.orm import Session, sessionmaker

from intellikat.application.use_cases.process_insight_job import ProcessInsightJob
from intellikat.domain.value_objects.prompt_version import make_prompt_version
from intellikat.infrastructure.config.prompt_loader import PromptLoader
from intellikat.infrastructure.llm.foundry_client import FoundryClient
from intellikat.infrastructure.llm.summarizer_adapter import FoundrySummarizerAdapter
from intellikat.infrastructure.messaging.servicebus_consumer import ServiceBusJobConsumer
from intellikat.infrastructure.persistence.engine import (
    build_engine,
    build_session_factory,
)
from intellikat.infrastructure.persistence.repositories.transcript_reader_pg import (
    TranscriptReaderPg,
)
from intellikat.infrastructure.persistence.unit_of_work import SqlAlchemyUnitOfWork
from intellikat.infrastructure.sentiment.factory import build_sentiment_analyzer

if TYPE_CHECKING:
    from intellikat.domain.ports.insight_writer import InsightWriter
    from intellikat.domain.ports.job_consumer import JobConsumer
    from intellikat.domain.ports.sentiment_analyzer import SentimentAnalyzer
    from intellikat.domain.ports.summarizer import Summarizer
    from intellikat.domain.ports.transcript_reader import TranscriptReader
    from intellikat.infrastructure.config.settings import Settings
    from intellikat.infrastructure.logging.logger import ContextLogger


@dataclass
class Container:
    settings: Settings
    logger: ContextLogger
    engine: Engine
    session_factory: sessionmaker[Session]
    sentiment_analyzer: SentimentAnalyzer
    summarizer: Summarizer
    foundry_client: FoundryClient
    consumer: JobConsumer
    process_use_case: ProcessInsightJob
    reader: TranscriptReader

    async def aclose(self) -> None:
        # Best-effort cleanup of every adapter that owns external state.
        await self.foundry_client.close()
        if hasattr(self.sentiment_analyzer, "close"):
            await self.sentiment_analyzer.close()  # type: ignore[func-returns-value]
        await self.consumer.close()
        self.engine.dispose()


def build_container(settings: Settings, logger: ContextLogger) -> Container:
    engine = build_engine(settings)
    session_factory = build_session_factory(engine)

    sentiment_analyzer = build_sentiment_analyzer(settings)

    foundry = FoundryClient(settings)
    prompt_template = PromptLoader(settings.prompt_dir).load(settings.prompt_version)
    summarizer = FoundrySummarizerAdapter(
        foundry=foundry,
        prompt_template=prompt_template,
        deployment=settings.foundry_summary_deployment,
        prompt_version=make_prompt_version(settings.prompt_version),
        logger=logger,
    )

    reader = TranscriptReaderPg(session_factory=session_factory)

    def writer_factory() -> "InsightWriter":
        return SqlAlchemyUnitOfWork(session_factory, hf_mode=sentiment_analyzer.hf_mode)

    process = ProcessInsightJob(
        reader=reader,
        analyzer=sentiment_analyzer,
        summarizer=summarizer,
        writer_factory=writer_factory,
        prompt_version=make_prompt_version(settings.prompt_version),
        failure_ratio_threshold=settings.sentiment_failure_ratio_threshold,
        max_summary_input_chars=settings.max_summary_input_chars,
        logger=logger,
    )

    consumer = ServiceBusJobConsumer(settings=settings, logger=logger)

    return Container(
        settings=settings,
        logger=logger,
        engine=engine,
        session_factory=session_factory,
        sentiment_analyzer=sentiment_analyzer,
        summarizer=summarizer,
        foundry_client=foundry,
        consumer=consumer,
        process_use_case=process,
        reader=reader,
    )
