# Grocery Router — Up Next

Status: **deferred backlog; none of this is v1 scope**  
Current scope: [`V1_SPEC.md`](V1_SPEC.md)

This document tracks the interview's “not yets”: useful capabilities deliberately excluded
because they would muddy v1. Items move out only through a newly scoped phase; this is not an
implementation queue and its ordering is not priority.

## 1. Purpose

This section is the holding area for ideas deliberately excluded because they would muddy
v1. An item being listed here does not authorize schema or UI work now. Moving one into
scope requires a new phase with its own acceptance criteria.

## 2. Planning intelligence

- **Agent-generated weeks.** Replace or augment uniform random selection with OpenAI-backed
  planning through Google ADK for Go.
- **Planner constraints.** Guests, serving targets, busy nights, effort limits, desired
  leftovers, proteins, cuisines, and must-include constraints beyond direct manual editing.
- **History-aware selection.** Recency, repetition avoidance, favorites, breadth, and other
  cooking-history signals.
- **Reasons and feedback.** Planner explanations, acceptance signals, outcomes, and quality
  measurement.
- **Day assignment.** Optional mapping of a pool to calendar days when household behavior
  justifies it.
- **Leftovers.** Model extra batches, repeat meals, and the distinction between recipes and
  eating occasions.
- **Recipe discovery.** Search for recipes outside the family corpus and propose additions.

## 3. Quantity and shopping intelligence

- **Guest/household scaling.** Scale from source yield only after yields and household
  serving behavior are trustworthy.
- **Household baseline batches.** Store a family-adjusted default distinct from the source
  batch when observed rather than guessed.
- **Substitutions and alternatives.** Represent `butter or oil`, raw/cooked/rotisserie
  options, variants, and user-selected defaults without duplicating recipes.
- **Item-specific conversions.** Reviewed mappings such as count-to-volume for produce or
  raw-to-cooked yield. v1 keeps incompatible requirements separate.
- **Purchase-unit conversion.** Translate recipe requirements into practical amounts to
  buy without underbuying.
- **Package optimization.** Choose package counts/sizes from total need.
- **Store-specific products.** SKUs, prices, promotions, availability, aisle locations, and
  carts.
- **Pantry and inventory.** Track what is actually owned and subtract it only from observed
  state, not assumptions.
- **Purchase history.** Use prior shopping behavior to improve inventory or package advice.

## 4. Corpus and recipe management

- **Recipe onboarding UI.** Add linked, pasted, photographed, or typed recipes using the
  same ingestion and verification services built for true-up.
- **Household correction UI.** Edit ingredients, quantities, mappings, times, instructions,
  and source relationships from the application.
- **Agent-assisted onboarding.** Package true-up rules, fixtures, and context as SDK/ADK
  agents rather than relying on a coding-session agent.
- **Agent context tools.** Controlled database queries, exports, and skills that build model
  context directly from SQLite.
- **Subsequent source refresh.** Deliberately compare an approved recipe with a changed
  website; never automatic silent synchronization.
- **Recipe revision history.** Preserve source snapshots, household corrections, approvals,
  and reversions instead of only current truth.
- **Recipe roles and taxonomy.** Mains, sides, breakfasts, lunches, desserts, cuisines,
  proteins, and tags.
- **Search and filtering.** Find recipes by name, role, time, ingredient, or future metadata.
- **Robust image ingestion.** Download, store, transform, attribute, and provide fallbacks
  for recipe images.

## 5. Week and list lifecycle

- **Future-week planning.** Select and manage a Sunday other than the current week.
- **Past-week UI.** Browse, reuse, or compare retained weeks.
- **Automatic weekly rollover.** Create a new week on Sunday rather than waiting for an
  explicit generation action.
- **Sophisticated edit reconciliation.** Warn and resolve when recipe changes collide with
  grocery quantity overrides, removals, or completed lines.
- **Explicit lock/finalize state.** Reconsider only if real use shows that the natural
  plan-then-shop workflow needs enforcement.
- **Custom store-section ordering.** Persist household shopping order rather than
  alphabetical sections.
- **List export and sharing.** Web Share/Apple Notes, text export, or copy workflows.
- **Offline shopping.** Installed PWA/service worker or another offline-capable list.
- **Interactive cooking mode.** Step completion, timers, wake-lock behavior, and cooking
  notes.

## 6. Platform and household expansion

- **Formal backup/export/import.** Replace manual SQLite copying and bootstrap re-ingestion
  with supported runtime-data tools.
- **Generated database releases.** Package and verify a ready-to-run SQLite snapshot from the
  approved Markdown corpus through an explicit release process.
- **Single-binary packaging.** Embed frontend assets in Go if distribution needs it.
- **Docker packaging.** Add only when local distribution or deployment needs it.
- **Hosted deployment.** Choose infrastructure after the local product proves useful; prior
  Talos, Flux, CockroachDB, CloudNativePG, Compose, and VPS designs are not presumed.
- **Authentication.** Required before trusting clients outside the local network.
- **Multiple users/households.** Identity, concurrent household use, tenancy, permissions,
  and attributed feedback.
- **Offline synchronization.** Reconcile edits made by multiple devices or while detached
  from the local server.

## 7. Engineering foundations

These are intentionally deferred from the current true-up slice, but should land before the
application grows several independently configured command paths:

- **Go lint policy.** Add a pinned, best-practices `golangci-lint` configuration aligned as
  loosely as practical with the Google Go Style Guide. Keep the enabled set high-signal,
  document intentional exceptions, and run it through the standard task entry point.
- **Task runner.** Add a root `Taskfile.yml` as the human and agent command surface for format,
  generate, test, lint, migration, corpus audit, corpus ingestion, and eventually frontend
  checks. Tasks should call ordinary underlying tools rather than hide bespoke behavior.
- **Kong CLI and environment contract.** Replace ad hoc subcommand/flag parsing with Kong as
  commands multiply. Define typed command configuration, validation, help, and environment
  bindings in one place so true-up, server, migration, and future runtime state do not develop
  separate configuration conventions.

## 8. Optional polish not blocking v1

- Recipe images beyond easy source images.
- More elaborate fraction typography.
- Advanced animation and transition choreography.
- Additional themes beyond the dark-first direction.
- Reorderable store sections.

## 9. Rules for deferred work

- Deferred features must not leak controls, empty tables, placeholder navigation, or
  explanatory copy into v1.
- v1 may preserve stable IDs and clean relationships that make later migrations possible,
  but it must not implement speculative generalized frameworks.
- Future work should cite the observed v1 limitation that activates it.
- If a deferred feature changes quantity trust or corpus truth, it requires fixtures and
  migration rules at least as strict as the v1 true-up.

---

## 10. Promotion rule

A deferred item moves into an implementation specification only when an observed limitation
or explicit new product decision justifies it. Promotion must define user value, data
requirements, migration impact, failure behavior, and acceptance tests before code begins.
