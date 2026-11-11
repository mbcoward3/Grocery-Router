# Grocery Router — Product Decision Record

Status: **durable rationale for the v1 direction and later approved phases**  
Behavioral authority: [`PRODUCT.md`](PRODUCT.md)  
Historical source: [`archive/interviews/v1-scope-interview.md`](archive/interviews/v1-scope-interview.md)

This document records why v1 has its current shape. It is not a second specification: when
wording conflicts, `PRODUCT.md` governs behavior. The interview archive is non-authoritative
and exists only to recover nuance. D036 supersedes earlier SQLite-specific decisions.

## Authority order

1. `PRODUCT.md` — the product as it is (the original v1 contract is archived at
   `archive/specs/V1_SPEC.md`).
2. `UP_NEXT.md` — explicitly deferred scope.
3. This record — rationale and revisit conditions.
4. Completed true-up ledger and raw interview archives — historical evidence only.

## Decision index

| ID | Title | Status |
| --- | --- | --- |
| D001 | Reset instead of repairing the current application | Partly superseded by D036 (SQLite rebuild; now PostgreSQL) |
| D002 | Reduce the product to corpus → week → groceries | Partly superseded by D038 (screen count; History added) |
| D003 | Random planning now; agent planning later | Partly superseded by D042 (Google ADK in revisit condition) |
| D004 | The week is an unordered pool | Active |
| D005 | Direct week manipulation | Active |
| D006 | No guest or serving scaling in v1 | Active |
| D007 | Include every ingredient | Partly superseded by D033 (non-shopping inputs such as water are excluded) |
| D008 | Preserve recipe requirements; do not estimate purchases | Active |
| D009 | Exact conversions only | Active |
| D010 | Shopping modes are reviewed item data | Active |
| D011 | Canonical grocery mapping happens during ingestion | Active |
| D012 | Purchased form matters | Active |
| D013 | Preserve package sizes | Active |
| D014 | Optional ingredients stay visible | Active |
| D015 | The PDF controls membership; websites control linked content | Active |
| D016 | Source completeness and recipe completeness are different | Active |
| D017 | Agent assistance is allowed only before approval | Active |
| D018 | Distinguish sourced from adapted recipes | Active |
| D019 | Preserve source text and structured interpretation | Partly superseded by D036 (SQLite storage; now PostgreSQL) |
| D020 | SQLite is the sole runtime recipe truth | Partly superseded by D036 (SQLite runtime truth; now PostgreSQL) |
| D021 | True up recipes one at a time | Active |
| D022 | Store full in-app recipes | Partly superseded by D036 (SQLite storage; now PostgreSQL) |
| D023 | Keep list provenance compact but available | Active |
| D024 | Weekly list edits do not alter recipes | Active |
| D025 | No explicit week lock | Active |
| D026 | Current week is manually created | Partly superseded by D038 (past weeks are now shown in History) and D051 (a plan lasts until the household starts a new one) |
| D027 | Responsive Linear-inspired UI | Active |
| D028 | Go, SQLite, Goose, and sqlc | Partly superseded by D036 and D042 (SQLite, deferred hosting; ADK rationale) |
| D029 | No authentication or offline mode locally | Partly superseded by AUTH_HOUSEHOLDS_SPEC.md and D036 (no auth, local-only) |
| D030 | Keep an injectable, approved Markdown corpus | Partly superseded by D036 (SQLite ingest target; now PostgreSQL) |
| D031 | Keep deferred ideas visible but out of v1 | Active |
| D032 | Suggestions do not replace recipe requirements | Active |
| D033 | Non-purchased inputs remain explicit | Active |
| D034 | Use a TanStack-first TypeScript frontend | Active |
| D035 | Use the three-stroke basket/list identity for v1 | Active |
| D036 | Replace SQLite with PostgreSQL managed by CloudNativePG | Active |
| D037 | Make the verified corpus directly browsable | Partly superseded by D038 (four-screen count) |
| D038 | Expose retained weeks as read-only history | Active |
| D039 | Shared catalog plus household-owned copies | Active |
| D040 | Release-qualified publication for catalog recipes | Active |
| D041 | AI runs offline only | Active |
| D042 | OpenAI through the official Go SDK | Active |
| D043 | Suggested recipes enter the week marked, and join when kept | Direction only (not implemented) |
| D044 | Shopping follows the household store walk | Partly superseded by D045 and D046 (completed-row sinking, section folding, Undo on check, Hide done and its default) |
| D045 | Remember where unknown items go; Hide done owns folding | Partly superseded by D046 (Hide done control and section folding) and D053 (known items can be placed too) |
| D046 | Plain rows, one Done group, undo only for removal | Active |
| D047 | One checklist row per grocery item; look-alike items need a decision | Active |
| D048 | Substitution is a household choice over frozen groups (proposed) | Direction only (not implemented) |
| D049 | Household preferences live per household and are edited in the app | Active |
| D050 | The household chooses its rollover day, time, and mode | Superseded by D051 |
| D051 | A plan lasts from one shopping trip to the next and can be reopened | Active |
| D052 | A household can avoid recipes from its last few plans | Active |
| D053 | A household can place a known item in its own store section | Active |
| D054 | Amounts in different units are shown side by side, not converted | Active |

---

## D001 — Reset instead of repairing the current application

**Status:** Partly superseded by D036 (SQLite rebuild; now PostgreSQL)

**Decision:** Treat all existing implementation, derived recipe Markdown, deployment work,
and superseded specifications as disposable. Preserve Git history and raw corpus evidence.

**Why:** The current Python UI exposed poor quantity behavior, weak pantry assumptions, and a
product model that had accumulated contradictory implementation and specification choices.
Patching it would preserve the wrong foundations.

**Consequence:** v1 is a clean Go/React/SQLite rebuild. Legacy code may be consulted as a hint
but defines no behavior.

