# Grocery Router v1 — Product and Implementation Specification

Status: **approved direction, ready for implementation planning**  
Scope settled by product interview.  
Canonical for v1: **this document supersedes the existing `.scratch/spec/`, architecture documents, prototype behavior, and prior implementation decisions.**

---

## 1. Product statement

Grocery Router v1 turns one family's verified recipe corpus into an editable weekly recipe pool and an accurate grocery checklist.

The product has one path:

> **verified corpus → choose a week → generate groceries → shop**

v1 is intentionally basic. It does not discover recipes, infer a pantry, optimize purchases, scale for guests, or use an agent to plan meals. Its value is trust: every selectable recipe has been reviewed, every grocery line can be traced to its recipes, and list generation performs deterministic arithmetic over approved data.

### 1.1 Target user

One household. There are no accounts, identities, permissions, or household boundaries in v1.

### 1.2 Target environment

- The application runs locally on a household computer.
- The planning and grocery interfaces must both work well on an iPhone browser.
- The iPhone may access the local server over the same network.
- Offline use is not required.
- Public hosting and production deployment are not part of v1.

### 1.3 Success condition

v1 is successful when the user can:

1. manually generate the current week's pool with a chosen number of recipes;
2. add, remove, randomly swap, specifically swap, or regenerate recipes;
3. inspect any selected recipe in a usable in-app recipe view;
4. produce a store-sectioned grocery list containing all requirements from the selected recipe batches;
5. understand which recipes contributed to every generated line;
6. make week-only list edits and check items off from an iPhone; and
7. trust that every displayed quantity came from verified recipe data and exact arithmetic, not an inventory or purchase estimate.

---

## 2. Governing principles

### 2.1 Complete the corpus before trusting it

A source does not have to be complete. A selectable recipe does. Missing information may be backfilled with agent assistance during ingestion, but it must be reviewed before the recipe becomes selectable.

### 2.2 Map once, aggregate deterministically

A source ingredient is mapped to its canonical shoppable grocery item during ingestion. No model interprets ingredients while selecting a week or generating a list.

### 2.3 Do not infer inventory

Every recipe requirement appears on the grocery list. v1 has no pantry, staple, “probably have,” or inventory-subtraction behavior.

### 2.4 Do not invent purchase advice

v1 carries approved recipe requirements into the list. It does not round meat to package sizes, estimate raw-to-cooked yields, or suggest SKUs. Package details explicitly present in a recipe remain visible.

### 2.5 Exact conversions are allowed; assumptions are not

Universally exact unit conversions may aggregate. Item-specific approximations may not.

Examples:

- `1 tbsp + 3 tsp` may become `2 tbsp`.
- `1 cup + 8 fl oz` may combine when the units represent the same exact volume dimension.
- `1 onion + 1 cup chopped onion` remains separate.
- Different requested package sizes remain separate.

### 2.6 Preserve evidence, store one current truth

The database keeps source ingredient text beside its approved structured interpretation. SQLite holds the current verified recipe, not a complete edit history.

### 2.7 Keep v1 replaceable and extensible

The schema must not prevent future serving scaling, substitutions, onboarding, agent planning, or agent context queries. Those features are not implemented in v1.

---

## 3. Scope

### 3.1 Included

- One-time true-up of every recipe represented by the canonical PDF.
- SQLite recipe corpus.
- Development-time, agent-assisted recipe completion.
- One-at-a-time human approval of completed recipes.
- Random weekly recipe generation.
- Manual week editing.
- Deterministic grocery aggregation.
- Editable grocery checklist.
- Recipe detail pages with in-app instructions.
- Responsive, polished planning and shopping interfaces.
- Retention of generated weeks in SQLite.

### 3.2 Explicitly excluded

- Runtime AI or Google ADK integration.
- AI-generated meal choices.
- Recipe discovery and recommendations.
- Candidate recipes or membership scoring.
- Search and filtering in the recipe picker.
- Day-by-day meal scheduling.
- Guest counts and household-size scaling.
- Leftovers and repeat-night planning.
- Pantry, inventory, staple, and purchase-history inference.
- Store, price, promotion, package optimization, SKU, and cart integrations.
- Ingredient substitutions and recipe variants.
- Recipe onboarding or editing UI.
- Past-week UI.
- Arbitrary future-week planning.
- Automatic source-website synchronization.
- Full recipe correction/version history.
- Authentication and multi-household support.
- Offline/PWA behavior.
- Docker, cloud deployment, Kubernetes, Talos, and CI/CD deployment work.
- Behavioral scoring, profile claims, planner reasons, and self-improvement analytics.
- Formal database export/import and backup tooling.

