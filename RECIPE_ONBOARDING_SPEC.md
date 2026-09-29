# Grocery Router — Shared Recipe Catalog and Onboarding Pipeline

Status: **approved implementation direction; initial generated batch remains unapproved review data**

Depends on [`V1_SPEC.md`](V1_SPEC.md), [`AUTH_HOUSEHOLDS_SPEC.md`](AUTH_HOUSEHOLDS_SPEC.md),
[`RECIPE_SOURCE_PILOT_SPEC.md`](RECIPE_SOURCE_PILOT_SPEC.md), and
[`RECIPE_INTELLIGENCE_SPEC.md`](RECIPE_INTELLIGENCE_SPEC.md).

## 1. Outcome

Build a repeatable operator pipeline that turns explicitly selected public recipe pages into a
reviewable, semantically rich shared catalog. Authenticated Grocery Router users may browse verified
catalog recipes. A household may add a verified catalog recipe as a trial and later adopt it into its
family collection.

```text
manifested public source
  -> immutable JSON-LD evidence
  -> deterministic source projection
  -> bounded Pi-agent standardization proposal
  -> deterministic validation and grocery proposals
  -> Jev semantic assessment
  -> human review
  -> versioned catalog release
  -> transactional PostgreSQL publication
  -> shared browse -> household trial -> household adoption
```

The initial run is limited to the existing 16 manifested sources and at most 25 selected Recipe
nodes. The current evidence contains 17 candidates. No generated candidate is verified merely
because the pipeline completed.

## 2. Product concepts

### Shared catalog

A global collection visible to every authenticated application user, independent of household
membership. Catalog recipes retain complete reviewed ingredients, instructions, source attribution,
source evidence digests, and approved semantic profiles.

### Family collection

The recipes a household may select into a week. Existing PDF-controlled recipes remain directly
owned by the Coward household. A verified shared recipe enters a household collection through an
explicit membership with state `trial` or `adopted`.

### Trial

A verified shared recipe intentionally added to a household for evaluation. Trial recipes are
selectable for weeks and generate groceries exactly like adopted recipes, but remain visibly marked
as trials.

### Adopted

A verified shared recipe retained as an ordinary family recipe. Promotion from trial to adopted does
not alter recipe quantities or grocery mappings.

### Candidate

A local, Git-reviewable proposal. Candidates may be incomplete, ambiguous, or wrong and are never
runtime recipe truth.

## 3. Authority and coexistence

`sources/Recipes.pdf` and `archive/trueup/recipes.csv` continue to control only the initial corpus.
They are not changed or reinterpreted. The initial audit must continue to report 24 verified recipes
and one excluded recipe.

Shared catalog membership is controlled separately by versioned release manifests under
`catalog/releases/`. Approved catalog documents live under `catalog/recipes/`; generated review
candidates live under `catalog/candidates/`. A combined audit reports initial household membership,
catalog releases, and household catalog memberships as distinct sets.

PostgreSQL is the only runtime source of truth. Runtime code never reads repository documents,
release manifests, source-pilot output, Pi output, Jev output, or `.local-run/`.

## 4. Scope

Included:

- batch processing with a hard maximum of 25 candidates per invocation;
- existing manifest-driven safe fetch and JSON-LD selection;
- immutable source and selected-candidate identity;
- temporary low-cost Pi-agent proposals for recipe standardization;
- strict versioned candidates with field-level provenance;
- deterministic parsing, validation, and grocery-item proposals;
- import-time Jev assessment and compact semantic profiles;
- Markdown review artifacts and batch review index;
- human approval and versioned catalog releases;
- additive, idempotent, transactional release publication;
- shared-catalog and family-collection browsing in the Recipes UI;
- trial and adoption actions for verified catalog recipes; and
- exact grocery generation after a catalog recipe enters a household.

Excluded:

- arbitrary crawling, sitemap discovery, search-engine scraping, or unreviewed URLs;
- browser-based source onboarding and recipe editing;
- automatic verification or approval;
- runtime Pi, Jev, TypeSafe, or other model calls;
- public anonymous catalog access;
- recipe source refresh or revision synchronization;
- household-specific recipe edits and substitution selection; and
- more than 25 candidates in the initial run.