**Revisit when:** Never as a wholesale choice. Individual algorithms may be re-derived from
legacy evidence only when they satisfy the new tests and data contract.

## D002 — Reduce the product to corpus → week → groceries

**Status:** Partly superseded by D038 (screen count; History added)

**Decision:** v1 has one path: select known recipes, form an unordered week, and generate a
traceable grocery checklist.

**Why:** Reliable recipe data and quantities are the prerequisite for every more ambitious
feature. Discovery, learning, inventory, and agent reasoning obscure whether the core works.

**Consequence:** The core workflow uses Week, Groceries, Recipes, and Recipe detail screens. The Recipes screen was added by D037 after household use exposed a corpus-access gap.

**Revisit when:** The complete workflow is trusted in real household use.

## D003 — Random planning now; agent planning later

**Status:** Partly superseded by D042 (Google ADK in revisit condition)

**Decision:** Generate the initial week uniformly at random from verified recipes.

**Why:** Random selection delivers recall without pretending the application has trustworthy
history, preferences, or model judgment. It is free and easy to regenerate.

**Consequence:** Initial results are unique and history-blind. Users can regenerate or edit
them manually. Runtime AI planning is excluded.

**Revisit when:** An OpenAI-backed planner through Google ADK for Go receives a separately
scoped contract and evaluation plan.

## D004 — The week is an unordered pool

**Status:** Active

**Decision:** Do not assign recipes to days.

**Why:** The household's weeks are dynamic; day labels would create false precision and
additional maintenance.

**Consequence:** A week contains recipe occurrences and a Sunday identity, not a day grid.

**Revisit when:** Actual use demonstrates value in day assignment.

## D005 — Direct week manipulation

**Status:** Active

**Decision:** Support add, remove, random swap, specific swap, and full regeneration.

**Why:** Generation is only a starting point. The user must be able to express a direct intent
without understanding planner mechanics.

**Consequence:** Manual duplication is allowed even though generated suggestions are unique.
Each occurrence contributes another complete batch.

**Revisit when:** These controls prove insufficient in use.

## D006 — No guest or serving scaling in v1

**Status:** Active

**Decision:** Use one approved source-baseline batch per recipe occurrence.

**Why:** Asking for guests implies the system can scale quantities correctly. The existing
corpus and yields do not yet justify that confidence.

**Consequence:** Number of dinners means number of recipe occurrences. Source yield is stored
for later but does not change list arithmetic.

**Revisit when:** Recipe yields and household serving behavior have been verified and scaling
receives dedicated fixtures.

## D007 — Include every ingredient

**Status:** Partly superseded by D033 (non-shopping inputs such as water are excluded)

**Decision:** Generate groceries from every approved recipe ingredient, including salt,
spices, oil, and optional ingredients.

**Why:** The prior attempt to assume what the household already owned was inaccurate. A short
list is worse than an explicit complete list.

**Consequence:** There is no pantry, staple, or probably-have classification. Optional items
remain visibly optional.

**Revisit when:** Inventory is based on observed state rather than assumptions.

## D008 — Preserve recipe requirements; do not estimate purchases

**Status:** Active

**Decision:** v1 reports approved recipe requirements and explicit package details. It does
not round, pad, or translate them into advice about how much to buy.

**Why:** Poor purchase estimates were a direct failure in the prototype. Accuracy begins with
correct recipes, not clever formatting.

**Consequence:** `1.3 lb` remains `1.3 lb` if that is the approved requirement. Raw-to-cooked
yields and package optimization are deferred.

**Revisit when:** Purchase conversion rules can be evidenced and tested without silent
underbuying.

## D009 — Exact conversions only

**Status:** Active

**Decision:** Aggregate through universally exact unit conversions; keep incompatible or
item-specific quantities separate.

**Why:** Converting tablespoons to teaspoons is arithmetic. Converting an onion to cups is an
assumption about size and preparation.

**Consequence:** The list may show separate requirements for one grocery item when no exact
conversion exists.

**Revisit when:** A reviewed item-specific conversion model is deliberately introduced.

## D010 — Shopping modes are reviewed item data

**Status:** Active

**Decision:** Each canonical grocery item is `measured`, `counted`, or `presence-only`.

**Why:** Recipe measurements are not always useful shopping quantities. Salt should appear as
`Salt`, not a summed teaspoon quantity, without implying that it is already owned.

**Consequence:** Shopping mode is assigned during true-up, never inferred during list creation.

**Revisit when:** Store products or inventory provide a stronger purchase representation.

## D011 — Canonical grocery mapping happens during ingestion

**Status:** Active

**Decision:** Every recipe ingredient maps to a canonical shoppable item before the recipe is
verified.

**Why:** Selection-time LLM mapping would make identical weeks nondeterministic and allow
silent ingredient merges. Mapping is corpus work, not planning work.

**Consequence:** Week selection and list generation require no model. A verified recipe cannot
contain an unmapped ingredient.

**Revisit when:** Do not move mapping to runtime. Future onboarding may automate proposals but
must retain verification.

## D012 — Purchased form matters

**Status:** Active

**Decision:** Related foods remain separate grocery items when they are bought differently.

**Why:** Raw chicken breast, cooked chicken, and rotisserie chicken are not interchangeable
shopping instructions even when a recipe could eventually support substitutions.

**Consequence:** Purchased form participates in canonical grocery identity. Substitution
relationships are deferred.

**Revisit when:** A first-class substitution model is scoped.

## D013 — Preserve package sizes

**Status:** Active

**Decision:** Keep package count and package size in grocery requirements, and do not merge
different requested sizes.

**Why:** `2 cans` is insufficient shopping guidance when the source specifies `2 × 14.5 oz
cans`.

**Consequence:** Different package-size requirements remain separate even when their total
weights could be added mathematically.

