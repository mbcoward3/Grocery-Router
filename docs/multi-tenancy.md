# Multi-tenancy, and what it costs

**This is the restructure `docs/architecture.md` decision 5 named.** That decision said
markdown files stay the source of truth, *no database until SaaS, and that is an explicit
restructure — a real project, not a config change.* SaaS is now the intent, so this is that
project, and this document is written the same way: every decision has its cost next to it,
because a design doc that only lists upsides is a sales document.

Read `docs/architecture.md` first. Nothing here overturns the model boundary, the write
rules, or the reason-is-the-product claim. What changes is who the data belongs to and
where it lives.

`docs/onboarding.md` is the other half: how a household comes to exist at all, and the
three single-household constants that a second family turns into defects.

---

## The thing that is actually in the way — **fixed**

*Step 1 of the sequence is done. `household.py` is the module, `test_household.py` is the
proof, and the account below is left as written because it is why the work happened.*

Not the database. **A data-crossing bug that already existed in the code.**

`app.py` is a `ThreadingHTTPServer`, so it is already concurrent. And which household it
reads is a set of module-level globals:

```python
pantry.ROOT = DEMO                      # app.py, start_demo()
pantry.CORPUS = DEMO / "corpus.md"
pantry._FILE_INDEX = None               # a process-wide slug → filename cache
shop.configure(root)                    # same pattern, different module
```

Two tenants in flight at once and one request repoints `pantry.ROOT` while another is
mid-write. That is not a race that corrupts a byte. **That is household A's corpus written
into household B's file** — a stranger's dinner, and a stranger's stated allergy, in
somebody else's kitchen.

It is safe today for exactly one reason: there is one household, and the globals are set
once at boot. The moment a second tenant exists, this is a P0 that ships silently.

**So the first work is not Postgres. It is deleting the globals**, and it must land before
anything is multi-tenant, including before any staging environment that has two rows in it.

### What is correctly global, and stays

Worth stating so the refactor does not over-reach:

| Global | Keyed by | Verdict |
|---|---|---|
| `acquire.adapters._chosen` | hostname | **Keep shared.** Which search strategy works for `thecountrycook.net` is a fact about that site, not about a household. Sharing it is what stops 500 tenants each walking the five-strategy ladder. |
| `acquire.adapters._robots` | hostname | **Keep shared**, same reason, and it is also the polite answer — one `robots.txt` fetch per host, not per customer. |
| `acquire.adapters._last_hit` | hostname | **Keep shared, and it becomes load-bearing.** It is the courtesy delay. Per-tenant, 500 households would hammer one recipe site simultaneously and each think itself well-behaved. |
| `adapters.kroger._token` | the platform's Kroger app | **Keep shared** while there is one developer account. Becomes per-tenant only if households bring their own. |

The pattern: **tenant state must be threaded, host state must be shared.** Getting that
backwards in either direction is a bug — one leaks data, the other turns the tool into a
scraper.

### How it was fixed, and what it cost

`household.Household` holds one household's root and derives every path off it. It is a
**required first argument** with no default on everything that touches household data —
about forty functions across `pantry`, `shop`, `acquire`, `planner`, `prep` and `review` —
and `app.handle()` resolves it once per request and passes it down.

Three alternatives were available and all were rejected for the same reason:

- **A context variable or a thread-local.** Fixes the concurrency and keeps the property
  that caused it: a function reading a household does not say so. Worse, forgetting to set
  one does not fail — it silently uses whatever was there.
- **A global with a setter.** That is what `shop.configure()` already was.
- **A default of `here()` on each function.** The implicit global returning one call site
  at a time.

All three fail the way the original failed: **not at all, until it is somebody else's
data.** A missing argument is a `TypeError` at the call, which is the gap this project's
own list of traps keeps asking for — *the failure is always a plausible value where there
should have been a gap.*

**The cost, stated honestly:** a wide diff across seven modules and six test files for zero
new behaviour, and one small ugliness — `Meal.file` and `Meal.has_file` were properties and
are now methods taking a household, because a property has nowhere to put the argument.

What it bought is checkable rather than asserted. `test_household.py` runs two households
with disjoint corpora planning, writing and re-reading weeks concurrently, and
`TheGlobalsStayGone` fails the build if any of the nine names comes back. Putting the old
indirection back behind the new signatures makes the concurrency test fail exactly as
predicted — *`alpha: ['Bravo bake', 'Bravo pie', 'Bravo stew'] is not this household's`*.