---

## 4. Canonical corpus and true-up

### 4.1 Source authority

The original `sources/Recipes.pdf` is canonical for **membership**: it determines which family recipes belong in the initial corpus.

For recipe content:

1. If a PDF entry links to a website, that website is authoritative for ingredients, quantities, and instructions at true-up time.
2. The PDF remains evidence and may contain household-specific information absent from the website.
3. If an entry has no usable website, a similar credible recipe may be found and used as a reference.
4. A backfilled recipe may be adjusted to represent the family recipe.
5. A recipe based on a reference must be marked `adapted-from`; it must not claim to reproduce that source exactly.
6. Existing transcripts, typed inputs, Markdown recipes, and derived tables may assist migration, but none is trusted without comparison to the PDF and authoritative website.

True-up is a one-time snapshot. Recipes do not automatically update when a source website changes later.

### 4.2 Agent assistance

The coding agent may help during true-up by proposing:

- source matches;
- ingredient sections;
- missing ingredients;
- missing quantities;
- canonical grocery-item mappings;
- store sections;
- shopping modes;
- hands-on and unattended times;
- ordered cooking instructions; and
- a concrete default where a source presents alternatives.

Agent proposals are drafts. Every recipe is presented to the user and approved individually before verification.

There is no LLM call in the shipped application's planning or grocery paths.

### 4.3 Future-agent context

True-up work must leave reusable context for later SDK agents. This includes:

- ingestion rules and invariants;
- representative difficult fixtures;
- canonical-item mapping guidance;
- validation queries;
- examples of accepted and rejected transformations; and
- a clear distinction between source evidence, agent proposals, and approved truth.

Future agents should receive controlled queries or exports from SQLite rather than Markdown replicas of the corpus.

### 4.4 Recipe lifecycle

The logical lifecycle is:

1. **Draft** — source evidence has been captured but may be incomplete.
2. **Reviewable** — proposed structured data passes mechanical validation.
3. **Verified** — the user approved the complete recipe.

Only `verified` recipes are eligible for generation or manual addition to a week.

No recipe-management UI is required in v1. The true-up pipeline must nevertheless use reusable application/domain services so a later onboarding UI does not require a second ingestion implementation.

### 4.5 Verification requirements

A recipe may become `verified` only when:

- it has a unique stable identifier and display name;
- source relationship and available source URL are recorded;
- it has at least one ordered ingredient section;
- every ingredient line preserves its source text;
- every ingredient maps to one canonical shoppable grocery item;
- every canonical grocery item has a store section and shopping mode;
- quantity state is explicit, including intentionally unspecified quantities;
- optional ingredients are explicitly marked;
- package count and package size are retained when present;
- ambiguous alternatives have one approved v1 default;
- it has ordered cooking instructions; and
- it passes all ingestion validation.

Hands-on time, unattended time, source yield, source URL, and image may be unknown when genuinely unavailable. Instructions may be rewritten for clarity and consistent step structure.

### 4.6 Execution authority

The operational inventory, source-recovery, drafting, review, audit, and snapshot process is
specified in [`TRUE_UP_PLAN.md`](TRUE_UP_PLAN.md). That plan is a required v1 deliverable,
not optional project management. The verification contract in this section remains the data
authority.

---

## 5. Ubiquitous language

### Recipe

A complete, approved dish that may be selected into a week. v1 does not distinguish mains, sides, breakfasts, or other roles.

### Source relationship

How a recipe relates to its reference: `source` or `adapted-from`.

### Ingredient section

An ordered grouping inside a recipe, such as Sauce, Filling, or Topping.

### Recipe ingredient

One approved ingredient requirement in one recipe. It preserves source text and carries structured quantity, canonical grocery-item mapping, preparation, optional status, and ordering.

### Grocery item

The canonical shoppable identity used for aggregation, such as `yellow onion`, `onion powder`, or `rotisserie chicken`. Purchased form is part of identity where it changes what the shopper must buy.

### Shopping mode

How a grocery item communicates quantity:

- `measured` — show an aggregated exact measurement;
- `counted` — show an aggregated count, retaining package size when applicable;
- `presence-only` — show the item once without a quantity.

Shopping mode is reviewed corpus data, not inferred at list-generation time.