Future discovery may propose URLs for the manifest, but fetching remains explicit, bounded, and
subject to the source-pilot safety contract.

## 5. Source identity and selected JSON-LD provenance

The immutable source identity records:

- source-pilot implementation and format version;
- source manifest ID and manifested URL;
- source-run index SHA-256 and artifact file SHA-256;
- requested/final URL, fetch timestamp, and response-body SHA-256;
- selected candidate index, script index, JSON path, and JSON-LD block SHA-256; and
- canonical-JSON SHA-256 of the selected `Recipe.raw` object.

File SHA-256 uses exact bytes. Canonical JSON recursively sorts object keys, preserves array order,
and removes insignificant whitespace. Candidate construction recomputes all digests and rejects
mismatches. A changed page is a different artifact even when the URL is unchanged.

Selection is explicit. Multiple Recipe nodes produce separate candidates and are not silently
collapsed. Source-pilot artifacts remain ignored local evidence; committed candidates retain the
identity and digests, not complete raw responses.

## 6. Candidate format and provenance

Candidate format version 1 is strict YAML front matter plus a deterministic Markdown review body.
Unknown fields, body drift, invalid states, and duplicate IDs fail audit.

Each candidate contains:

- stable candidate ID and proposed catalog key;
- lifecycle state `draft`, `reviewable`, or `approved`;
- source identity from §5;
- ordered source fields, ingredient sections, instruction sections, yield, and durations;
- standardization proposals and unresolved issues;
- grocery mapping proposals;
- Pi-agent and Jev attachment metadata;
- semantic profile proposals; and
- review decisions.

Every reviewable field preserves a provenance chain:

- `source`: literal JSON-LD value and JSON pointer;
- `deterministic`: parser/normalizer ID and version plus inputs;
- `pi-proposed`: Pi provider, model, prompt digest, input digest, run timestamp, and proposed value;
- `jev-proposed`: catalog/model/question/input provenance and typed answer; and
- `human-approved`: reviewer, approval timestamp, chosen value, and rationale where needed.

Approval appends provenance; it never relabels generated content as source content.

Field states are `resolved`, `ambiguous`, `unresolved`, `not_applicable`, and `approved`. Required
ambiguous or unresolved values block release publication.

## 7. Temporary Pi-agent standardization

The initial implementation invokes isolated Pi print-mode workers using the authenticated local Pi
installation and a low-cost model. The default initial model is
`openai-codex/gpt-5.3-codex-spark` with low reasoning.

Pi receives only a compact selected Recipe projection, the strict output schema, canonical grocery
reference data, and an instruction to treat source text as untrusted data. Workers have no tools,
use no persistent session, and cannot write the repository or database. Output is captured beneath
`.local-run/`, parsed as untrusted JSON, and accepted only after deterministic schema and source
coverage validation.

The command records model and prompt provenance but never reads, prints, or stores Pi credentials.
A worker failure leaves that candidate failed and does not approve or omit it. The pipeline continues
other candidates and exits non-zero after producing a complete batch report.

Pi is an operator adapter, not a runtime dependency. A future native structured-extraction service
may replace it without changing candidate or approval contracts. Automated tests use injected
fixtures and never launch Pi or make model calls.

## 8. Ingredient interpretation

Every source ingredient line is preserved exactly and accounted for once. Candidate structure
includes:

- exact amount or inclusive range;
- one primary arithmetic unit;
- equivalent source-displayed measurements;
- package count/type/size/unit;
- item phrase and preparation;
- optionality and grocery inclusion;
- alternatives and selected baseline proposal; and
- source-note references.

Dual units represent one requirement, never two grocery contributions. Exact source alternatives
remain visible. Because runtime substitutions are deferred, human approval selects one baseline
while preserving alternatives in source text and review notes.