`app.serving()` is where identity lands in step 4. It returns the one household this
process was started for; making it read a session instead is a change to one function,
because every route below it already takes the household as an argument.

---

## Decisions

| # | Decision | Costs us |
|---|---|---|
| 1 | **Shared process, tenant context threaded through.** One deployment serves every household | A bug can cross tenants, so isolation needs tests that prove it and a second line of defence below the application |
| 2 | **Postgres, run by CloudNativePG on the cluster** | The end of "standard library only" — see decision 5. Backups, migrations and a failover story become ours |
| 3 | **Row-level security, not just careful code** | Every query needs the tenant set on the session; a forgotten `SET` fails closed and loudly rather than returning someone else's data |
| 4 | **One storage interface, two implementations** — files locally and for the browser demo, Postgres for the hosted product | Two implementations of one thing drift. This project has a receipt for that, so **both run the same test suite** or the second one is not allowed to exist |
| 5 | **The standard-library-only property ends here** | It was real and it was defended in CI. A driver is a dependency; see below |
| 6 | **The app becomes stateless** | Nothing on disk means a plain `Deployment` and horizontal scale — but also that a lost database is a lost household, where before it was a git repo |
| 7 | **Free tier is the ranker; the model planner is paid** | The cheap tier has to stay genuinely good, which it already had to be |
| 8 | **Households own their data and can export it whole** | An export path to maintain, and it has to keep working |

### Decisions 2 and 5, revisited: the backend is deferred, on purpose

**CNPG is no longer a hard requirement**, and the reason is worth recording because it
changes what step 3 is allowed to assume.

CNPG itself is not the difficult part — the operator is one manifest and the `Cluster` CR
is short, and it is what removes the failover, backup and minor-upgrade work rather than
adding it. The difficulty on Talos sits underneath: **Talos ships no storage class**, so
local-path, Longhorn, Rook or democratic-csi has to come first, plus an off-cluster bucket
for backups. Both are needed for *any* stateful workload there.

That prompted the honest question of whether Postgres is needed yet at all. Three rungs,
and the product works on any of them:

| Rung | Isolation | Dependency | Gives up |
|---|---|---|---|
| **Files, one directory per household** | a filesystem path | none | transactions; cross-household queries are a directory walk |
| **SQLite, one database per household** | physical — no shared table, so no RLS policy to forget | none, `sqlite3` is stdlib | the app stops being stateless; one writer per household |
| **Postgres via CNPG** | RLS policy on shared tables | a driver, and CI's rule | nothing — this is the top rung |

The middle rung is the interesting one: it **keeps the standard-library property** decision
5 was going to spend, and its isolation story is arguably stronger than decision 3's, since
a database per household has no shared table to leak across. It costs decision 6 — a
`ReadWriteOnce` volume means one writer and `Recreate` rollouts, not horizontal scale. At
homelab scale that constraint does not bind. It is what eventually forces the top rung.

One correction to an earlier claim in this document: the browser demo was said to be
permanently unable to share a backend with production. With SQLite that stops being true —
but not for free. Pyodide ships `sqlite3` as a loadable *package*, not in `python_stdlib.zip`,
so it would mean vendoring one more wheel into a build that is deliberately CDN-free.

**Nothing about this changes step 2.** `store/` is precisely what makes the choice cheap
and late, which is why it comes before any backend. One design consequence is being taken
now: the interface carries an **explicit transaction boundary** even though the files
backend can only no-op it. Designing against files alone would bake in *no transactions*
and make both database rungs awkward later — and that asymmetry is how two implementations
become two behaviours.

### Decision 5, in full, because it is the one that hurts

`.github/workflows/ci.yml` currently **fails the build** if `requirements.txt` or
`pyproject.toml` appears:

> *"A dependency file appeared. The whole project is the standard library."*

That rule was not decoration. It meant the household could run this with a python and a
git clone, and it is why the browser build works at all — Pyodide runs `pantry.py` and
`shop.py` unmodified because there is nothing to install.

Postgres ends it. There is no stdlib driver. Three ways to spend this, and the cost is
different in each:

- **`psycopg3`** — the right driver. Binary wheels, needs `libpq`. Costs the property
  outright.