### Week

A manually generated, unordered pool associated with the current Sunday-to-Saturday period.

### Week recipe

One occurrence of a recipe in a week. Initial random generation is unique, but manual actions may add duplicate occurrences. A duplicate represents another full baseline batch.

### Grocery line

One visible row on a week's grocery checklist. It may be recipe-derived or manually added.

### Contribution

The trace from a generated grocery line to a particular week-recipe occurrence and recipe ingredient.

---

## 6. Weekly planning

### 6.1 Creating the week

- A week is created only through an explicit user action.
- Creation automatically assigns the current week's Sunday date.
- The user supplies the number of dinners/recipes.
- No guest count or serving count is requested.
- No week is created automatically on Sunday.
- Future weeks cannot be selected in v1.

### 6.2 Random generation

- Selection is uniform across all verified recipes.
- Selection ignores cooking history and previous weeks.
- Initial results contain no duplicates.
- If the requested count exceeds the number of verified recipes, generation must fail clearly rather than silently duplicate recipes.
- Random selection invokes no model.

### 6.3 Editing the pool

The Week screen supports:

- add a specific recipe;
- remove a recipe occurrence;
- swap an occurrence for another random recipe;
- swap an occurrence for a specific recipe; and
- regenerate the entire pool.

Random swaps should prefer a recipe not already in the pool. A user may manually choose or add a duplicate recipe. Every occurrence contributes one complete baseline recipe batch.

The pool has no assigned days and no leftover semantics.

### 6.4 Natural workflow

The expected workflow is to settle the recipe pool before editing the grocery list. v1 does not introduce a lock/unlock state. If recipes change later, recipe-derived grocery contributions must be recalculated deterministically while preserving unrelated manual list additions and reasonable user state where unambiguous.

Unusual conflicts between list overrides and later recipe edits do not justify a complex reconciliation system in v1.

---

## 7. Recipe quantities and aggregation

### 7.1 Baseline batch

Every week-recipe occurrence uses the approved baseline quantities stored for that recipe. For linked recipes, this normally means the authoritative website's recipe batch. v1 performs no household, guest, yield, or portion scaling.

### 7.2 Quantity states

A recipe ingredient quantity may be:

- an exact numeric amount;
- an exact range, if the approved recipe genuinely states one; or
- intentionally unspecified, such as `to taste`, `as needed`, or `for serving`.

An intentionally unspecified quantity is valid data, not an ingestion failure.

Exact values should be represented without floating-point loss. Human-readable fractions may be used for display, but fraction formatting is not itself a major product feature.

### 7.3 Exact unit conversion

The system may convert and aggregate only through a closed table of exact dimensional conversions.

Required behavior:

- compatible exact units for the same grocery item combine;
- incompatible units remain separate;
- item-specific size or yield assumptions are prohibited;
- source package sizes are not converted into supposedly equivalent purchase packages; and
- conversions must never silently understate a requirement.

### 7.4 Packages

A package requirement preserves both count and size.

Example:

- Display `2 × 14.5 oz cans diced tomatoes`, not merely `2 cans`.
- `1 × 28 oz can` and `2 × 14.5 oz cans` remain separate requirements.

### 7.5 Presence-only items

For a presence-only item such as salt, source measurements do not produce a misleading purchase quantity. Multiple contributions consolidate into one line, such as `Salt`, with all contributing recipes available in detail.

This is a shopping-display rule, not an assumption that the household already owns the item.

### 7.6 Preparation

Preparation belongs to the recipe ingredient, not normally to grocery-item identity.

Example:

- `yellow onion, diced` and `yellow onion, sliced` map to the same grocery item and aggregate when their units are compatible.
- Preparation details remain visible in expanded contribution details and recipe pages.

Purchased form remains part of identity where necessary. `Raw chicken breast`, `cooked chicken`, and `rotisserie chicken` may be separate grocery items.

### 7.7 Optional ingredients

Optional ingredients:

- are included in the generated list by default;
- remain part of totals;
- are visibly marked optional; and
- retain their contributing recipe traces.

---

## 8. Grocery checklist

### 8.1 Generation

The grocery list is computed from all week-recipe occurrences and their approved ingredients.

It must:

- include every recipe ingredient;
- perform no pantry subtraction;
- aggregate by canonical grocery item when permitted;
- preserve incompatible requirements as separate lines;
- group lines by store section;
- show recipe provenance through expansion; and
- never silently drop a contribution.

