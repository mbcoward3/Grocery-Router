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

## Nothing is deployed, and here is what that cost to find out

An earlier version of this page said Hugging Face Spaces was free for this and was **the
only option that had never asked for a payment method**. That was wrong. Creating the Space
returns:

```
402 Payment Required
Static Spaces are free for everyone, but hosting Gradio and Docker
Spaces on free cpu-basic requires a PRO subscription.
```

Verified against a real account: a **static** Space creates fine, a **Docker** Space does
not. So this whole page's premise — that a container is the easy free path — held for
neither of the hosts it recommended without qualification.

Left here rather than deleted, because the deploy machinery below all works and the
constraint is worth knowing before anyone spends an evening on it.

## Hugging Face Spaces — needs PRO ($9/mo)

Everything is ready for it. `./deploy-space.sh <user>/<space>` creates the Space, generates
the front matter Spaces reads its config from, and pushes 532KB. The Dockerfile already
defaults `PORT` to 7860 to match. It is one command the moment the account can host Docker.

## Render — free tier, unverified

Detects the Dockerfile with no config. New → Web Service → point at the repo → instance
type **Free**: 750 instance-hours a month, spins down after 15 minutes without traffic and
takes about a minute to wake, no persistent disk. Their docs do not say plainly whether a
payment method is required up front — **I have not tested this one**, and given the
paragraph above, treat it as a claim to check rather than a recommendation.

## The option that is genuinely free

A **static** Space, with [Pyodide](https://pyodide.org) running `pantry.py` and `shop.py`
in the browser. CPython compiled to WebAssembly, so it is the real code rather than a
port — which matters, because a JavaScript reimplementation would be a second copy of the
business logic to keep honest, and this project already found what that costs when two
ingredient parsers disagreed on three of twelve hard cases.

Not built. It needs `app.py`'s subprocess call to `shop.py` replaced with a direct function
call, and the markdown files written into Pyodide's in-memory filesystem at startup.
Neither is hard; nobody has needed it yet.

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