**Revisit when:** Package optimization can choose actual products.

## D014 — Optional ingredients stay visible

**Status:** Active

**Decision:** Include optional ingredients by default and mark them visibly optional.

**Why:** Omitting them silently creates an incomplete list; treating them as mandatory loses
source meaning.

**Consequence:** Optional state survives aggregation and remains traceable to recipes.

**Revisit when:** The user requests per-week optional-item inclusion controls.

## D015 — The PDF controls membership; websites control linked content

**Status:** Active

**Decision:** `sources/Recipes.pdf` determines which recipes belong. For linked entries, the
website inspected during true-up controls ingredients, quantities, and instructions.

**Why:** The PDF is the family's canonical recipe inventory, while the linked page is the
stronger recipe source.

**Consequence:** Existing derived corpus counts and Markdown files cannot add or remove
membership. Source URL and check date are recorded.

**Revisit when:** The household explicitly changes corpus membership after v1 onboarding
exists.

## D016 — Source completeness and recipe completeness are different

**Status:** Active

**Decision:** A source may be sparse; a selectable database recipe must be complete.

**Why:** Several genuine family recipes are only grocery lists in the PDF. Rejecting them would
lose real corpus membership, while selecting them unchanged would produce unusable recipe
pages and uncertain groceries.

**Consequence:** Missing structure, ingredients, quantities, and instructions may be
backfilled, reviewed, and approved.

**Revisit when:** Never as an invariant. Future onboarding should preserve the same distinction.

## D017 — Agent assistance is allowed only before approval

**Status:** Active

**Decision:** The coding agent may propose missing recipe facts during true-up, including
entire ingredients and quantities. The user approves each recipe before verification.

**Why:** Backfilling is necessary, but model output must not pass as household truth or enter a
shopping list without review.

**Consequence:** Recipes move through draft, reviewable, and verified states. Only verified
recipes are selectable.

**Revisit when:** Future onboarding agents may change the interface, not the approval boundary.

## D018 — Distinguish sourced from adapted recipes

**Status:** Active

**Decision:** Record whether an approved recipe is a direct source or adapted from a reference.

**Why:** A similar online recipe can fill gaps for a PDF-only family recipe, but the final
household version should not be represented as an exact copy.

**Consequence:** Attribution remains honest while permitting useful completion.

**Revisit when:** A richer source/revision model is introduced.

## D019 — Preserve source text and structured interpretation

**Status:** Partly superseded by D036 (SQLite storage; now PostgreSQL)

**Decision:** Store every original ingredient line beside exact structured quantity, package,
preparation, optional state, and canonical grocery mapping.

**Why:** Structured fields enable arithmetic; source text makes that interpretation auditable.
One without the other is either unusable or untrustworthy.

**Consequence:** Ingestion and review show both representations. SQLite stores one approved
current truth rather than a complete correction history.

**Revisit when:** Revision history is scoped.

## D020 — SQLite is the sole runtime recipe truth

**Status:** Partly superseded by D036 (SQLite runtime truth; now PostgreSQL)

**Decision:** Keep one strict, approved Markdown file per recipe as the Git/bootstrap corpus,
then ingest it one-way into SQLite. The running application reads only SQLite and never writes
back to Markdown.

**Why:** The prior loose Markdown conversion was sloppy, but one-file-at-a-time human review is
valuable and Git needs a readable way to recreate the corpus. A strict importer avoids making
Markdown a second runtime store.

**Consequence:** Approved Markdown is validated bootstrap input; SQLite is runtime truth.
Runtime agents use database queries or exports. Goose owns schema and sqlc owns typed query
access.

**Revisit when:** Never for bidirectional synchronization. The bootstrap format may evolve
through versioned ingestion migrations.

## D021 — True up recipes one at a time

**Status:** Active

**Decision:** Present and approve one completed recipe per review by default. The household may
explicitly delegate a completion pass, but every recipe is still reviewed independently and
all non-obvious calls are written to a named re-review queue.

**Why:** Ingredient mapping, backfilled facts, packages, and instructions require focused human
judgment. Batch approval would hide mistakes.

**Consequence:** The true-up ledger tracks each recipe through inventory, drafting, review, and
verification. Delegation changes the interaction cadence, not the requirement for per-recipe
review evidence or visible conflict decisions.

**Revisit when:** Future onboarding may streamline low-risk fields but must preserve explicit
recipe verification.

## D022 — Store full in-app recipes

**Status:** Partly superseded by D036 (SQLite storage; now PostgreSQL)

**Decision:** Recipe details include sections, ingredients, hands-on and unattended time,
source attribution, and ordered read-only instructions from SQLite.

**Why:** The application should be useful while cooking, not merely redirect to potentially
changing websites.

**Consequence:** Instructions are required for verification; times may be `unknown`. Instructions
may be rewritten clearly during true-up.

**Revisit when:** Interactive cooking mode or source refresh is scoped.

## D023 — Keep list provenance compact but available

**Status:** Active

**Decision:** Group groceries by store section and expose contributing recipes behind an expand
action.

**Why:** Provenance is essential for trust, but displaying it constantly would reduce shopping
scanability.

**Consequence:** Contribution rows are first-class data. The default list remains compact.

**Revisit when:** Mobile use indicates another presentation is clearer.

## D024 — Weekly list edits do not alter recipes

**Status:** Active

**Decision:** Allow arbitrary additions, removal, completion, and quantity overrides on the
week's list only.

**Why:** The list needs to behave like a practical checklist without turning incidental weekly
choices into corpus mutations.

**Consequence:** Manual additions resolve known household items or enter `Other` with a section
picker (D044). Generated edits retain contribution provenance. Recipe correction UI is deferred.

**Revisit when:** A deliberate promotion flow from list correction to recipe correction is
scoped.

## D025 — No explicit week lock

**Status:** Active

