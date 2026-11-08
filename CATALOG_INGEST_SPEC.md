# Grocery Router — Shared Recipe Catalog Ingest Pipeline

Status: **revision 3, draft for approval.** On approval, slices 1–2 are authorized. Slices 3–5 stay
gated until the conditions in §8 are met.  
Depends on: [`V1_SPEC.md`](V1_SPEC.md), [`AUTH_HOUSEHOLDS_SPEC.md`](AUTH_HOUSEHOLDS_SPEC.md)  
Rationale: [`PRODUCT_DECISIONS.md`](PRODUCT_DECISIONS.md) D039–D043 (proposed)  
Review history: [`archive/reviews/catalog-ingest-astra-2026-11-08.md`](archive/reviews/catalog-ingest-astra-2026-11-08.md)
(non-authoritative)  
Supersedes: the unmerged `feat/recipe-intelligence-pilot` specs. That branch is reference code only.

## 1. Goal

Build an operator-run pipeline that turns public recipe pages into a shared catalog of recipes that
produce **correct grocery lists and complete cooking instructions**. It is mostly deterministic Go;
AI is an offline fallback that may abstain. Publication is automatic, but only under a qualified
release (§6). The long-term target is tens of thousands of recipes. Scale comes last, after the
household confirms the output is useful (slice 5).

Nothing in this phase is visible to users. Adoption into households, suggestions, and the semantic
layer are separate later specs (§10).

## 2. Invariants

These hold in every slice. A change that breaks one is a defect.

1. **Tenancy.** Every new table is shared catalog data (no `household_id`, never written by user
   requests) or operator-only pipeline data in the `ingest` schema.
2. **No runtime AI.** Only operator commands may call a model. No application server package
   imports the AI client.
3. **Household corpus untouched.** The bootstrap validator rejects everything it rejects today.
   `verified` and `approved_on` keep meaning "a person approved this recipe." Catalog trust is
   recorded as release provenance, never as `verified`.
4. **Nothing dropped, nothing invented.** Every source ingredient line has a publishable
   disposition or the recipe is not published. No amount, unit, or item is inferred.
5. **Exact quantities.** Rationals only; no floating point; values fit 64-bit storage.
6. **Immutability.** Evidence, candidate documents, vocabulary snapshots, and release manifests are
   never edited after creation. Changes create new versions.
7. **Politeness.** Robots rules, rate limits, and blocks are obeyed; no bypassing.

## 3. Terms

| Term | Meaning |
| --- | --- |
| Site | One registry entry: attribution name, one or more hosts, crawl policy, measurement convention, eligibility record. |
| Evidence | One immutable fetch: compressed body, metadata, JSON-LD blocks. A re-fetch creates new evidence. |
| Source node | One Schema.org `Recipe` node in evidence, identified by normalized content digest (plus `@id` if present). |
| Source line | One `recipeIngredient` string of a source node, by position. |
| Requirement | One structured need (item, quantity, preparation, optional) with typed provenance (§5.2). |
| Candidate | Immutable interpretation of a source node under a working build. |
| Vocabulary snapshot | Immutable set of catalog grocery items, aliases, and shopping policy. |
| Release manifest | Frozen set of stage, prompt, model, and policy versions, vocabulary snapshot, overrides digest, and site list. |
| Publication set | Immutable mapping of every active catalog key to one candidate document, created per publication. |
| Material error | A difference that changes what is bought or cooked (§6.2). |

## 4. Slices and action items

### Slice 1 — Evaluation set and content contract (authorized on approval)

Tasks:

- [ ] **Split the content contract** in `internal/ingest`:
  - extract shared **recipe content validation** (name, source, yield, durations, sections, units
    exist in the unit vocabulary, rationals fit 64-bit storage);
  - keep bootstrap approval checks (`status: verified`, `approved_on`) in the bootstrap validator,
    which calls the shared validation;
  - add a versioned **catalog content format**: ingredients gain typed provenance, relationship, and
    interpretation state (§5);
  - prove the bootstrap corpus still validates and imports to identical database contents.
