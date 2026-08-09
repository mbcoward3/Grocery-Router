# Grocery Router

**Decide the week, confirm the list, source the items.**

The household cooks a fraction of what it enjoys. Roughly 32 recipes have been tried and
liked; under the stress of picking a week, about 15 surface. The gap between 15 and 32 is
the product.

## Run the local web app

Python 3.12; the local file-backed path has no third-party dependencies. From the repository root:

```sh
python3 -m gr.web
```

The app binds to `0.0.0.0:8765` and prints two exact URLs: one for this laptop and one for
the iPhone shopping list. Open the phone URL in Safari while the phone and laptop are on
the same network, and keep the laptop awake while shopping. To choose a different port, run
`python3 -m gr.web --port 9000`.

The planning screen sets nights and guests, generates or regenerates a pool, and swaps one
meal without moving the others. The separate phone list has large checkboxes. Local ticks
are written into `weeks/<sunday>.md`; production plans and ticks use the configured durable
store, so both survive a page reload and a restart.

The planner needs the `claude` CLI on `PATH`; without it the week is still planned by code
and the screen says so. A planner call usually takes about a minute.

**Start at [`.scratch/spec/map.md`](.scratch/spec/map.md)** — the destination, the settled
decisions with their costs, the traps this project already paid for, and the open tickets.

### The one rule the code is built around

**The model picks meals. The model never produces a line of the shopping list.**

That is a process boundary, not a convention. The planner subprocess runs with `--tools ""`,
so it has no file access and physically cannot open a recipe file. Its prompt carries
`corpus.md`, `profile.md`, `candidates.md` and `sides.md`, and none of those holds an
ingredient list. Every quantity, conversion, merge and aisle on the list is arithmetic in
`gr/`, checkable line by line against the recipe files.

### Validate and containerize it

```sh
python3 -m unittest discover -s tests    # core, persistence, planner-boundary and web tests
python3 -m gr.audit                      # parse every recipe, print every unresolved line

docker build --pull -t grocery-router:dev .
docker run --rm -e APP_ENV=development -e GROCERY_ROUTER_STORAGE=file \
  -p 8765:8765 grocery-router:dev
```

The image is digest-pinned, runs as non-root UID 10001, installs the PostgreSQL driver from
`requirements.lock` with hashes, and contains no runtime credentials. One planner call costs
roughly $0.15–$0.30. The model only selects meals; every list line is built by deterministic
Python from `recipes/` and `items.md`.

### What lives where

| | |
|---|---|
| `gr/items.py` | The item table and the mis-merge rule. The most expensive knowledge here |
| `gr/parse.py` | One ingredient line in, a quantity and an item out — or a refusal with a reason |
| `gr/recipes.py` | Recipe files, the four yield shapes, and the multiplier each one earns |
| `gr/shoplist.py` | Aggregation, unit reconciliation, staple routing, the unknown channel |
| `gr/planner.py` | The one model call, and every check that refuses to trust it |
| `gr/weekfile.py` | The reviewable generated-week representation and local file helpers |
| `gr/storage.py` | File development store and CockroachDB/PostgreSQL production boundary |
| `gr/session.py` | Plan, build, persist. The seam the web app calls |
| `gr/web.py` | Planning, phone-list, liveness and database-aware readiness surfaces |
| `static/` | The supplied design tokens, locally vendored fonts, CSS and small browser script |
| `gr/notices.py` | The five gaps `profile.md` names, computed from live data |
| `gr/audit.py` | `python3 -m gr.audit` — every ingredient line with no `items.md` row |

Current resolution: **250 of 254 ingredient lines (98.4%)**. All four remaining misses are
correct refusals — two ingredients on one line, an unstated choice between two, and one
malformed source line. A refused line is printed on the list in full, never dropped.

## What is here

| | |
|---|---|
| `corpus.md` | Recipes cooked and liked. 25 rows |
| `recipes/` | The ingredients, verbatim. 27 files, 265 lines |
| `items.md` | The canonical item table — aisle, unit, conversion, match tolerances |
| `profile.md` | The household. One rule: **no claim without a trace** |
| `candidates.md` | Found, not yet cooked |
| `sides.md` | Empty on purpose. Seeding it would invent what this household eats |
| `sources/` | The original inputs the corpus was read out of |
| `decisions.jsonl` | Local-development proposal history; production events are durable rows |
| `weeks/` | Local-development generated weeks; production uses CockroachDB |
| `migrations/` | Explicit, numbered PostgreSQL-compatible schema migrations |
| `deploy/`, `clusters/` | Reusable Kubernetes resources and Flux cluster overlays |
| `docs/platform.md` | Provision, secret, migrate, deploy, observe, rollback and promotion runbook |
| `.scratch/spec/` | The map and its tickets |

## Deployment foundation

GitHub Actions tests and audits every change, builds the container, and on `main` publishes
immutable `sha-<commit>` images to GHCR. It holds no kubeconfig. Flux reconciles the desired
Kustomize overlay from Git. Production requires `DATABASE_URL` with `sslmode=verify-full`;
missing or unavailable storage never falls back to container files. The model-unavailable
fallback remains deterministic and separate from database availability.

See **[`docs/platform.md`](docs/platform.md)** for the complete local Talos Docker and Flux
operator path. Exact prerequisite pins are in `scripts/platform-versions.env`.

## Where the prototype went

**The `prototype` tag does not exist on this remote.** `git tag -l` is empty and
`git ls-remote origin` shows no tags. The 265-line parser it held — described elsewhere in
this repository as the most expensive knowledge in the old codebase — is not recoverable
from here, and the captain's decision is that it is deliberately not being recovered.

It has been re-derived instead, in `gr/parse.py` and `gr/items.py`, and it reaches 98.4% on
the same 254 lines. Any document in `.scratch/spec/` that tells you to run
`git show prototype:shop.py` is describing something that is gone.