The parser and Pi may propose but cannot approve a missing noun, amount, unit, alternative, optional
state, or non-shopping decision. Source-note references absent from JSON-LD remain explicit and must
be dispositioned during review. No ingredient may disappear because parsing failed.

## 9. Grocery standardization

Catalog candidates use reviewed canonical grocery definitions: stable key, display name, store
section, and shopping mode. A proposal is `match`, `create`, `ambiguous`, or `unresolved`.

Name similarity may suggest candidates but never decides a match. Existing definitions must agree
exactly. New definitions require human approval. Catalog publication stores reviewed default grocery
definitions with the catalog document.

When a verified catalog recipe first enters a household, the adoption transaction materializes a
household recipe snapshot through the existing ingestion/domain validation. Existing household
items are reused only on exact key/name/section/mode agreement; reviewed missing items are created in
the same transaction. This intentional snapshot preserves week and grocery history and avoids
silently changing a household when a future catalog revision appears.

The household membership records the catalog recipe, materialized household recipe, state, and
catalog document digest. This is an explained adoption, not an unexplained corpus copy.

## 10. Instructions, notes, yield, and durations

Source instruction section and step order are preserved. Rewrites require approval and must retain
every ingredient use, duration, temperature, and doneness cue.

Source yield text is retained. Structured values are added only when exact. Source prep/cook/total
durations remain distinct from Grocery Router hands-on/unattended values; translation between them
is an approved interpretation.

Dangling source notes such as `Note 1` are review blockers until transcribed with source-body digest,
resolved into an approved field decision, or explicitly judged non-operative.

## 11. Semantic profiles

Jev runs only during operator import/review work. It receives compact candidate projections and may
not alter recipe truth. Full answers, distributions, confidence, and token usage remain local.

The committed compact profile records all executable intrinsic question keys and versions, typed
answers, reviewer dispositions, catalog digest/version, model, candidate digest, and local assessment
artifact digest. One explicit recipe-level approval approves the reviewed answer set; individual
answers may still be rejected or marked not applicable.

Only accepted profile values are published. Missing or rejected answers remain unknown rather than
invented. Semantic profiles support catalog text/facet browsing and future suggestions; there are no
runtime model calls.

## 12. Approval and release files

Only a human may change a candidate to `approved`. Approval requires reviewer identity and timestamp,
all verification invariants, zero unresolved blockers, complete grocery mappings, dispositioned
source notes/alternatives, and a reviewed semantic profile.

An immutable release manifest records:

- release ID and format version;
- ordered catalog recipe keys;
- candidate, approved document, and profile paths/digests;
- immutable source identity;
- reviewer and approval timestamp; and
- expected publication action.

Raw source pages, source-pilot artifacts, Pi outputs, Jev responses, credentials, and API keys are
never committed.

The initial generated batch may be published to the stable development database as `reviewable` for
visual inspection. Reviewable catalog entries are clearly labeled, cannot be trialed/adopted, and
are never exposed by production catalog queries. Human approval plus a verified release is required
before adoption.

## 13. Database representation

Catalog storage is global rather than household-owned. It stores the approved document using the
same versioned recipe document contract consumed by household ingestion, plus indexed summary fields,
source identity, semantic profile, lifecycle status, release identity, and content digests.

Household catalog membership is tenant-scoped and links:

- household ID;
- catalog recipe ID;
- materialized household recipe ID;
- state `trial` or `adopted`; and
- creation/update timestamps and catalog document digest.

Existing household recipe, grocery, week, and contribution tables remain authoritative for planning
and shopping. Catalog candidates cannot be referenced by weeks. Cross-household membership or
materialized-recipe references are rejected by database constraints.

## 14. Publication, conflicts, and rollback

`catalog-publish` validates the complete release before opening a transaction. It verifies approval,
digests, format versions, source uniqueness, recipe uniqueness, semantic attachment, and all recipe
invariants.

One release transaction inserts catalog records and records the applied release. Any error rolls back
the entire release. Reapplying an identical release is idempotent. Reusing a release ID, recipe key,
source identity, or selected Recipe digest with different content is a conflict.