- **`pg8000`** — pure Python, DB-API compliant, no compiler and no system library. Keeps
  *"no build step, no wheels"* even though it does not keep *"no dependencies"*. Slower,
  and a smaller community to lean on.
- **Speak the wire protocol ourselves.** Not seriously. This is the kind of decision that
  looks principled for a week and is a liability for years.

**Recommendation: `psycopg3`, and change the CI rule to say what is actually true** —
that the *household-facing* path stays dependency-free and the *hosted* path does not.
The check becomes: `pantry.py`, `shop.py`, `onboard.py` and `planner/` must still import
under a bare interpreter. That is a stronger property than the one it replaces, because it
is the one that was actually load-bearing.

**What this costs that is easy to miss:** the Pyodide demo cannot talk to Postgres — there
are no sockets in a browser tab. So the hosted marketing demo stays on the files backend
forever. That is not a workaround; it is decision 4 earning its keep on day one.

### Decision 4, and the trap it is walking toward

Two implementations of one thing drift. `onboard.py` and `shop.py` both parse ingredients
and disagreed about what the item *was* in three of twelve hard cases. That is this
project's most expensive recorded mistake and it is exactly the shape of "a files backend
and a Postgres backend."

The only defence that has ever worked here is **one test suite, run twice.** The ~317
existing tests are written against `pantry`'s functions, not against markdown parsing — so
they can be parameterised over the backend, and a Postgres implementation that does not
pass the files suite is not finished. Anything else is a promise.

---

## Shape

```
store/                One interface, two implementations. The only code that
  __init__.py         knows a Repository exists.
  files.py            Today's behaviour, one directory per household. Local
                      use, the CLIs, and the browser demo.
  postgres.py         The hosted product. CNPG.
  schema.sql          Tables and the RLS policies. Migrations are ordered
                      files, applied by a Job before the app rolls.

household.py          The tenant context. Carries an id, a Repository, and
                      the tier. Threaded through; never a global.
```

`pantry.py` keeps its function names and loses its module paths — `load_corpus()` becomes
`load_corpus(household)`, and every write door (`promote`, `add_candidate`, `add_side`,
`add_profile_claim`, `log`) takes the same first argument. That is a wide diff and a
shallow one, and 331 tests will say immediately if it is wrong.

### Why row-level security and not just discipline

Decision 1 accepts that a bug can cross tenants. RLS is the answer to *accepting* that
rather than hoping:

```sql
ALTER TABLE recipes ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant ON recipes
  USING (household_id = current_setting('app.household')::uuid);
```

Every connection sets `app.household` before it does anything. A query that forgets returns
**zero rows**, loudly, instead of everyone's. The application bug becomes an empty page
instead of a breach, and an empty page is a bug report.

This is worth the ceremony precisely *because* the isolation model is shared-process. Under
pod-per-tenant it would be belt and braces; here it is the braces.

---

## What the household loses, and what has to replace it

`profile.md` opens by saying that **correcting the file is the trust mechanism**, and that
it beats any opaque score. That sentence is why decision 5 in `architecture.md` reversed
*away* from a database in the first place. It has to be answered, not waved at.

Three things replace it, and none of them is optional:

1. **The editors already exist.** §5 of the brief built profile editing in the session, and
   §3 built recipe onboarding. The thing files bought — a person can go and fix it — is now
   a feature rather than a filesystem property. That is most of the debt already paid, and
   it was paid before this decision was taken, which is lucky rather than clever.
2. **History has to be real.** Git was the audit log for free. Postgres does not do that by
   accident, so every household-visible table needs an append-only history and the session
   needs to show it. `decisions.jsonl` already establishes the pattern and the argument:
   *a decision that was not recorded cannot be recovered.*
3. **Export has to be whole and it has to be markdown.** A household must be able to
   download exactly the files this project runs on today and keep using it locally with no
   account. That is decision 8, it is the honest answer to "you took my files away", and it
   is also the best possible answer to a customer asking what happens if you shut down.

---

## Tiers, which the architecture already built

Worth stating plainly because it was not designed for this and fits anyway.

`planner/` is two implementations behind one call, selected on whether a key is present.
`pantry.rank()` is deterministic, costs nothing to run, and is held to the same bar as the
model because it is what the demo and CI use. `planner/model.py` costs money per week.