- [ ] **Build the evaluation set** under `catalog/eval/`, for 3–4 starting sites:
  - **must-pass** (~10): supported cases that must publish correctly, including the pilot's
    walnut additive and butter "plus more" lines;
  - **expected-reject** (~10): cases that must reject, including the pilot's missing chicken marsala
    notes, "1 garlic", and an incoherent alternative;
  - **regression**: empty at start; every later material defect is added here;
  - **development** (~20): used freely for tuning.
- [ ] **Build the final holdout** (~20 recipes) **outside the repository**, kept by the operator so
  development and coding agents cannot tune to it. A holdout recipe that informs a fix moves to
  regression and is replaced.
- [ ] For each evaluation recipe, record the expected requirements and the expected grocery output,
  produced through the existing aggregation code, plus an instruction-completeness judgment.
- [ ] **Qualification harness** (`catalog-qualify`) that runs a candidate set against the
  evaluation set and reports by site and by line category (§6.3).
- [ ] **Compatibility fixture**: a hand-written catalog-format recipe converts, imports into an
  empty test household, and aggregates with the expected grocery output.

Done when: the bootstrap corpus is unaffected, the harness scores a hand-built candidate set
correctly (including an empty set as a failure), and the evaluation set has its expected outputs.

Owner review: the household reviews the expected outputs for the must-pass, expected-reject, and
holdout recipes (~40 recipes).

### Slice 2 — Bounded acquisition (authorized on approval)

Tasks:

- [ ] `catalog/sites.yaml` registry (§7.1) with the 3–4 starting sites active; the remaining
  pilot origins (13 total) are listed but inactive.
- [ ] `ingest` schema migration for sites, pauses, host rate limits, source pages, fetch attempts,
  evidence, JSON-LD blocks, source nodes, and runs.
- [ ] Port safe fetch and extraction from `internal/recipepilot` on the pilot branch.
- [ ] Bounded sitemap discovery (§7.2), polite fetching (§7.3), persistent pauses (§7.4).
- [ ] Store compressed bodies; replay extraction from stored evidence without network access.
- [ ] Identity layers (§7.5).
- [ ] Commands: `catalog-sites`, `catalog-discover`, `catalog-fetch`, `catalog-site-resume`,
  `catalog-report` (acquisition metrics only).

Done when: a few hundred source nodes are captured from the active sites; an interrupted run
resumes without re-fetching; a paused site stays paused across runs; extraction replays from stored
HTML; all fetch rules have fixture tests with no live network.

### Slice 3 — Interpretation (gated, §8)

Tasks:

- [ ] Deterministic parser implementing the requirement model (§5.1) and states (§5.3).
- [ ] Alternative coherence check per recipe (§5.4); missing-note rejection (§5.5); instruction
  discrepancy detector (§5.6).
- [ ] Grocery resolution: protected identity (§5.7), measurement conventions (§5.8), shopping
  policy (§5.9).
- [ ] Two-level cache (§5.10).
- [ ] AI client on `openai-go` behind an internal interface: structured outputs with abstention,
  request records, spend reservations, ceilings (§5.11). Fake client for all tests.
- [ ] `catalog-build` producing immutable candidates in a working build.
- [ ] **Comparison run**: rules only versus rules plus AI on development and must-pass data,
  reporting correct-recipe rate, rejection reasons, and AI cost per correct recipe. Decide how much
  more parser to build from this, not from rule-coverage percentage.

Done when: all must-pass recipes are correct, all expected-reject recipes reject, fixtures cover
every relationship, state, protected attribute, and adversarial prompt case, and the comparison is
reported.

### Slice 4 — Releases, publication, and recovery (gated, §8)

Tasks:

- [ ] Catalog tables (§9) and vocabulary snapshots.
- [ ] Working build → frozen release manifest → qualification → publication (§6.1).
- [ ] The publication predicate (§6.4), enforced per candidate at publish time.
- [ ] Exact de-duplication and duplicate-candidate groups (§7.6).
- [ ] Overrides in `catalog/overrides.yaml`; invalidation; vocabulary merge and split staged into
  the next snapshot (§6.6).
- [ ] Publication sets, withdrawal, rollback, and block precedence (§6.7).
- [ ] Audits and stop rules (§6.5).
- [ ] Backup and restore procedure (§6.8).
- [ ] Commands: `catalog-freeze`, `catalog-qualify`, `catalog-publish`, `catalog-audit`,
  `catalog-invalidate`, `catalog-vocab`, `catalog-rollback`.

