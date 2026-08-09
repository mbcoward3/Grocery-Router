# syntax=docker/dockerfile:1.7
FROM python:3.12.13-slim-bookworm@sha256:4766d8b510c428e595d74b9cc5bbb2fae8e26316fffb4adc89908d79aacd58a2

LABEL org.opencontainers.image.source="https://github.com/mbcoward3/Grocery-Router" \
      org.opencontainers.image.description="Deterministic meal planning and grocery routing"

ENV PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1

WORKDIR /app

RUN groupadd --gid 10001 grocery-router \
    && useradd --uid 10001 --gid grocery-router --no-create-home --shell /usr/sbin/nologin grocery-router

COPY requirements.lock ./
RUN python -m pip install --no-cache-dir --no-compile --require-hashes -r requirements.lock

COPY --chown=grocery-router:grocery-router gr/ ./gr/
COPY --chown=grocery-router:grocery-router migrations/ ./migrations/
COPY --chown=grocery-router:grocery-router static/ ./static/
COPY --chown=grocery-router:grocery-router recipes/ ./recipes/
COPY --chown=grocery-router:grocery-router weeks/ ./weeks/
COPY --chown=grocery-router:grocery-router items.md corpus.md candidates.md profile.md sides.md decisions.jsonl ./

USER 10001:10001
EXPOSE 8765

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
  CMD ["python", "-c", "import urllib.request; urllib.request.urlopen('http://127.0.0.1:8765/health/ready', timeout=2).read()"]

CMD ["python", "-m", "gr.web", "--host", "0.0.0.0", "--port", "8765", "--root", "/app"]