**That is a pricing boundary that already exists, tested, with a fallback path that is the
best-covered code in the repo.**

| Tier | Planner | Acquisition | Kroger |
|---|---|---|---|
| Free | ranker | manual paste | — |
| Paid | model | search + capture | prices, cart |

And the metering is already being written: `decisions.jsonl` records `planner`, `model` and
`asked` on every proposal, so per-household model spend is a query against data that has
been accumulating since before anyone thought about billing.

**Whose API key** is the question that decides the shape of the business. Platform-owned
means metering and margin and a real cost line. Bring-your-own means near-zero marginal
cost and a much worse first-run experience. This is open and it is a business decision, not
an engineering one.

---

## On the cluster

Talos, CNPG. The app is stateless once the files are gone, which is the quiet win in
decision 6.

```
Deployment          pantry-router, 2+ replicas, no volumes
Service + Ingress   TLS via cert-manager
Cluster (CNPG)      postgres, 3 instances, scheduled backups to object storage
Job                 migrations, as a pre-sync/init step before the rollout
Secret              ANTHROPIC_API_KEY, KROGER_*, session signing key
CronJob             prep — the Step 0 briefing, which is already a real job with
                    one entry point and was written to be triggered by a scheduler
```

Things the app does not have yet and needs:

- **`/healthz` and `/readyz`.** Readiness must check the database, or a rolling deploy
  serves 500s from a pod that has not connected.
- **Graceful shutdown.** `ThreadingHTTPServer` needs to stop accepting and drain on
  `SIGTERM`, or every deploy drops in-flight sessions.
- **Structured logs to stdout** with the household id on every line, so a support question
  is answerable.
- **A real WSGI/ASGI server**, probably. `ThreadingHTTPServer` is in the standard library
  and was the right call for a laptop; it is not what should terminate customer traffic.
  This is decision 5's bill arriving a second time.

`prep.py` as a `CronJob` is a nice fit — `docs/architecture.md` decision 7 said prep is a
real job *"triggered by a button in v1 and a scheduler later"*, and later is now.

---

## Sequence

Ordered by what is unsafe to do out of order.

1. ~~**Kill the globals.**~~ **Done.** `household.py`, threaded through seven modules. The
   317 tests stayed green and 14 joined them, including a two-household concurrency test
   that fails if the module state ever comes back. No behaviour change, no new dependency.
2. **`store/` with the files implementation.** Still no dependency, still no behaviour
   change; the interface exists and the current code is one implementation of it.
3. **`store/postgres.py` + schema + RLS**, and the same suite run against both. This is
   where the dependency lands and where CI's rule changes.
4. **Identity.** Households, members, sessions. `profile.md` already says members are
   attribution and that auth arrives against data that knows who said what.
5. **Cluster.** Manifests, CNPG, probes, migrations job, backups. Single tenant in
   production first, because a deployment that works is worth more than one that scales.
6. **Metering and billing.** Last, and easy, because the log already has the data.

Steps 1 and 2 are the ones that need care. 3 through 6 are ordinary work.

---

## Still open

1. ~~**Whose Anthropic key.**~~ **Answered: the platform's, with no cost constraints for
   now.** So `architecture.md` stands — model cost is not a design driver. What that
   re-opens is different and is recorded in `docs/onboarding.md`: onboarding now spends
   tokens per signup, before a household has earned anything, and nothing caps it. "For
   now" ends the first time somebody runs the interview a thousand times.
2. ~~**What a household is.**~~ **Answered: one kitchen, one login, members stay
   attribution.** No account layer above the household, so `Household.id` is the tenant
   and billing keys on it. An account owning several kitchens stays possible later — it is
   a layer above, not a change to what a household is.
3. **Concurrency between two members**, which decision 5 in `architecture.md` accepted as
   "crude" under files. Postgres makes it fixable and therefore makes it a decision rather
   than an excuse — two people editing one week is now a product question.
4. **What happens to the CLIs.** `shop.py`, `plan.py`, `acquire.py`, `review.py` and
   `onboard.py` all read files. They keep working against the files backend, which means
   the local single-household experience survives intact — but nobody has decided whether a
   hosted customer gets a way to run them.
5. **Whether the demo household stays invented.** It has to. `demo/` exists so a hosted
   deployment does not serve a real family, and that requirement gets stronger with real
   customers, not weaker.