Done when: a release qualifies and publishes the starting sites; a deliberately bad release is
caught by qualification; an audit failure stops a site; rollback restores the previous publication
set without resurrecting blocked content; a restore into an empty database reproduces catalog keys,
documents, and grocery output.

### Slice 5 — Usefulness review, then scale (gated, §8)

Tasks:

- [ ] The household reviews ~20 published recipes as they would plan, shop, and cook from them.
- [ ] Only after that review passes: activate more sites in batches, add the Batch API with
  provider batch IDs and `custom_id` matching, run cross-site concurrency, and evaluate Web Data
  Commons under the same gates, attribution, and eligibility rules.

## 5. Interpretation contracts

### 5.1 Requirement model

One source line yields zero or more requirements. Each requirement records typed provenance:
`ingredient_line` with line position and character span in the original source text.

| Relationship | Example | Result |
| --- | --- | --- |
| `single` | `3 lb boneless beef chuck` | one requirement |
| `additive` | `½ cup walnuts (plus 2 tablespoons for topping)` | two requirements, same item |
| `compound` | `1 tsp each thyme and rosemary` | one requirement per item |
| `alternative` | `2 chicken breasts or 4 thighs` | first option, only if coherent (§5.4) |
| `equivalent` | `2 tbsp / 30g butter` | one requirement; other measure is a note |
| `divided` | `1 cup sugar, divided` | one requirement for the total |
| `heading` | `For the sauce:` | zero requirements; starts an ingredient section |

`plus more ...` without an amount yields a second requirement with an `intentionally_unspecified`
quantity. A line that fits no relationship is `unresolved`; it is never reduced to its most
plausible single item.

### 5.2 Provenance types

- `ingredient_line` — the only requirement source in this phase.
- `instruction_step` is reserved. Instruction text never creates requirements in this phase (§5.6).

### 5.3 Interpretation states

| State | Meaning | Publishable |
| --- | --- | --- |
| `resolved` | fully structured | yes |
| `intentionally_unspecified` | source explicitly gives no amount: closed phrase list in fixtures (`to taste`, `as needed`, `for serving`, `for garnish`, `plus more ...`) | yes |
| `unresolved` | could not determine | no |
| `unsupported` | understood but not representable | no |

`for serving` is a requirement, not optional. Only an explicit source marker makes an ingredient
optional.

### 5.4 Alternatives

Checked per recipe, never cached across recipes. The first alternative is the default only when:
the alternatives share a quantity or each has its own; no instruction refers only to a non-default
option; and the default needs no different preparation in the instructions. Otherwise
`unresolved` with reason `incoherent_alternative`.

### 5.5 Source notes

A recipe whose lines or steps reference a note absent from the JSON-LD is rejected with
`missing_note_dependency`. References are preserved in stored text.

### 5.6 Instruction discrepancy detector

- Detects known grocery names and aliases in instruction text that match no requirement, excluding
  recipe-produced components and non-shopping items.
- Any discrepancy rejects the recipe with `instruction_ingredient_discrepancy`. Nothing is
  auto-added.
- The detector is a measured check, not a completeness guarantee: it cannot find unknown
  ingredients.
- Fixtures cover affirmative (`season with salt`), negative (`do not add salt`), conditional
  (`serve with pepper if desired`), and quantified (`add 2 teaspoons salt`) mentions; the report
  shows the false-rejection rate.
- Whether to auto-add a narrow affirmative pattern is decided after slice 3 data (§11).

### 5.7 Protected identity

Name normalization never removes protected descriptors: fresh/dried/frozen/canned, whole/ground,
sweetened/unsweetened, salted/unsalted, raw/cooked, boneless/bone-in, skin-on/skinless, and stated
low-sodium. The list is operator-maintained. String similarity only builds a shortlist; AI may pick
from it, propose a new item naming its protected attributes, or abstain.

### 5.8 Measurement conventions

Each site declares `us` or `metric`. `cup`, `tbsp`, and `tsp` map to existing US units only for
`us` sites. Metric units are accepted from any site. A `metric` site's `cup` or `tbsp` is
`unsupported` until convention-specific units exist with exact ratios.