**Decision:** Assume users settle recipes before shopping, but do not add a lock/unlock state.

**Why:** The natural workflow already minimizes recipe/list collisions. Enforcing it adds
ceremony for an edge case.

**Consequence:** Recipe changes recompute generated contributions with simple, predictable
best effort. Sophisticated reconciliation is deferred.

**Revisit when:** Real usage demonstrates repeated destructive conflicts.

## D026 — Current week is manually created

**Status:** Partly superseded by D038 (past weeks are now shown in History) and D051 (a plan
lasts until the household starts a new one)

**Decision:** The user explicitly generates a week; the application assigns the current
Sunday. It does not auto-roll over or expose future/past week interfaces.

**Why:** Automatic lifecycle behavior adds state and edge cases without improving the core v1
flow.

**Consequence:** Historical rows are retained for future use but not shown.

**Revisit when:** Weekly use makes history or automatic rollover valuable.

## D027 — Responsive Linear-inspired UI

**Status:** Active

**Decision:** Use Linear as the primary visual and interaction reference, following its visual
grammar closely where it fits Grocery Router. Both planning and shopping must work well on
iPhone.

**Why:** The current UI felt like data administration. The target should look serious, move
cleanly, and avoid explanatory clutter. Linear provides a concrete reference for compact
density, hierarchy, typography, dark surfaces, borders, interaction feedback, and restrained
motion.

**Consequence:** React and TypeScript are the preferred frontend; mobile is a first-class layout.
Borrow aggressively from Linear's presentation without copying its branding or importing
product-specific controls that do not serve Grocery Router's five-screen workflow (D037, D038).
The Atlas workspace exploration (`design/ui-explorations/concept-a.html`) is the production starting
composition. Concept C may inform one-handed grocery interactions, but Atlas controls the
visual language and information architecture.

**Revisit when:** User testing identifies a clearer visual direction.

## D028 — Go, SQLite, Goose, and sqlc

**Status:** Partly superseded by D036 and D042 (SQLite, deferred hosting; ADK rationale)

**Decision:** Build the backend in Go with local SQLite, Goose migrations, and sqlc queries.

**Why:** This is the desired durable backend direction and supports a future Google ADK for Go
integration without making AI part of v1.

**Consequence:** Existing Python code is not the implementation base. Docker, hosting, and
single-binary packaging are deferred.

**Revisit when:** Only if implementation discovery reveals a blocking technical constraint.

## D029 — No authentication or offline mode locally

**Status:** Partly superseded by AUTH_HOUSEHOLDS_SPEC.md and D036 (no auth, local-only)

**Decision:** Trust clients on the local network and require the host computer to remain
available.

**Why:** Authentication, offline synchronization, and deployment do not test the core product.

**Consequence:** Local iPhone access is sufficient. Public exposure is unsupported.

**Revisit when:** Hosting or offline shopping becomes an explicit phase.

## D030 — Keep an injectable, approved Markdown corpus

**Status:** Partly superseded by D036 (SQLite ingest target; now PostgreSQL)

**Decision:** Commit Goose migrations and one strictly structured approved Markdown file per
recipe, not the live SQLite database or a hand-maintained SQL dump.

**Why:** Recipe-by-recipe Markdown is readable during true-up and can remain the durable Git
bootstrap. Hand-writing the same recipe again as SQL would add a second transcription step and
another place for mistakes.

**Consequence:** A Go command validates all approved files and ingests them transactionally
into an empty migrated SQLite database. Automatic export, backup, and bidirectional
synchronization are deferred.

**Revisit when:** Formal backup/export or release automation is scoped.

## D031 — Keep deferred ideas visible but out of v1

**Status:** Active

**Decision:** Track all “not yets” in `UP_NEXT.md` rather than leaving placeholders in schema,
navigation, or implementation.

**Why:** The interview identified valuable future work, but designing it now would repeat the
scope expansion that made the prototype incoherent.

**Consequence:** Deferred work needs a new phase and observed justification before promotion.

**Revisit when:** An item meets the promotion rule in `UP_NEXT.md`.

## D032 — Suggestions do not replace recipe requirements

**Status:** Active

**Decision:** Keep a generic approved requirement when the source allows multiple ways to
satisfy it, and preserve a preferred option as a short suggestion note.

**Why:** Mapping `2 cups cooked chicken` to `Rotisserie Chicken` would force one purchase choice
before substitutions exist. The shopper should retain the source quantity and decide how to
satisfy it.

**Consequence:** The Chicken and Biscuits pilot displays `2 cups Cooked Chicken` with the note
`suggestion: rotisserie chicken`. A suggestion is not a substitution relationship and does not
change aggregation identity or quantity.

**Revisit when:** First-class substitutions and alternatives move into scope.

## D033 — Non-purchased inputs remain explicit

**Status:** Active

**Decision:** Retain non-purchased recipe inputs such as tap water in the structured recipe,
but mark them explicitly as non-shopping so grocery generation excludes them.

**Why:** Dropping water would make the recipe incomplete, while putting `2 cups Water` on every
grocery list would misrepresent an ordinary cooking input. This is a reviewed source mapping,
not pantry inference.

**Consequence:** Recipe ingredients store an `include_on_grocery_list` decision. Ingestion
requires that decision to be explicit in the approved Markdown; runtime week generation never
infers it.

**Revisit when:** Household inventory or configurable staple policies enter scope.

## D034 — Use a TanStack-first TypeScript frontend

**Status:** Active

**Decision:** Build the React frontend with strict TypeScript and Vite. Use TanStack Router for
typed navigation and TanStack Query for PostgreSQL-backed server state. Prefer a mature TanStack
library when a future in-scope need fits one directly, including charting if charts are ever
introduced.

**Why:** A coherent TanStack stack provides strong TypeScript inference and consistent async
state conventions without introducing a separate global client-state architecture.