### 8.2 Store sections

- Every canonical grocery item receives a manually reviewed store section during true-up.
- Sections display alphabetically in v1.
- Custom ordering is deferred.
- A manually added line goes into `Other`.

### 8.3 User editing

The user may:

- add an arbitrary grocery line;
- remove a line from that week's list;
- change a generated quantity for that week; and
- mark a line done or not done.

These operations modify only the week. They never update recipe data.

A weekly quantity override should be represented separately from the underlying generated requirement so later recipe changes do not corrupt the recipe or erase provenance. The expected workflow minimizes edits made before recipe selection is finished; v1 should prefer simple predictable behavior over a large conflict-resolution subsystem.

### 8.4 Completion display

Completed items remain in place, visibly crossed out and slightly de-emphasized. They do not disappear or move to another screen.

### 8.5 Provenance display

The default list remains compact. Recipe names and per-recipe contributions appear behind an expand action on each generated line.

---

## 9. Recipe detail

Every verified recipe has an in-app detail page containing:

- recipe name;
- optional image;
- hands-on time or `unknown`;
- unattended time or `unknown`;
- informational source yield when known;
- source relationship and source link when available;
- ordered ingredient sections;
- source-aware ingredient quantities and preparation; and
- clear, read-only ordered cooking steps.

Interactive cooking mode, timers, step checkboxes, and recipe editing are deferred.

---

## 10. User interface direction

### 10.1 Screens

v1 has exactly three product screens:

1. **Week**
2. **Groceries**
3. **Recipe detail**

Past weeks may be retained in SQLite but are not exposed in v1.

### 10.2 Visual direction

Linear is the visual inspiration, specifically:

- high visual polish;
- compact information density;
- dark presentation;
- restrained, useful transitions;
- clear hierarchy;
- professional, “means business” tone; and
- little instructional copy.

The goal is not to clone Linear or build a generalized design system.

### 10.3 Responsive behavior

Both planning and shopping must work well on iPhone, not merely shrink a desktop layout. Requirements include:

- touch-friendly targets;
- no hover-only actions;
- no essential horizontally overflowing tables;
- readable quantities and labels;
- stable controls while asynchronous actions complete; and
- grocery completion usable one-handed where practical.

Desktop must remain polished and information-dense.

### 10.4 Accessibility

The implementation must provide:

- semantic controls and labels;
- keyboard operation on desktop;
- visible focus states;
- sufficient contrast;
- meaningful loading, empty, and error states; and
- reduced-motion support for nonessential transitions.

---

## 11. Logical data model

The physical schema may refine names and normalization, but it must implement these concepts and invariants.

### 11.1 `recipes`

Required concepts:

- stable ID;
- unique display name;
- lifecycle status (`draft`, `reviewable`, `verified`);
- source relationship (`source`, `adapted-from`);
- source URL and attribution when available;
- optional image reference;
- optional source yield text/structured value;
- optional hands-on duration;
- optional unattended duration; and
- deterministic display ordering where needed.

Only verified recipes may be referenced by new week-recipe rows.

### 11.2 `recipe_ingredient_sections`

- stable ID;
- recipe ID;
- section name; and
- position unique within the recipe.

Every verified recipe has at least one section. A source with no heading receives an approved default section.

### 11.3 `recipe_ingredients`

- stable ID;
- section ID;
- position unique within the section;
- exact source text;
- canonical grocery-item ID;
- structured quantity state;
- exact amount or range when numeric;
- unit when applicable;
- package count, package size, and package-size unit when applicable;
- preparation text;
- optional flag; and
- any concise approved display note.

A structured numeric value must use an exact representation, not binary floating point.

### 11.4 `recipe_instruction_sections` and `recipe_steps`

Instructions must preserve ordering. The schema should permit optional instruction sections without requiring them in the v1 UI.

Each step has:

- recipe/section relationship;
- position; and
- approved instruction text.

A verified recipe has at least one step.

### 11.5 `store_sections`

- stable ID;
- unique name.

Custom household ordering may be added later without changing grocery-item identity.

### 11.6 `grocery_items`

- stable ID;
- unique normalized identity;
- household-facing display name;
- store-section ID; and
- shopping mode (`measured`, `counted`, `presence-only`).

Canonical identity must not collapse meaningfully different purchased forms.

### 11.7 `units`

The unit model must distinguish:

- unit identity;
- display symbol/name;
- dimension, such as mass, volume, or count; and
- exact conversion ratio to the dimension's base unit where conversion is allowed.