### 5.9 Shopping policy

- **Non-shopping:** only unqualified `water` used as a cooking input. A line naming a purchased form
  (`bottled water`, `sparkling water`, `bag of ice`, `ice`) is a shopping requirement. Tests prove
  purchased forms stay on the list.
- **Presence-only:** only items on the operator list (initially salt, black pepper, cooking spray).
  The original numeric requirement is preserved even when displayed as presence-only.
- **New items:** shopping mode is set when the vocabulary snapshot is frozen, from all of the
  item's requirements in the build: `counted` if every quantified requirement is a count or package;
  otherwise `measured`. Items with only unspecified quantities default to `measured`. AI never sets
  shopping mode or non-shopping.

### 5.10 Caching

- **Level 1, context-free parse:** tokens, amounts, units, and spans keyed by the exact normalized
  line, parser version, normalization policy version, and measurement convention. Spans are stored
  against the normalized text with a mapping back to raw text; if a span cannot be rebound to a new
  raw line, the entry is not reused for it.
- **Level 2, name resolution:** name key → item, keyed by vocabulary snapshot, alias policy version,
  shopping policy version, prompt version, and model configuration. Reused globally because it is
  context-free by contract.
- **Recipe-level decisions** (alternative coherence, note and instruction checks) and **AI line
  interpretation** are computed per source node and never reused across recipes.
- Every result keeps its original method (`override`, `rule`, `ai`) on reuse.

### 5.11 AI calls

- `openai-go` behind one internal interface; structured outputs with a fixed schema per task and an
  explicit abstention option; no tools, browsing, or free-form recipe text.
- Results are validated deterministically, must cite spans in the source line, and are rejected if
  they reference text not present.
- Source text is untrusted. Adversarial fixtures embed instructions in recipe text.
- Every request is recorded before it is sent: task, prompt version, model configuration, input
  digest, state, usage, raw output, and validation outcome.
- States: `pending`, `sent`, `succeeded`, `abstained`, `failed_retryable`, `failed_final`,
  `outcome_unknown`. For `outcome_unknown` (crash after sending), the spend reservation is kept and
  one accounted retry is allowed.
- Estimated spend is reserved before sending and reconciled after. Per-run (`--ai-max-calls`,
  `--ai-max-spend`) and monthly (stored in the database) ceilings apply to reserved plus actual
  spend. Reaching one **defers** work; deferral is never rejection.

## 6. Release and publication contracts

### 6.1 Release lifecycle

1. **Working build** — `catalog-build` creates candidates and may stage new provisional items,
   aliases, merges, and splits for the next vocabulary snapshot.
2. **Freeze** — `catalog-freeze` creates an immutable vocabulary snapshot and a release manifest.
3. **Qualify** — `catalog-qualify` scores the manifest's candidates (§6.3). Any change to the
   manifest's contents requires a new manifest and new qualification.
4. **Publish** — a qualified manifest may publish candidates from its listed sites in repeated
   **publication batches**. A candidate that needs a vocabulary item not in the manifest's snapshot
   is deferred to the next release. Adding a site requires a new release.

### 6.2 Material errors

A material error is: a missing or extra requirement; a wrong item or purchased form; a wrong
amount, unit, range, or package; a wrong optional, unspecified, or non-shopping decision; a
missing or wrong instruction step; or a preparation change that changes what is cooked (whole vs.
minced, changed order). Rewording that preserves meaning is cosmetic.

### 6.3 Qualification criteria

A manifest qualifies only if all hold:

- every must-pass recipe publishes with zero material errors;
- every expected-reject recipe is rejected;
- every regression recipe publishes correctly or is rejected; a regression or must-pass recipe that
  previously published and now rejects fails qualification unless the manifest lists it as an
  accepted regression with a reason;
- at least 15 of ~20 holdout recipes publish, with **zero** material errors among them; an empty
  holdout result fails; and
- each active site has at least 3 published evaluation recipes.

This criterion supports no population-level accuracy claim. Results are reported by site and by
category: additive, compound, alternative, AI-created item, non-shopping, presence-only.

### 6.4 Publication predicate

A candidate may publish only if **every** clause holds. Qualification never substitutes for it.

