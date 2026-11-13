# Grocery Router — the product as it is

Status: **current authority for product behavior** (#61). It consolidates what the approved specs
and the decisions in force require today. Where this file and the code disagree, this file states
the approved behavior and the difference is a bug. The original v1 contract is archived at
[`archive/specs/V1_SPEC.md`](archive/specs/V1_SPEC.md); decision numbers (Dnnn) refer to
[`PRODUCT_DECISIONS.md`](PRODUCT_DECISIONS.md).

## 1. What it is

Grocery Router turns one household's reviewed recipes into a meal plan and an exact,
store-ordered grocery checklist.

> **verified recipes → plan meals until the next shopping trip → shop**

- **Who uses it.** One household, the `Coward` household, shared by its members. Each member
  signs in with Google; only configured identities may enter. There is no public signup or
  invitation flow ([`AUTH_HOUSEHOLDS_SPEC.md`](AUTH_HOUSEHOLDS_SPEC.md)).
- **Where.** Deployed as a container at `groceries.matthewcoward.com`, with a development
  instance at `groceries-dev.matthewcoward.com` ([`DEPLOYMENT.md`](DEPLOYMENT.md)). Planning and
  shopping must work well on an iPhone browser. Offline use is not required.
- **What it is not.** It does not discover recipes, infer a pantry, round to packages, scale for
  guests, or use AI at runtime. Its value is trust: every selectable recipe was reviewed, every
  grocery row traces to its recipes, and the list is exact arithmetic over approved data.

## 2. Rules that must hold

1. **Complete before trusted.** A source may be incomplete; a selectable recipe may not. Missing
   information may be filled in with agent help during ingestion, but a person approves each
   recipe before it becomes selectable.
2. **Map once.** Each source ingredient is mapped to one canonical shoppable grocery item during
   ingestion, never while planning or building a list. No model interprets ingredients at runtime.
3. **No inventory inference.** Every shoppable requirement appears on the list. There is no
   pantry, staples, "probably have", or subtraction. A non-purchased input such as tap water may
   be marked non-shopping during approved ingestion only (D033).
4. **No invented purchase advice.** No rounding to packages, raw-to-cooked estimates, or SKUs.
   Package details stated in a recipe stay visible.
5. **Exact arithmetic only.** Amounts in the same unit add exactly. Amounts in different units are
   never converted into each other; they are shown side by side (D054). Item-specific
   approximations are never used. Nothing is ever understated.
6. **Nothing dropped.** No path may silently omit a recipe ingredient or a contribution.
7. **One runtime truth.** PostgreSQL is the only runtime source of recipe data. Approved Markdown
   files under `corpus/recipes/` are a one-way bootstrap; the application never reads or
   synchronizes them. The database keeps each ingredient's source text beside its structured
   interpretation, and the current verified recipe rather than an edit history.
8. **Household isolation.** Members of a household see and change the same data; nobody else can
   read or change it. Household-owned rows carry their household and are tenant-checked by
   composite foreign keys.

## 3. The five screens

- **Plan** (`/`). The current plan: meals for the time until the next shopping trip (D051).
  - With no current plan: "Build your first plan", or, after a plan has closed, "Start a new
    plan" with a link to the plan that ended. Both offer a recipe count (the household default,
    capped by the recipes available) and **Generate plan**.
  - With a plan: its meals, "Since <start date>", and actions to add a recipe, remove one, swap
    one at random, choose a specific replacement, and **Refresh pool** (regenerate at the same
    count). Random swap prefers a recipe not already in the pool.
  - **Start new plan** asks for confirmation, then closes the plan and moves it to History. It
    never generates the next plan.
- **Groceries** (`/groceries`). The current plan's checklist (§5). With no plan it points to the
  last list in History and to starting a new plan.
- **Recipes** (`/recipes`). Every verified recipe alphabetically, with instant name search (kept in
  the URL, so returning from a recipe keeps it), time and yield, and "add to plan".
- **History** (`/history`). Past plans, newest first, with meal and checked-grocery counts. A past
  plan shows its meals and its retained checklist (checked, unchecked, removed, manual, and
  adjusted rows), and offers **Reopen this plan** (§4).
- **Recipe detail** (`/recipes/:id`). Name, optional image, hands-on and unattended time or
  "unknown", source yield, source link, ordered ingredient sections with source-aware quantities
  and preparation, and read-only ordered steps.

Settings (`/settings`) holds the household's preferences (§6).

## 4. Plans

- A plan is created only by an explicit Generate. It starts on today's household-local date and
  has no fixed length. Weeks, rollover days, and automatic closing do not exist (D051).
- A household has at most one current plan. It stays current until someone presses **Start new
  plan** or reopens a past plan.
- **Reopen** makes a past plan current again with its meals and checklist as they were; the plan
  that was current, if any, closes and moves to History.
- A closed plan records the local date it closed (`endsOn`). A past plan cannot be edited unless
  it is reopened.
- **Generation.** Picks the requested number of unique verified recipes uniformly at random. If
  the household avoids repeats (§6), recipes from its most recent past plans are used only when
  too few others remain (D052). Requesting more recipes than are verified fails clearly and
  changes nothing.
- Manual add and specific swap may add a duplicate; every occurrence is one full baseline batch.
  There is no scaling, no assigned days, and no leftovers.
- Changing meals rebuilds the generated part of the list in the same transaction, keeping manual
  rows and carrying checked, removed, and adjusted state by item where unambiguous.