Publication makes no network, Pi, Jev, or TypeSafe calls. There is no destructive unpublish command
in this phase. Corrections require a new reviewed release; revision semantics are deferred.

Trial/adoption is a separate household-scoped transaction. It validates that the catalog recipe is
verified, materializes or reuses the exact approved snapshot, creates/reuses grocery definitions,
and writes membership atomically. A late failure leaves no recipe, grocery items, or membership.

## 15. API and UI

Authenticated catalog reads are global. Household mutation routes remain explicitly household-scoped.
Required operations are:

- list verified shared recipes with text search and accepted semantic facets;
- fetch shared recipe detail and semantic summary;
- list the household family collection including direct, trial, and adopted recipes;
- add a verified catalog recipe as trial;
- promote trial to adopted; and
- add trial/adopted materialized recipes to a week through existing week operations.

The Recipes screen provides `Family` and `Shared` views. Rows clearly show direct, trial, adopted, or
review-only development state. Shared recipe detail provides `Try with family` or `Adopt` as
appropriate. Reviewable development entries disable these actions and explain that approval is
required. Controls are accessible and mobile-capable.

No UI performs source fetching, candidate editing, model execution, approval, or release publication.

## 16. Operator commands

The command surface must provide equivalents of:

```sh
task catalog-build        # source artifacts -> deterministic packets/candidates, max 25
task catalog-amplify      # isolated cheap Pi workers -> untrusted proposals
task catalog-assess       # import-time Jev assessment
task catalog-review       # deterministic per-recipe Markdown and batch index
task catalog-audit        # strict candidate/release/digest/membership audit
task catalog-release      # build an approved immutable release
task catalog-publish      # migrate + apply release to PostgreSQL, no network/model
task corpus-audit         # report initial and subsequent memberships separately
task corpus-ingest        # reconstruct initial corpus and approved catalog releases
```

A simple development publication command may load reviewable candidates for inspection but must
require an explicit development flag and refuse verified/adoptable status without approval.

## 17. Reproducibility

A fresh database is reconstructed without network or model access:

1. apply Goose migrations;
2. audit/import the unchanged initial PDF corpus;
3. audit/publish approved catalog releases in manifest order; and
4. create household memberships only through explicit fixtures or application actions.

Applied release IDs and digests make reconstruction and production drift auditable. The application
container may include approved corpus/releases, but never local evidence or credentials.

## 18. Acceptance

Tests use fixtures and injected agent/evaluator outputs; automated tests perform no live fetch, Pi,
Jev, or TypeSafe calls. They must cover:

- the real 17-candidate batch and the hard 25-item limit;
- artifact/index/selected-node digest validation and tamper rejection;
- strict candidate parsing and source-line coverage;
- Chicken Marsala dual units, alternatives, malformed garlic line, optional garnish, two instruction
  sections, source-note references, yield, durations, and alcohol semantics;
- duplicate Recipe nodes, overlapping household recipes, and two distinct banana breads;
- unresolved fields/mappings/notes blocking verification and adoption;
- compact semantic-profile provenance and accepted-facet publication;
- reviewable development entries being visible but non-adoptable;
- verified catalog browsing by authenticated users without household ownership;
- trial/adopt membership tenancy and exact materialized document digest;
- transaction rollback, idempotent release publication, and conflict rejection;
- exact grocery output and contribution provenance after trial/adoption;
- initial-ledger integrity and fresh-database reconstruction; and
- no runtime model calls or repository/local-artifact reads.

Run Go formatting, sqlc generation, SQL lint, focused tests, frontend generation/typecheck/tests/build,
and the full suite. PostgreSQL-dependent checks must be reported as infrastructure-blocked rather
than claimed as passed when unavailable.

## 19. Initial review and delivery

The first delivery processes no more than the 17 existing selected Recipe candidates. It creates a
pull request containing code, schemas, UI, tests, deterministic candidates/review artifacts, and no
raw live responses or secrets. Stable development receives the reviewable batch for authenticated
inspection. The reviewer may then approve selected recipes, create a release, and publish it through
the documented command without rerunning network or model stages.