**Consequence:** OpenAPI generates the frontend contract types, query keys define server-state
ownership, and transient interaction state remains local to components. Do not install TanStack
packages speculatively for deferred features, and do not add Redux-style duplication of server
state.

**Revisit when:** A concrete requirement is poorly served by the corresponding TanStack
library or the library is not production-ready.

## D035 — Use the three-stroke basket/list identity for v1

**Status:** Active

**Decision:** Use Concept I as the provisional v1 mark: a basket handle above two grocery-list
rows. Replace the prototype purple accent with Market teal (`#50BFA5`) and its documented dark,
hover, focus, and success companions.

**Why:** The three-stroke mark is the simplest explored combination of the product's planning
and shopping roles. It stays legible at favicon size and avoids generic cart, AI, food-blog,
and workflow-diagram imagery. Teal avoids both Anthropic's coral identity and the beehive
association produced when the three-stroke mark was paired with gold.

**Consequence:** v1 ships a primary mark, monochrome mark, horizontal lockup, optically adjusted
favicon, and Apple touch icon. Ordinary UI icons remain a separate lighter-weight system. Minor
optical corrections are allowed, but additional logo concepts or an expanded marketing-image
system are not required to complete v1.

**Revisit when:** Real-device testing reveals a legibility issue or a later public/hosted phase
requires a broader marketing identity.

## D036 — Replace SQLite with PostgreSQL managed by CloudNativePG

**Status:** Active

**Decision:** PostgreSQL is the sole runtime data store. Production and preview databases run
as CloudNativePG `Cluster` resources on the Talos Kubernetes cluster; local development uses a
containerized PostgreSQL instance. Goose and sqlc remain the migration and query tools.

**Why:** Deployment is no longer local-only, and PostgreSQL removes the application's coupling
to a pod-mounted database file while providing a supported Kubernetes database lifecycle.

**Consequence:** `GROCERY_ROUTER_DATABASE_URL` replaces the SQLite path, application pods are
stateless, and CNPG owns credentials and persistent volumes. Existing SQLite data is not migrated
because it contains no data worth preserving. A single database instance is appropriate while
the Talos cluster has one node; additional instances would not provide node-level availability.

**Revisit when:** The cluster gains multiple failure domains or requires object-store backups,
point-in-time recovery, or a tested disaster-recovery policy.

## D037 — Make the verified corpus directly browsable

**Status:** Partly superseded by D038 (four-screen count)

**Decision:** Add a permanent Recipes screen that lists verified recipes alphabetically, supports
name search, opens the existing recipe detail view, and allows adding a recipe to an existing
current week.

**Why:** Household use exposed that recipe details were reachable only through the current week.
Finding an approved recipe for ad hoc cooking required temporarily changing the plan, even though
the API already treated the verified corpus as readable application data.

**Consequence:** Grocery Router has four product screens. Search remains intentionally limited to
recipe names; role, ingredient, time, and taxonomy filters remain deferred until their metadata and
use cases justify them. The active query is preserved when returning from recipe detail.

**Revisit when:** Corpus size or observed lookup behavior makes name search insufficient.

## D038 — Expose retained weeks as read-only history

**Status:** Active

**Decision:** Add a dedicated History destination listing earlier Sunday-to-Saturday weeks.
Each historical detail shows the final meal pool and retained grocery checklist state, including
checked, unchecked, removed, manual, and adjusted lines. The current Week screen explicitly
shows `Planned` or `Not planned` and its date range.

**Why:** Real household use showed that an unlabelled current pool was ambiguous and that past
meal combinations are valuable household memory. The schema already retains one row per week
and the checklist state needed to answer both needs without introducing event history.

**Consequence:** History is read-only and displays one final state per week, not each regeneration
or edit. Copying a past pool into the current week and comparing weeks remain deferred.
This decision supersedes D002 and D027 only on the screen count, D037 only on its four-screen
count, and D026 only where it says past weeks are not exposed.

**Revisit when:** The household repeatedly wants to reuse an earlier pool or needs an audit trail
of changes within a week.

D039–D043 were approved with [`CATALOG_INGEST_SPEC.md`](CATALOG_INGEST_SPEC.md) on 2026-11-08.

## D039 — Shared catalog plus household-owned copies

**Status:** Active

**Decision:** Grocery Router is built for one known household but shaped so other households can
join later. Every new table is either shared catalog data with no household, or household-owned
data scoped like the existing tenant tables. A household adopts a catalog recipe by receiving its
own editable copy; catalog recipes are never edited by households.

**Why:** The family is the only consumer, but the authentication phase already made all product
data household-owned. Keeping catalog work (fetching, tagging, mapping) shared means its cost is
paid once per recipe regardless of household count, while copies let each household correct or
tune recipes without affecting others.

**Consequence:** Signup, invitations, billing, and administration stay deferred. New work must not
add product data that is neither catalog nor household-owned. Onboarding a future household reuses
the same adoption path as a single suggested recipe.

**Revisit when:** A second household is actually admitted or catalog sharing needs per-household
visibility rules.

## D040 — Release-qualified publication for catalog recipes

**Status:** Active

**Decision:** Catalog recipes publish automatically, but only under a pipeline release
that has passed qualification against a reviewed evaluation set, with sampled audits and stop rules
after publication. Catalog trust is recorded as release provenance, never as `verified`. The
household corpus keeps its human-approval rule.

**Why:** The pilot produced 17 web candidates and every one had blockers. Per-recipe review cannot
reach tens of thousands of recipes, but the blockers were not all trivial: some would have produced
wrong grocery lists. Trust therefore moves from each recipe to each measured pipeline release.

**Consequence:** Rejection is normal and is fixed through rules or overrides, not manual review of
each recipe. No requirement may be dropped or inferred to pass a gate. A failing site or release
stops publishing and can be withdrawn or rolled back. Users can swap a suggested recipe out of a
week or edit their own copy, so no confirmation step is added when a catalog recipe enters a week.