## 5. Grocery list

- **Built from** every recipe occurrence in the plan and its approved ingredients marked for the
  list. Optional ingredients are included, counted, and marked optional.
- **One row per grocery item** (D047). Amounts that can be added exactly are added; others stay
  separate amounts on the row ("2 cups + 1 tsp", largest unit of a dimension first). An
  ingredient with no amount adds "+ more" rather than a second row (D054).
- **Packages** keep count and size ("2 × 14.5 oz cans"); different package sizes stay separate.
- **Presence-only items** such as salt show once without a quantity; their recipes stay visible.
- **Provenance.** Each generated row shows its recipe names. A row more than one recipe needs is
  marked "N recipes" and its amount highlighted (D054). The details sheet lists each recipe's
  source text.
- **Store sections.** Rows are grouped by section in the household's order (default Produce,
  Bakery, Meat, Seafood, Dairy, Pantry, Baking, Spices, Condiments, Wine & Beer, Frozen, Other,
  Non-shopping). Every section can be reordered, saved as a whole, or reset (D044). Sections are
  created by ingestion only; new ones enter before Other.
- **Placement.** Moving a line of a known item saves where the household's store keeps it for
  every later plan; choosing its reviewed section again clears that (D053). An unknown manual
  name enters Other, and the section chosen for it is remembered by name (D045).
- **Editing.** Add a manual item (with suggestions from known items and remembered names); remove
  a row, with a short-lived Undo; set a plan-only quantity, kept apart from the generated amount;
  check and uncheck. These never change recipe data.
- **Checking off.** Tapping a row checks it; the screen updates at once and saving never blocks
  further taps. A checked row stays crossed out until about 1.8 seconds pass without another
  check, then settles into one collapsed Done group. Sections show only what is left. The header
  shows one progress count including Done (D046).

## 6. Household settings

Any member may change them; they apply to the whole household (D049).

- **Time zone** (IANA name). Used to date plans.
- **Recipes per plan** (1-21, default 5). The count a new plan starts with; it does not scale with
  the plan's length.
- **Avoid repeats** (Off or the last 1-8 plans, default Off) (D052).

## 7. Failure behavior

- Too few verified recipes for a unique pool: a clear error, and the plan is not partly replaced.
- An invalid or incomplete recipe cannot be verified, so it cannot be selected.
- Amounts in different units stay separate; nothing is guessed.
- Any failed change rolls back completely; meals and list never fall out of step.
- No current plan: an intentional empty state with the Generate action.
- A missing image or time renders as unknown, without failure.
- A manual item that is just a name becomes a row without a quantity rather than being lost.
- Server errors are logged; the browser receives a generic message, never raw error text.

## 8. Data ownership

- **Household-owned** (each row carries `household_id`): recipes and their sections, ingredients,
  steps, sources and review flags; grocery items; store sections and their order; plans (`weeks`),
  their recipe occurrences, lists, rows, quantities, and contributions; remembered names and item
  placements; household settings.
- **Shared reference data:** units and their exact conversion ratios.
- **Identity:** application users, external identities, sessions, and household memberships with
  roles `owner`, `admin`, or `member`. A household always keeps an owner.
- **Operator only:** the `ingest` schema used by the catalog pipeline
  ([`CATALOG_INGEST_SPEC.md`](CATALOG_INGEST_SPEC.md)). It is never read by the application.

The database enforces: only verified recipes are selectable; a verified recipe has ingredients,
mappings, and instructions; every ingredient has a grocery item and every item a section and
shopping mode; valid fractions, ranges, dates, and ordering; and no contribution without its
recipe occurrence and ingredient. `internal/database/migrations/` is the schema authority.

## 9. Recipes and corpus

- `sources/Recipes.pdf` controlled the initial corpus; linked websites controlled linked recipe
  content at true-up. 24 recipes are approved under `corpus/recipes/`; Chicken and Dumplings is
  explicitly excluded. Evidence is archived under `archive/`.
- Recipe lifecycle: draft → reviewable → verified. Verified recipes cannot be edited in place.
- A household's recipes are loaded once, into an empty database (`corpus-ingest`, or `bootstrap`
  at container start). Changing them afterwards is not yet possible. The proposed recipe library
  ([`RECIPE_LIBRARY_SPEC.md`](RECIPE_LIBRARY_SPEC.md), #55) adds in-app editing and following
  shared catalog recipes.

## 10. Stack and commands

Go, PostgreSQL (Goose migrations, sqlc queries), and React/TypeScript/Vite with TanStack Router
and Query. The API contract is `api/openapi.yaml`, from which frontend types are generated, and
tests check every response against it. Commands and setup: [`README.md`](README.md) and
[`AGENTS.md`](AGENTS.md).

## 11. Not built

- [`UP_NEXT.md`](UP_NEXT.md): deferred ideas, including planning ahead, starting from a past plan,
  household-managed sections, and history-aware selection beyond avoiding repeats.
- [`CATALOG_INGEST_SPEC.md`](CATALOG_INGEST_SPEC.md): the shared recipe catalog pipeline (slices
  1-2 built; slices 3-5 continue per its gates, #62).
- [`SUBSTITUTIONS_SPEC.md`](SUBSTITUTIONS_SPEC.md): household substitution groups (direction only).
- The recipe library: in-app editing and followed catalog recipes (#55, proposed).
