# Pantry Router, hosted.
#
# No dependencies to install - the whole project is Python's standard library,
# which is why this image is four lines of real work and no requirements file.
#
#   docker build -t pantry . && docker run -p 7860:7860 pantry
#
# PANTRY_DEMO=1 makes every write land in a scratch copy instead of the files
# in this image. It is on by default here because a hosted URL means strangers'
# hands on the corpus, and app.py refuses to serve the real one publicly.

FROM python:3.12-slim

WORKDIR /app
COPY . .

ENV HOST=0.0.0.0 \
    PORT=7860 \
    PANTRY_DEMO=1 \
    PYTHONUNBUFFERED=1

# Cache the briefing once at build time so the first visitor sees a full page
# rather than an empty card. It is regenerated on demand from inside the app.
RUN python3 prep.py || true

EXPOSE 7860
CMD ["python3", "app.py", "--no-open"]