**Revisit when:** Qualification or audits repeatedly fail, or household use shows grocery errors
from catalog recipes.

## D041 — AI runs offline only

**Status:** Active

**Decision:** Models may be called only from operator-run pipeline commands. No user request, week
operation, or grocery operation calls a model. Recipe suggestions are computed from data prepared
offline, such as embeddings, without runtime AI.

**Why:** Runtime AI adds per-user cost with no payment structure to cover it, which is risky for a
multi-household shape. Offline AI is a one-time cost per recipe or ingredient name and benefits
every household.

**Consequence:** Plain-language requests and interactive onboarding chat remain deferred. A runtime
model may be added later only for a specific feature, with a measured benefit (for example, the
suggestion hide-and-recover evaluation) and a per-household spend cap.

**Revisit when:** The AI-free recommender measurably fails to produce good suggestions, or new-user
onboarding is scoped.

## D042 — OpenAI through the official Go SDK

**Status:** Active

**Decision:** Use OpenAI through the official `openai-go` SDK behind a narrow internal interface.
Do not adopt Google ADK for structured extraction and embeddings. This supersedes
the ADK direction in D003's revisit condition.

**Why:** The pipeline needs structured outputs, embeddings, and later batch jobs, not multi-step
agents. A direct SDK is simpler, and the Batch API lowers bulk cost when the pipeline scales.

**Consequence:** Provider or model changes stay local to one package. ADK may be reconsidered if a
real conversational agent is scoped.

**Revisit when:** An agent workflow with tools and memory is approved, or another provider is
clearly better for the pipeline's tasks.

## D043 — Suggested recipes enter the week marked, and join when kept

**Status:** Direction only (not implemented)

**Decision (direction for a later phase):** A household sets how many new recipes it wants per
week. Suggested catalog recipes appear in the week with a subtle "picked for you" marker and can be
swapped like any other meal. A suggestion that stays in the week joins the household collection; one
swapped out does not and counts as a mild negative signal.

**Why:** It keeps the collection from filling with recipes nobody cooked, uses existing week
editing instead of a new review step, and produces feedback for later suggestions.

**Consequence:** Ratings and the exact signal model are decided in the suggestion phase spec. A
suggestion needs a household snapshot before it can generate groceries, and a recipe kept in a week
is evidence of retention, not proof it was cooked or liked.

**Revisit when:** That spec is written.


## D044 — Shopping follows the household store walk

**Status:** Partly superseded by D045 and D046 (completed-row sinking, section folding, Undo on check, Hide done and its default)

**Decision:** Promote issues #36–#38 after a real 54-item shop exposed misplaced manual items,
excessive scrolling through completed rows, and unsuitable alphabetical section ordering.
Known manual items resolve deterministically against household names and normalized keys; unknown
items enter Other immediately with a section picker. Manual lines stay separate from recipe
requirements and preserve their identity and section through regeneration.

**Why:** Shopping should follow the store and keep attention on what remains to pick up.

**Consequence:** One persistent household section order applies to every week and history.
Defaults follow the issue's perimeter-first layout. All sections, including Other and Non-shopping,
can move; new sections insert before Other. Full-order saves validate membership and duplicates,
serialize with section creation, and roll back on failure. Reset restores defaults. Completed rows
sink within sections after a short idle delay, fully-done sections collapse, and Undo is available.
Hide done defaults off and persists per device. Failures retain input and show errors. This
supersedes the original in-place completion rule and D024's Other-only manual placement.

**Revisit when:** Repeated unknown terms justify household aliases or within-section item ordering
needs scope. Shared catalog aliases remain gated by the catalog phase.

## D045 — Remember where unknown items go; Hide done owns folding

**Status:** Partly superseded by D046 (Hide done control and section folding)

**Decision:** When the household places an unknown manual item in a section, remember that name's
section for the household and reuse it in suggestions and later additions. Hide done is the single
control for getting completed rows out of the way: it hides checked rows and folds fully completed
sections; Show done shows everything. Hide done defaults on.

**Why:** Real use after D044: cottage cheese placed in Dairy was unknown again next time, and fully
completed sections folded whether or not Hide done was on, so the toggle appeared to do nothing
and opening a folded section under Hide done showed an empty list.

**Consequence:** `remembered_item_sections` stores normalized name to section per household.
Placing a name in Other forgets it. Known corpus items are never overridden by memory. This
supersedes D044's "remembered terms require separate scope" and its Hide done default.

**Revisit when:** The shared catalog vocabulary (CATALOG_INGEST_SPEC) supplies aliases that should
replace or seed household memory.

## D046 — Plain rows, one Done group, undo only for removal

**Status:** Active

**Decision:** The grocery row is the check target. Every other line action (quantity, section,
recipes, remove) lives behind one quiet details control. Checked lines settle into a single collapsed
Done group at the bottom instead of folding sections behind a Hide done toggle. Undo moves from
checking to removal, and dismisses itself. Checks update optimistically and never wait on a save.
Reorder sections becomes a quiet header icon.

**Why:** D044 and D045 solved real problems by adding visible controls: three always-visible icons
per row on iPhone, a toggle whose label described the action rather than the state, per-section
folding with a peek button, and a check toast that never dismissed and only covered the last item,
while removal had no undo. Checking is trivially reversible; removal is not. A 26px checkbox was the
only check target, and checks were blocked while any save was in flight.

**Consequence:** This supersedes D044's fold and Undo-on-check behavior and D045's Hide done
control. The grocery line update accepts `removed: false` to restore a removed line; removal stays
DELETE. Done group open state is not persisted.

**Revisit when:** Real shopping shows the Done group needs to stay open, or swipe gestures are wanted
for remove.