There is no approximate conversion in the v1 unit graph.

### 11.8 `weeks`

- stable ID;
- unique Sunday start date;
- creation/update timestamps.

Weeks are retained. v1 only exposes the explicitly generated current week.

### 11.9 `week_recipes`

- stable occurrence ID;
- week ID;
- recipe ID;
- display position.

The same recipe ID may appear more than once through separate occurrence IDs.

### 11.10 `shopping_lists`

One current grocery list belongs to a week. Recipe changes rebuild its generated contributions without modifying recipe truth.

### 11.11 `shopping_lines`

Required concepts:

- stable ID;
- shopping-list ID;
- optional grocery-item ID for generated/canonical lines;
- display name;
- store section;
- structured displayed quantity when present;
- origin (`generated` or `manual`);
- optional state;
- removed state;
- completion state;
- deterministic display position; and
- week-only quantity adjustment/override where applicable.

### 11.12 `shopping_line_contributions`

Every generated quantity is traceable through contribution rows containing:

- shopping-line ID;
- week-recipe occurrence ID;
- recipe-ingredient ID; and
- exact contributed quantity/package requirement.

Manual lines have no recipe contribution.

### 11.13 Integrity rules

The database and domain layer must enforce:

- no selectable non-verified recipe;
- no verified recipe without ingredients, mappings, and instructions;
- no recipe ingredient without a canonical grocery item;
- no grocery item without a store section and shopping mode;
- valid exact-quantity denominators and ranges;
- unique ordering within each ordered parent;
- Sunday week dates;
- no cross-dimension unit conversion;
- no contribution without its week-recipe and ingredient; and
- no silent deletion of contribution data during aggregation.

SQLite foreign keys must be enabled on every connection.

---

## 12. Implementation architecture

### 12.1 Backend

- Go.
- SQLite database.
- Goose for versioned schema migrations.
- sqlc for typed SQL access.
- Domain and aggregation behavior in ordinary Go, independent of HTTP handlers.
- No runtime model or ADK dependency.

### 12.2 Frontend

- React and TypeScript are the preferred default.
- Vite is an acceptable build tool unless implementation discovery identifies a simpler equivalent.
- Generated or shared API types should prevent avoidable frontend/backend drift.
- The frontend may run separately during development; whether release assets are embedded in the Go binary is an implementation convenience, not a product requirement.

### 12.3 Runtime data

SQLite is the sole runtime recipe source of truth. There is no synchronized Markdown recipe store.

Git contains:

- Goose migrations;
- application code;
- true-up rules and fixtures;
- preserved raw corpus evidence; and
- a manually refreshed SQL corpus snapshot for bootstrap/recovery.

The snapshot is not automatically synchronized and is not a second live data source.

### 12.4 Local operation

A documented local command must start the usable application. Docker is not required. The server may trust clients that can reach it on the local network.

---

## 13. Application service boundaries

Exact URLs are implementation details, but the backend must expose clear operations for:

### Corpus reads

- list all verified recipes;
- fetch one recipe with sections, ingredients, times, source, and steps.

### Week operations

- create/regenerate the current week with a requested recipe count;
- add a specific recipe occurrence;
- remove an occurrence;
- random-swap an occurrence;
- specifically swap an occurrence; and
- return the current week.

### Grocery operations

- return the current grouped checklist;
- expand a line's contributions;
- add a manual line;
- remove/restore a line as supported by the UI;
- apply a week-only quantity override; and
- toggle completion.

Mutating operations must be transactional. A failed week mutation must not leave recipe occurrences and grocery contributions out of sync.

### Ingestion services

Development tooling must reuse domain validation for:

- creating/updating draft recipes;
- validating completeness;
- mapping grocery items;
- promoting a recipe to verified; and
- producing the review representation shown during one-at-a-time approval.

---

## 14. Failure behavior

- Insufficient verified recipes for a requested unique random pool: show a clear error and do not partially replace the week.
- Invalid or incomplete recipe: reject verification; it cannot appear in selection.
- Unsupported unit combination: retain separate grocery requirements; do not guess.
- Database mutation failure: roll back the complete operation.
- Missing current week: show an intentional empty state with the generation action.
- Missing optional image or time: render without failure.
- Manual list value that cannot be parsed structurally: permit a simple presence-only manual line rather than losing the entry.

No trusted path may respond to bad data by silently omitting a recipe ingredient.