1. Passes shared recipe content validation (§4, slice 1), including known units and storage
   bounds.
2. Detected language is English; otherwise rejected `non_english`.
3. Has at least one requirement and at least one instruction step.
4. Every source line has a publishable disposition: one or more `resolved` or
   `intentionally_unspecified` requirements, or a validated `heading`. No other line may yield zero
   requirements.
5. Every requirement has valid `ingredient_line` provenance with a rebound span and a publishable
   state.
6. Passes alternative, missing-note, and instruction discrepancy checks.
7. Every item is in the manifest's vocabulary snapshot, with store section and shopping mode.
8. Its site is active, eligible, and not paused or blocked, and no override blocks the node.
9. It was built under the qualified manifest it is published with.

### 6.5 Audits and stop rules

- An audit samples 5 published recipes per newly published site and 10 per release; evaluation
  recipes already reviewed count toward the sample.
- Any material error stops publication for that site until a corrected release passes
  qualification with the failing recipes added to regression.
- A missing requirement or a mapping defect affecting more than one recipe stops the whole release
  and triggers rollback.
- A release's recipes are not eligible for any later user-facing phase until its audit is
  complete. Unaudited releases cannot publish a further batch.

### 6.6 Corrections

- Overrides in `catalog/overrides.yaml` (Git) may fix one node's line requirements, map a name key,
  correct an item's name, section, or shopping mode, or block a node or site. Overrides beat rules
  and AI and are part of the manifest digest.
- `catalog-invalidate` lists every candidate that depends on a changed line result, name result,
  policy, item, or override, and rebuilds them in the next working build.
- Merges and splits are staged into the next vocabulary snapshot. Published documents and old
  snapshots are never rewritten; affected recipes switch at the next publication.
- Every vocabulary change is logged with its reason.

### 6.7 Publication sets, withdrawal, and rollback

- Each publication batch creates a new immutable publication set (catalog key → candidate document)
  and switches the active pointer in one transaction. Derived rows are rebuilt from the active set.
- Candidate documents and vocabulary snapshots referenced by any publication set are retained.
- Withdrawal creates a new set without the key.
- When a published recipe fails under a new manifest: a stricter rule withdraws it; a deferral keeps
  the previous document.
- Rollback creates a new set equal to an earlier set, then re-applies all current blocks and
  ineligible sites, so blocked content is never resurrected. Keys that did not exist in the earlier
  set are absent.
- Exact duplicates keep all eligible sources' attributions. If the representative's source is
  withdrawn, the next eligible source becomes representative; if none remains, the key is withdrawn.

### 6.8 Backup and restore

Before every publication, the operator takes a `pg_dump` of the `ingest` and catalog tables. The
restore procedure is documented and tested: restore into an empty database, then verify catalog
keys, active documents, and generated grocery output for the evaluation set. Overrides and the
evaluation set are in Git; the holdout is backed up by the operator. Formal object-store backups
remain an `UP_NEXT.md` item.

## 7. Acquisition contracts

### 7.1 Site registry

`catalog/sites.yaml`, strictly parsed. Per site: stable ID, attribution name, hosts (explicit list;
no implicit subdomains), sitemap URLs, include/exclude patterns, optional stricter interval,
measurement convention, `active` flag, and eligibility record (intended use, review date, notes).
Rejected before any network access: unknown fields, duplicate IDs, a host claimed by two sites,
non-HTTPS URLs, and active sites without an eligibility record. Marking a site ineligible withdraws
its recipes (§6.7).

### 7.2 Discovery

Registered sitemaps only, no link following. Indexes followed to depth 3 with cycle detection.
Sitemap fetch profile: 50,000 URLs and 50 MiB decoded per file. Entries on unregistered hosts are
ignored and counted.

### 7.3 Fetching

- Page fetch profile: default-port HTTPS only, DNS resolved before dialing, private and reserved
  destinations rejected, at most 5 redirects each revalidated and restricted to the site's hosts,
  no proxies, cookies, auth, or scripts, 20-second deadline, 5 MiB decoded body.
- Robots, per actual host, cached 24 hours: 2xx is parsed and obeyed including `Crawl-delay`; 429 and
  5xx or network failure mean fully disallowed and pause the site; other 4xx mean unrestricted.
