# Putting it somewhere you can click it

The whole project is Python's standard library, so there is nothing to install and the
image is a `COPY` and a `CMD`. Any host that runs a container will run this.

## Two things the code does to make hosting safe

**It refuses to serve your real corpus publicly.** `app.py` binds `127.0.0.1` by default,
and if you point it at a public interface without `--demo` it exits rather than start:

```
$ ./app.py --host 0.0.0.0
Refusing to serve the real corpus on a public interface.
  Pass --demo to run against a scratch copy, or keep --host 127.0.0.1.
```

**Demo mode is not a stub.** `--demo` (or `PANTRY_DEMO=1`) copies the household's files
into a scratch directory at boot and points everything there. Planning, feedback,
promotion, the decision log and the grocery list all run for real — a visitor can promote a
candidate and watch the corpus change — and none of it touches the repo. **Start over**
puts it back.

The Dockerfile sets `PANTRY_DEMO=1`, so anything built from it is safe to expose.

## Hugging Face Spaces — free, no card

The path of least friction, and the only one on this page that has never asked me for a
payment method.

1. huggingface.co → **New Space** → SDK **Docker**, template **Blank**, visibility public.
2. Push this repo to the Space's git remote.
3. Add this to the top of the Space's `README.md` — Spaces reads its config from there:

```yaml
---
title: Pantry Router
emoji: 🧺
colorFrom: yellow
colorTo: red
sdk: docker
app_port: 7860
---
```

That's it — it builds and gives you a public URL. The `PORT` default in the Dockerfile is
already 7860 to match.

## Render — free web service

Detects the Dockerfile with no config. New → Web Service → point at the repo → instance
type **Free**. Free services spin down after inactivity, so the first visit after a quiet
spell takes about thirty seconds. Check whether a card is required at signup; that has
changed more than once.

## Anything else that runs a container

```sh
docker build -t pantry .
docker run -p 7860:7860 pantry
```

Fly, Railway, Cloud Run, a Pi on your desk. `PORT` and `HOST` are read from the
environment, which is what most platforms set for you.

## What a visitor will see

Honest about the empty spots, because they are the point rather than an oversight:

- **No "Last week" section on the first visit.** There is no previous week. It appears
  once one exists.
- **Metrics read 24 dormant, 0 cooked.** Correct. Nothing has been cooked yet, and the gap
  between what gets reached for and what the household likes *is* the product — so the
  number that should move over time starts where it honestly starts.
- **The sale lines say `demo`.** There is no Kroger integration. The staleness and
  open-question lines beneath them are real, computed off the corpus.

## What state survives

Nothing, deliberately. Every host on this page has an ephemeral filesystem, and demo mode
starts from a fresh copy each boot anyway. A visitor's session is real while they are in
it and gone afterwards, which is the right behaviour for a demo and the wrong behaviour for
a household — which is why the real one runs on your machine, against files you own, in git.

That is a property of the storage decision in [`architecture.md`](architecture.md), not an
accident, and it is the first thing that has to change if this ever becomes a hosted
product rather than a hosted demo.
