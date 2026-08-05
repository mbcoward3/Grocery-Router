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

## The demo household is invented

A hosted URL serves `demo/`, not the real household. Two different reasons, and only the
first one is about privacy:

**The real `profile.md` names who lives here, their ages and a food allergy.** That should
not be on a public URL. `demo/` supplies its own profile, corpus and candidates; the recipe
files and `items.md` are shared, because those are published recipes with nothing private
in them and duplicating them would only let them drift. The handwritten family recipe card
is dropped from the demo corpus for the same reason as the profile.

**The invented household is also the better demo.** The real corpus has no cooking history
yet, which is the one state where the tool has least to say — every reason comes out as
*no record of cooking this yet*. The demo household has a fabricated year of it, so a
visitor sees the thing actually working:

```
Tuna melt              not cooked in 11 months
Enchiladas             the only beef in the week so far
Pork loin and rice     low active — a night a bad day cannot break
Chicken noodle soup    not cooked in 7 months
Sheet pan fajitas      new here — widening the corpus is the other half of the job
```

`test_pantry.py` enforces the separation rather than trusting it — the demo profile may not
contain a real member's name, the demo corpus may not contain the family recipe, and both
files have to say outright that they are invented.

## What a visitor will see

Honest about the empty spots, because they are the point rather than an oversight:

- **Last week is seeded and unanswered on purpose.** Four meals, three proven and one
  candidate. Answering them stamps last-cooked dates, promotes the candidate into the
  corpus, and moves the metrics — which is the mechanism the whole product rests on, and it
  cannot be shown any other way.
- **Four recipes still read as dormant.** Correct, and the point: the gap between what gets
  reached for and what the household likes *is* the product, so that number is the one
  worth watching.
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