- Rate limit keyed by **host**, held in PostgreSQL and shared by all commands and request types: one
  in-flight request, at least 5 seconds apart, or longer by registry or `Crawl-delay`.
- User agent names Grocery Router and the public support address.
- Each stage takes a PostgreSQL advisory lock.

### 7.4 Failures and pauses

Timeouts and 5xx on pages and sitemaps retry at most twice with backoff. 403, 429, and
bot-challenge responses on pages and sitemaps pause the site persistently with a reason until
`catalog-site-resume`. Every attempt is recorded; stages resume from recorded state.

### 7.5 Identity

- Fetched page: requested and final URL.
- Source node: normalized content digest plus `@id`; array position is only a location.
- Canonical link: used only if a valid HTTPS URL on the site's hosts; otherwise, or on conflict,
  the final URL.
- Catalog key: assigned at first publication; a later node maps to an existing key by site,
  canonical-or-final URL, and normalized name. Ambiguous matches get a new key and record the
  possible predecessor.

### 7.6 De-duplication

Exact duplicates are candidates with identical requirement **multisets per ingredient section**
(including preparation), yield, and normalized instruction text. They publish once; the
representative is chosen by completeness, then evidence quality, then registry order. Every looser
match is recorded as a duplicate-candidate group and suppresses nothing.

## 8. Gates for slices 3–5

- **Slice 3** starts when slices 1–2 are done and the household has reviewed the evaluation
  expected outputs.
- **Slice 4** starts when slice 3's done-when criteria pass.
- **Automatic publication** is allowed only after slice 4's done-when criteria pass with fixtures
  demonstrating §5.6, §6.3, §5.10, §6.1, §6.7, and §6.4.
- **Scale** (slice 5) starts only after the household usefulness review passes.

## 9. Data model

The physical schema may refine names; it must implement these concepts.

`ingest` schema (operator-only): `sites`, `site_pauses`, `host_rate_limits`, `source_pages`,
`fetch_attempts`, `evidence`, `evidence_jsonld_blocks`, `source_nodes`, `parse_cache`,
`name_results`, `ai_requests`, `ai_spend_ledger`, `working_builds`, `candidates`,
`duplicate_groups`, `catalog_identity_map`, `vocabulary_changes`, `release_manifests`,
`qualifications`, `audits`, `runs`.

Catalog tables (shared, no `household_id`, not writable through the HTTP API):
`catalog_vocabulary_snapshots`, `catalog_store_sections`, `catalog_grocery_items`,
`catalog_grocery_aliases`, `catalog_recipes` (key, attribution, URLs, image URL, source times,
evidence fields `recipeCategory`/`recipeCuisine`/`keywords`/`suitableForDiet`, status),
`catalog_publication_sets`, `catalog_publication_set_members`, `catalog_active_publication`,
`catalog_recipe_grocery_items`.

## 10. Out of scope

Adoption, trial, or any user-visible catalog API or UI; recommendations, embeddings, descriptions,
or taxonomy; runtime AI; automatic re-fetching of published recipes; image download; non-English
recipes; HTML or microdata extraction fallbacks; paywalled or bot-protected content; scaling,
substitutions, nutrition, or purchase-unit conversion; the Batch API before slice 5.

Later specs must settle, not assume: a single-recipe transactional import and grocery-item conflict
policy for adoption; how automated origin is represented in a household and whether it confers
selectability; that a suggested recipe needs a household snapshot before it generates groceries;
and that keeping a recipe in a week shows retention, not that it was cooked or liked.

## 11. Decisions needed (owner: Matt)

Defaults apply if not changed.

| Needed by | Decision | Default |
| --- | --- | --- |
| Slice 2 | Which 3–4 starting sites (all `us` convention) | Budget Bytes, Once Upon a Chef, Natasha's Kitchen, Cookie and Kate |
| Slice 3 | Monthly AI spend ceiling | $20 |
| Slice 3 | `kosher salt`, `sea salt`, `salt`: one item or several | one presence-only item, aliases |
| After slice 3 | Auto-add a narrow affirmative instruction pattern (e.g., salt and pepper) | no |
| Slice 5 | Web Data Commons as a source | evaluate only |
