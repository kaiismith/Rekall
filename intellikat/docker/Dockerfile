# syntax=docker/dockerfile:1.7

# ── shared base ──────────────────────────────────────────────────────────────
FROM python:3.12-slim AS base
ENV PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1
RUN apt-get update && apt-get install -y --no-install-recommends \
        curl ca-certificates libpq5 \
    && rm -rf /var/lib/apt/lists/*
RUN pip install --upgrade pip uv

# ── deps stage ───────────────────────────────────────────────────────────────
FROM base AS deps
WORKDIR /app
COPY pyproject.toml uv.lock* ./
# Install only base dependencies; the `local` extras image layer extends this.
RUN uv sync --frozen --no-dev || uv sync --no-dev

# ── source ───────────────────────────────────────────────────────────────────
FROM deps AS src
COPY src ./src
COPY migrations ./migrations
COPY config ./config

# ── target: hosted (default; small image, ~150 MB) ───────────────────────────
FROM src AS intellikat-hosted
RUN useradd --uid 1000 --shell /usr/sbin/nologin intellikat \
    && chown -R intellikat:intellikat /app
USER 1000:1000
EXPOSE 8090
HEALTHCHECK --interval=15s --timeout=5s --retries=3 \
    CMD curl -fsS http://127.0.0.1:8090/healthz || exit 1
ENTRYPOINT ["uv", "run", "python", "-m", "intellikat.main"]
CMD ["--mode", "worker"]

# ── target: local (heavy image with torch + weights baked in, ~3 GB) ─────────
FROM src AS intellikat-local
RUN uv sync --no-dev --extra local
COPY scripts/download_hf_model.sh /tmp/download_hf_model.sh
RUN bash /tmp/download_hf_model.sh \
        "${HF_MODEL_ID:-MarieAngeA13/Sentiment-Analysis-BERT}" \
        /var/cache/intellikat/hf
RUN useradd --uid 1000 --shell /usr/sbin/nologin intellikat \
    && chown -R intellikat:intellikat /app /var/cache/intellikat
USER 1000:1000
EXPOSE 8090
HEALTHCHECK --interval=30s --timeout=10s --retries=3 \
    CMD curl -fsS http://127.0.0.1:8090/healthz || exit 1
ENTRYPOINT ["uv", "run", "python", "-m", "intellikat.main"]
CMD ["--mode", "worker"]