## D047 — One checklist row per grocery item; look-alike items need a decision

**Status:** Active

**Decision:** The checklist shows each grocery item once. Exact addition still happens first;
amounts that cannot be added exactly stay as separate quantities on the same row
(`8 cloves + 1 tbsp`), and a requirement with no amount adds `+ more` instead of its own row.
Separately, grocery items whose names look alike must be classified (alias, group, or distinct)
before the corpus audits clean, and the catalog freeze applies the same gate automatically.

**Why:** Real use showed Black Pepper twice (2 tsp from two recipes, and a "generous" amount from
a third). Unspecified requirements were keyed per recipe and incompatible units per unit, so 18
items could repeat. Separately named near-duplicates (Kosher Salt and Salt, Onion and Yellow Onion)
are a vocabulary problem that aggregation cannot fix and that new recipes would keep introducing.

**Consequence:** Migration 00007 stores a row's second and later quantities; the current week
rebuilds once on read and keeps checked, removed, and single-row adjusted state; history is not
rebuilt. The catalog evaluation format is unchanged because grouping happens after aggregation.
`corpus/item-reviews.yaml` classifies 37 pairs: 2 aliases, 19 substitution-group pairs, and 16
distinct. Turning aliases into one item is a separate corpus change; groups take effect only
under D048.

**Revisit when:** The catalog vocabulary supplies aliases for the bootstrap corpus, or a row's
combined amount needs a purchase-oriented summary.

## D048 — Substitution is a household choice over frozen groups (proposed)

**Status:** Direction only (not implemented)

**Decision (proposed, not authorized):** Split "same product" from "would we buy one for both".
The pipeline settles identity strictly (aliases from rules or overrides). It also assigns
different-but-swappable products to flat substitution groups, which AI may propose. Each household
chooses per group, when two members first meet in one week, whether to combine them on its list.
See [`SUBSTITUTIONS_SPEC.md`](SUBSTITUTIONS_SPEC.md).

**Why:** Households differ in how freely they substitute (yellow for white onion, any shredded
cheese), so combining cannot be a shared rule, and the owner cannot review pipeline output item by
item. Recipes and the list must stay deterministic.

**Consequence:** Recipes never change; groups are frozen with the vocabulary; the list is a pure
function of recipes, snapshot, and recorded answers; no AI runs while building a list. A wrong
group costs one question, so AI may assign groups but not aliases.

**Revisit when:** The owner approves or revises the proposal.

## D049 — Household preferences live per household and are edited in the app

**Status:** Active

**Decision:** Preferences that change how a household plans (its time zone and how many recipes
a new week offers) are stored on the household row and edited by any member on the Settings
screen. Deployment configuration (environment variables, secrets, operator commands) is for the
operator only and never carries household preferences.

**Why:** The time zone could only be changed by an operator command and the recipe count was a
constant in the Week screen. Both are household choices; asking an operator to change them, or
redeploying to do so, puts household decisions in the wrong hands.

**Consequence:** Migration 00010 adds `households.default_recipe_count` (1-21, default 5) beside
the time zone. `GET`/`PATCH /households/{householdID}/settings` read and change either or both
values; an invalid value is rejected with 422 and nothing is stored. The Week screen starts from
the household's count, still capped by the recipes available, and saving reloads the current
week, its groceries, and history because a new time zone can move the week. The
`household-timezone` command remains for operators. Preferences are per household, not per
member.

**Revisit when:** A preference needs to differ between members, or a household-level setting
needs to be restricted to some members.

## D050 — The household chooses its rollover day, time, and mode

**Status:** Superseded by D051. Kept for history; migration 00013 removes its settings.

**Decision:** Each household chooses, on the Settings screen, the local day its week starts
(default Sunday), the local time it rolls over (default 12:00 AM), and whether it rolls over
automatically (the open week closes itself at that day and time) or manually (the week stays
open until someone starts the next one). Rolling over only closes the week; the next week still
exists only once someone presses Generate, and the Week and Groceries screens then lead into
that step with a link to the week that just ended.

**Why:** A household that shops on a different day, or late on Friday, has a different week than
the fixed Sunday-to-Saturday one, and a household that plans irregularly does not want its week
to close by the calendar at all.

**Consequence:** Migration 00012 adds `households.week_start_day` (0-6, 0 is Sunday),
`rollover_minutes` (0-1439), and `rollover_mode` (`automatic` or `manual`), drops the Sunday
check on `weeks.starts_on`, and replaces the unique `(household_id, starts_on)` constraint with a
plain index, because a week closed early and its successor can share a start date. An automatic
week starts on the local date of the most recent rollover at or before it was opened and closes
at the next one; a manual week starts on today's local date and has no close time. Saving a
change to the time zone, day, time, or mode moves the open week's close time, in the same
transaction, to the next rollover after now under the new settings (none in manual mode); saving
never closes a week, so a week is not cut short or ended by a settings change. The
`household-timezone` command moves the open week the same way. Weeks and History report
`endsOn`, the local date one second before the week closes or closed, so a Sunday 12:00 AM
rollover ends on Saturday and a Friday 6:00 PM rollover ends on Friday. A closed week stores that
date (`weeks.ends_on`) in the zone the household had when it closed, so a later time-zone change
never re-cuts History; any code that closes a week must set it. This partly supersedes
D026: the household, not the application, decides when its week rolls over.