---

## 15. Verification and acceptance

### 15.1 Schema tests

Tests must prove migration from an empty database and all integrity rules in §11.13.

### 15.2 Ingestion tests

Fixtures must cover at least:

- linked authoritative recipe;
- PDF-only ingredient list;
- adapted reference recipe;
- missing quantity approved as unspecified;
- optional ingredient;
- package count and size;
- ingredient alternative resolved to a default;
- same grocery item with different preparation;
- distinct purchased forms that must not merge; and
- attempted verification with an unmapped ingredient.

### 15.3 Aggregation tests

Tests must cover:

- exact same-unit addition;
- exact cross-unit conversion;
- incompatible-unit separation;
- package-size separation;
- presence-only consolidation;
- optional-state propagation;
- duplicate recipe occurrences;
- recipe contribution traces;
- no pantry subtraction;
- manual list additions; and
- week-only overrides that do not change recipes.

Representative outputs should be stored as readable golden fixtures.

### 15.4 Week tests

Tests must prove:

- explicit creation only;
- current-Sunday assignment;
- requested count;
- uniqueness of generated results;
- uniform/history-blind eligibility;
- manual duplicate support;
- add/remove/random swap/specific swap/full regeneration; and
- transactional grocery recomputation.

Statistical randomness quality beyond ordinary unbiased selection is not a v1 concern.

### 15.5 Browser acceptance

Automated browser coverage must verify the critical mobile-capable path:

1. open an empty current week;
2. generate a chosen number of recipes;
3. add, remove, and swap recipes;
4. open recipe details;
5. view grouped groceries;
6. expand recipe contributions;
7. add and remove a manual item;
8. override a weekly quantity; and
9. check an item off and observe its de-emphasized state.

The same path must remain usable at an iPhone-sized viewport.

### 15.6 Corpus acceptance

Before v1 is considered complete:

- every PDF recipe has an explicit disposition;
- every included recipe was reviewed individually;
- all included recipes are verified;
- no verified ingredient is unmapped;
- all grocery items have store sections and shopping modes;
- all recipes have ordered instructions;
- exact source/package quantities have been checked against the authoritative source; and
- a complete-corpus validation command exits successfully.

---

## 16. Delivery plans and deferred work

The product and implementation contract ends here; execution mechanics and future scope live
in focused companion documents:

- [`TRUE_UP_PLAN.md`](TRUE_UP_PLAN.md) — corpus inventory, one-at-a-time review, migration,
  audit, and snapshot plan.
- [`REPO_CLEANUP_PLAN.md`](REPO_CLEANUP_PLAN.md) — branch preservation, active-tree cleanup,
  scaffold, and consolidation plan.
- [`UP_NEXT.md`](UP_NEXT.md) — all interview “not yets,” explicitly outside v1.
- [`PRODUCT_DECISIONS.md`](PRODUCT_DECISIONS.md) — durable rationale and revisit conditions.
- [`archive/interviews/v1-scope-interview.md`](archive/interviews/v1-scope-interview.md) —
  non-authoritative raw interview transcript for nuance recovery only.

The first four documents are required project authorities but do not expand the v1 product
scope. The transcript is historical evidence and must not be loaded as routine agent context.

---

## 17. Definition of done

Grocery Router v1 is done when:

1. the active repository contains one coherent Go/React/SQLite application and current documentation;
2. a fresh clone contains no contradictory prototype, deployment, or superseded-spec instructions;
3. obsolete PRs and work branches have been closed or archived according to `REPO_CLEANUP_PLAN.md`;
4. Goose can create the schema from an empty SQLite database;
5. sqlc provides typed access for application queries;
6. the PDF inventory ledger gives every canonical recipe an explicit disposition;
7. every included recipe has been individually reviewed and resolved;
8. every selectable recipe satisfies the verification contract;
9. the complete-corpus audit passes and a fresh database can load the committed SQL snapshot;
10. reusable ingestion rules, fixtures, validation queries, and future-agent context remain after true-up;
11. the three required screens work well on desktop and iPhone;
12. random generation and all specified pool edits work without AI;
13. grocery generation includes every recipe requirement with exact, traceable aggregation and no inventory assumptions;
14. weekly list editing and completion work without modifying recipe truth;
15. schema, ingestion, aggregation, service, and critical browser tests pass; and
16. a user can perform the complete weekly workflow without consulting repository files or understanding the database.

Anything beyond this definition requires a new scoped phase.
