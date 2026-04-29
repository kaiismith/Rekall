"""Catalog of every event_code emitted by intellikat.

Every log call MUST pass an `Event` constant from this module — never a
string literal. Reviewers can audit log emissions by reading this file.
"""

from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True, slots=True)
class Event:
    code: str
    message: str


# ── lifecycle ───────────────────────────────────────────────────────────────
STARTUP_OK = Event("INTELLIKAT_STARTUP_OK", "intellikat started")
STARTUP_FAILED = Event("INTELLIKAT_STARTUP_FAILED", "intellikat failed to start")
CONFIG_INVALID = Event("INTELLIKAT_CONFIG_INVALID", "configuration validation failed")
GRACEFUL_DRAIN_BEGIN = Event("INTELLIKAT_GRACEFUL_DRAIN_BEGIN", "shutdown drain begin")
GRACEFUL_DRAIN_END = Event("INTELLIKAT_GRACEFUL_DRAIN_END", "shutdown drain end")
FATAL = Event("INTELLIKAT_FATAL", "fatal runtime failure")

# ── infra connections ───────────────────────────────────────────────────────
DB_CONNECTED = Event("INTELLIKAT_DB_CONNECTED", "database connection established")
DB_DISCONNECTED = Event("INTELLIKAT_DB_DISCONNECTED", "database disconnected")
SERVICEBUS_CONNECTED = Event("INTELLIKAT_SERVICEBUS_CONNECTED", "service bus connected")
SERVICEBUS_DISCONNECTED = Event("INTELLIKAT_SERVICEBUS_DISCONNECTED", "service bus disconnected")
HF_MODE_SELECTED = Event("INTELLIKAT_HF_MODE_SELECTED", "hugging face mode selected")
HF_LOAD_OK = Event("INTELLIKAT_HF_LOAD_OK", "hugging face model ready")
HF_LOAD_FAILED = Event("INTELLIKAT_HF_LOAD_FAILED", "hugging face model failed to load")
FOUNDRY_INIT_OK = Event("INTELLIKAT_FOUNDRY_INIT_OK", "foundry client initialised")
FOUNDRY_INIT_FAILED = Event("INTELLIKAT_FOUNDRY_INIT_FAILED", "foundry client init failed")

# ── jobs ────────────────────────────────────────────────────────────────────
JOB_RECEIVED = Event("INTELLIKAT_JOB_RECEIVED", "service bus message received")
JOB_PARSE_FAILED = Event("INTELLIKAT_JOB_PARSE_FAILED", "service bus message body could not be parsed")
JOB_STARTED = Event("INTELLIKAT_JOB_STARTED", "insight job started")
JOB_COMPLETED = Event("INTELLIKAT_JOB_COMPLETED", "insight job completed")
JOB_ABANDONED = Event("INTELLIKAT_JOB_ABANDONED", "insight job abandoned (will be redelivered)")
JOB_DEAD_LETTERED = Event("INTELLIKAT_JOB_DEAD_LETTERED", "insight job dead-lettered (no retry)")
LOCK_RENEW_FAILED = Event("INTELLIKAT_LOCK_RENEW_FAILED", "service bus lock renewal failed")

# ── sentiment ───────────────────────────────────────────────────────────────
SENTIMENT_BATCH_STARTED = Event("INTELLIKAT_SENTIMENT_BATCH_STARTED", "sentiment batch started")
SENTIMENT_BATCH_DONE = Event("INTELLIKAT_SENTIMENT_BATCH_DONE", "sentiment batch done")
SENTIMENT_SEGMENT_FAILED = Event("INTELLIKAT_SENTIMENT_SEGMENT_FAILED", "sentiment segment failed")
SENTIMENT_SKIP_EMPTY = Event("INTELLIKAT_SENTIMENT_SKIP_EMPTY", "sentiment skipped: empty text")
SENTIMENT_TRUNCATED = Event("INTELLIKAT_SENTIMENT_TRUNCATED", "sentiment input truncated")

# ── summarization ───────────────────────────────────────────────────────────
SUMMARY_REQUESTED = Event("INTELLIKAT_SUMMARY_REQUESTED", "summary request sent to foundry")
SUMMARY_DONE = Event("INTELLIKAT_SUMMARY_DONE", "summary received and parsed")
SUMMARY_PARSE_RETRY = Event("INTELLIKAT_SUMMARY_PARSE_RETRY", "summary parse failed; retrying")
SUMMARY_PARSE_FAILED = Event("INTELLIKAT_SUMMARY_PARSE_FAILED", "summary parse failed (final)")
SUMMARY_INPUT_TRUNCATED = Event("INTELLIKAT_SUMMARY_INPUT_TRUNCATED", "summary input truncated to fit")

# ── persistence ─────────────────────────────────────────────────────────────
DB_WRITE_OK = Event("INTELLIKAT_DB_WRITE_OK", "insight rows written")
DB_WRITE_FAILED = Event("INTELLIKAT_DB_WRITE_FAILED", "insight write failed; rolling back")