Manual rollover exists in both modes: any household member can press "Start next week" on the
Week screen (`POST .../week/current/close`), which closes the open week now and moves it to
History. Closing never generates the next week; the Week screen goes straight to the "Plan next
week" step, and the next week exists only once someone presses Generate. A week closed by hand
keeps its `auto_close_at`. If the next week is generated in automatic mode while that rollover is
still ahead, the new week takes over the upcoming period instead of the one just ended: it starts
on the local date of that rollover and closes at the one after it, so a household that shops on
Saturday for a week starting Sunday is not rolled over again on Sunday. This holds for a close in
the middle of the week too: the next week still starts at the upcoming rollover, and the days
between the close and that rollover belong to no week. Once that rollover has passed, the next
week opens by the ordinary rule. If the settings changed after the close, the new week begins at
whichever of two rollovers under the new settings is nearer the skipped rollover: the last one at
or before now, or the next one after now; it closes at the rollover after its beginning. A
settings change saved while such a week has not yet reached the rollover it skips moves its close
to whichever of the next two rollovers under the new settings is nearer its old close, so it
neither ends at the rollover it was opened to skip nor runs a week longer. Because that close is
recomputed from now, a settings change after a mid-week close can shorten a week started early,
but its date range never goes backwards. A week that closes,
by hand or by its rollover, before its own start date (one started early and closed again before
its period began) ends on its start date.

**Revisit when:** A household needs more than one rollover a week, wants the next week
generated automatically at the rollover or when the week is closed by hand, or wants to plan a
future week while the current one stays open.

## D051 — A plan lasts from one shopping trip to the next and can be reopened

**Status:** Active

**Decision:** The current week becomes a *plan*: the meals and groceries between two shopping
trips, of no fixed length. Generate starts a plan on today's household-local date when none is
current. A plan stays current until a household member presses "Start new plan" (which closes it
and moves it to History) or reopens a past plan from History. Reopening makes that plan current
again with its meals and checklist as they were, and closes the plan that was current, if any.
There is still one current plan at a time, and nothing closes a plan by the calendar. The app
says "Plan" on screen; the code, API paths, and tables keep the name "week".

**Why:** D050 still assumed a seven-day cadence, which does not fit shopping for ten days, or for
a few, and it had no way back into a plan that was closed too early or is wanted again. Letting
the trip, not the calendar, end a plan covers every cadence with fewer settings and less code
than configuring when a week rolls over.

**Consequence:** Migration 00013 drops `households.week_start_day`, `rollover_minutes`,
`rollover_mode`, and `weeks.auto_close_at`; an open week stays open. Settings are again only the
time zone, which dates plans, and the default recipe count, which stays a fixed default and does
not scale with a plan's length. `POST /households/{householdID}/weeks/history/{weekID}/reopen`
reopens a plan; `endsOn` is null while a plan is current and is the local date it closed once it
is in History. Opening, closing, and reopening a plan lock the household row, so they run one at
a time per household. This supersedes D050 and the parts of D026 about when a week begins.

**Revisit when:** A household wants to plan the next trip while the current plan is still in
use, or to start a new plan from a past plan's meals without reopening it.

## D052 — A household can avoid recipes from its last few plans

**Status:** Active

**Decision:** Settings has "Avoid repeats": Off, or the last 1-8 plans (default Off). Generate
puts recipes from that many of the most recently closed plans last, so it picks them only when
too few other verified recipes remain; it never fails because of the setting. Random swap
prefers a recipe in neither the current pool nor those plans, then one not in the pool, then any
other recipe. Manual add and choosing a specific replacement are unaffected. Selection is still
plain uniform randomness, with no model and no scoring.

**Why:** With 24 recipes and about five per plan, recent dinners came back often, and History
already records them (#71).

**Consequence:** Migration 00014 adds `households.avoid_recent_plans` (0-8). "Recent" means plans
in History ordered by when they closed, so a reopened plan counts from its latest close. This
partly supersedes the archived V1 §6.2's "selection ignores previous weeks".

**Revisit when:** The household wants weighting, favorites, or breadth rules rather than a plain
exclusion.

## D053 — A household can place a known item in its own store section

**Status:** Active

**Decision:** Any line of a known grocery item, generated or manual, can be moved to another
section from its details sheet. That saves where the household's store keeps the item: all of
the item's lines on the current list move, and later lists, manual adds, and add-item
suggestions use the placement. Choosing the item's reviewed section again clears it. Only the
section can be changed this way. An item's shopping mode and whether it appears on the list stay
corpus decisions (#70: the owner approved only the section override).

**Why:** Stores differ. If the household's store keeps tortillas in Bakery and the corpus says
Pantry, the list was wrong every week (#70). This extends D045 from unknown names to known items.

**Consequence:** Migration 00015 adds `household_item_sections (household_id, grocery_item_id,
store_section_id)` with tenant-proving composite foreign keys. `ListWeekIngredientRequirements`
and `ListGroceryItems` use the placement in place of the reviewed section. History keeps the
sections its lists had. This partly supersedes D045's "known items keep their reviewed section".

**Revisit when:** A household wants to change shopping mode or leave an item off the list.

## D054 — Amounts in different units are shown side by side, not converted

**Status:** Active

**Decision:** On the grocery list, amounts in the same unit still add exactly (1 tsp + 2 tsp shows
3 tsp). Amounts in different units are no longer converted into one unit. They are listed on the
item's one row, largest unit of a dimension first: 2 cups and 1 tsp show as "2 cups + 1 tsp",
not "97 tsp". A row that more than one recipe needs is marked with "N recipes" and its amount is
highlighted, so it is checked at the store.

**Why:** Converting kept the first recipe's unit, so the display depended on recipe order and
gave totals like 97 tsp or 40 oz (#51). Showing each needed amount is what a shopper can act on.
The owner chose this over picking a tidy unit.

**Consequence:** `grocery.Aggregate` groups mass and volume by unit instead of by dimension, and
`grocery.Group` orders a row's amounts. Aggregation version 3 rebuilds open lists when they are
next read; History keeps the amounts it had. The unit table's conversion ratios stay but are no
longer used to combine amounts. This partly supersedes the archived V1 §2.5 and §7.3.

**Revisit when:** The household wants mixed amounts simplified, or metric and US display.

