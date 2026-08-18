# Grocery Router — Product Decision Record

Status: **durable rationale for the v1 direction**  
Behavioral authority: [`V1_SPEC.md`](V1_SPEC.md)  
Historical source: [`archive/interviews/v1-scope-interview.md`](archive/interviews/v1-scope-interview.md)

This document records why v1 has its current shape. It is not a second specification: when
wording conflicts, `V1_SPEC.md` governs behavior. The interview archive is non-authoritative
and exists only to recover nuance.

## Authority order

1. `V1_SPEC.md` — product and implementation contract.
2. `TRUE_UP_PLAN.md` — corpus migration execution contract.
3. `UP_NEXT.md` — explicitly deferred scope.
4. This record — rationale and revisit conditions.
5. Raw interview archive — historical evidence only.

---

## D001 — Reset instead of repairing the current application

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

**Decision:** v1 has one path: select known recipes, form an unordered week, and generate a
traceable grocery checklist.

**Why:** Reliable recipe data and quantities are the prerequisite for every more ambitious
feature. Discovery, learning, inventory, and agent reasoning obscure whether the core works.

**Consequence:** There are only three product screens: Week, Groceries, and Recipe detail.

**Revisit when:** The complete workflow is trusted in real household use.

## D003 — Random planning now; agent planning later

**Decision:** Generate the initial week uniformly at random from verified recipes.

**Why:** Random selection delivers recall without pretending the application has trustworthy
history, preferences, or model judgment. It is free and easy to regenerate.

**Consequence:** Initial results are unique and history-blind. Users can regenerate or edit
them manually. Runtime AI planning is excluded.

**Revisit when:** An OpenAI-backed planner through Google ADK for Go receives a separately
scoped contract and evaluation plan.

## D004 — The week is an unordered pool

**Decision:** Do not assign recipes to days.

**Why:** The household's weeks are dynamic; day labels would create false precision and
additional maintenance.

**Consequence:** A week contains recipe occurrences and a Sunday identity, not a day grid.

**Revisit when:** Actual use demonstrates value in day assignment.

## D005 — Direct week manipulation

**Decision:** Support add, remove, random swap, specific swap, and full regeneration.

**Why:** Generation is only a starting point. The user must be able to express a direct intent
without understanding planner mechanics.

**Consequence:** Manual duplication is allowed even though generated suggestions are unique.
Each occurrence contributes another complete batch.

**Revisit when:** These controls prove insufficient in use.

## D006 — No guest or serving scaling in v1

**Decision:** Use one approved source-baseline batch per recipe occurrence.

**Why:** Asking for guests implies the system can scale quantities correctly. The existing
corpus and yields do not yet justify that confidence.

**Consequence:** Number of dinners means number of recipe occurrences. Source yield is stored
for later but does not change list arithmetic.

**Revisit when:** Recipe yields and household serving behavior have been verified and scaling
receives dedicated fixtures.

## D007 — Include every ingredient

**Decision:** Generate groceries from every approved recipe ingredient, including salt,
spices, oil, and optional ingredients.

**Why:** The prior attempt to assume what the household already owned was inaccurate. A short
list is worse than an explicit complete list.

**Consequence:** There is no pantry, staple, or probably-have classification. Optional items
remain visibly optional.

**Revisit when:** Inventory is based on observed state rather than assumptions.

## D008 — Preserve recipe requirements; do not estimate purchases

**Decision:** v1 reports approved recipe requirements and explicit package details. It does
not round, pad, or translate them into advice about how much to buy.

**Why:** Poor purchase estimates were a direct failure in the prototype. Accuracy begins with
correct recipes, not clever formatting.

**Consequence:** `1.3 lb` remains `1.3 lb` if that is the approved requirement. Raw-to-cooked
yields and package optimization are deferred.

**Revisit when:** Purchase conversion rules can be evidenced and tested without silent
underbuying.

## D009 — Exact conversions only

**Decision:** Aggregate through universally exact unit conversions; keep incompatible or
item-specific quantities separate.

**Why:** Converting tablespoons to teaspoons is arithmetic. Converting an onion to cups is an
assumption about size and preparation.

**Consequence:** The list may show separate requirements for one grocery item when no exact
conversion exists.

**Revisit when:** A reviewed item-specific conversion model is deliberately introduced.

## D010 — Shopping modes are reviewed item data

**Decision:** Each canonical grocery item is `measured`, `counted`, or `presence-only`.

**Why:** Recipe measurements are not always useful shopping quantities. Salt should appear as
`Salt`, not a summed teaspoon quantity, without implying that it is already owned.

**Consequence:** Shopping mode is assigned during true-up, never inferred during list creation.

**Revisit when:** Store products or inventory provide a stronger purchase representation.

## D011 — Canonical grocery mapping happens during ingestion

**Decision:** Every recipe ingredient maps to a canonical shoppable item before the recipe is
verified.

**Why:** Selection-time LLM mapping would make identical weeks nondeterministic and allow
silent ingredient merges. Mapping is corpus work, not planning work.

**Consequence:** Week selection and list generation require no model. A verified recipe cannot
contain an unmapped ingredient.

**Revisit when:** Do not move mapping to runtime. Future onboarding may automate proposals but
must retain verification.

## D012 — Purchased form matters

**Decision:** Related foods remain separate grocery items when they are bought differently.

**Why:** Raw chicken breast, cooked chicken, and rotisserie chicken are not interchangeable
shopping instructions even when a recipe could eventually support substitutions.

**Consequence:** Purchased form participates in canonical grocery identity. Substitution
relationships are deferred.

**Revisit when:** A first-class substitution model is scoped.

## D013 — Preserve package sizes

**Decision:** Keep package count and package size in grocery requirements, and do not merge
different requested sizes.

**Why:** `2 cans` is insufficient shopping guidance when the source specifies `2 × 14.5 oz
cans`.

**Consequence:** Different package-size requirements remain separate even when their total
weights could be added mathematically.

**Revisit when:** Package optimization can choose actual products.

## D014 — Optional ingredients stay visible

**Decision:** Include optional ingredients by default and mark them visibly optional.

**Why:** Omitting them silently creates an incomplete list; treating them as mandatory loses
source meaning.

**Consequence:** Optional state survives aggregation and remains traceable to recipes.

**Revisit when:** The user requests per-week optional-item inclusion controls.

## D015 — The PDF controls membership; websites control linked content

**Decision:** `sources/Recipes.pdf` determines which recipes belong. For linked entries, the
website inspected during true-up controls ingredients, quantities, and instructions.

**Why:** The PDF is the family's canonical recipe inventory, while the linked page is the
stronger recipe source.

**Consequence:** Existing derived corpus counts and Markdown files cannot add or remove
membership. Source URL and check date are recorded.

**Revisit when:** The household explicitly changes corpus membership after v1 onboarding
exists.

## D016 — Source completeness and recipe completeness are different

**Decision:** A source may be sparse; a selectable database recipe must be complete.

**Why:** Several genuine family recipes are only grocery lists in the PDF. Rejecting them would
lose real corpus membership, while selecting them unchanged would produce unusable recipe
pages and uncertain groceries.

**Consequence:** Missing structure, ingredients, quantities, and instructions may be
backfilled, reviewed, and approved.

**Revisit when:** Never as an invariant. Future onboarding should preserve the same distinction.

## D017 — Agent assistance is allowed only before approval

**Decision:** The coding agent may propose missing recipe facts during true-up, including
entire ingredients and quantities. The user approves each recipe before verification.

**Why:** Backfilling is necessary, but model output must not pass as household truth or enter a
shopping list without review.

**Consequence:** Recipes move through draft, reviewable, and verified states. Only verified
recipes are selectable.

**Revisit when:** Future onboarding agents may change the interface, not the approval boundary.

## D018 — Distinguish sourced from adapted recipes

**Decision:** Record whether an approved recipe is a direct source or adapted from a reference.

**Why:** A similar online recipe can fill gaps for a PDF-only family recipe, but the final
household version should not be represented as an exact copy.

**Consequence:** Attribution remains honest while permitting useful completion.

**Revisit when:** A richer source/revision model is introduced.

## D019 — Preserve source text and structured interpretation

**Decision:** Store every original ingredient line beside exact structured quantity, package,
preparation, optional state, and canonical grocery mapping.

**Why:** Structured fields enable arithmetic; source text makes that interpretation auditable.
One without the other is either unusable or untrustworthy.

**Consequence:** Ingestion and review show both representations. SQLite stores one approved
current truth rather than a complete correction history.

**Revisit when:** Revision history is scoped.

## D020 — SQLite is the sole runtime recipe truth

**Decision:** Do not keep recipe Markdown synchronized with SQLite.

**Why:** Two active representations drift. The prior Markdown conversion was itself considered
sloppy and must not remain an equal authority.

**Consequence:** Runtime and future agents use database queries or exports. Goose owns schema;
sqlc owns typed query access.

**Revisit when:** Never for synchronized Markdown. Add controlled exports if needed.

## D021 — True up recipes one at a time

**Decision:** Present and approve one completed recipe per review.

**Why:** Ingredient mapping, backfilled facts, packages, and instructions require focused human
judgment. Batch approval would hide mistakes.

**Consequence:** The true-up ledger tracks each recipe through inventory, drafting, review, and
verification.

**Revisit when:** Future onboarding may streamline low-risk fields but must preserve explicit
recipe verification.

## D022 — Store full in-app recipes

**Decision:** Recipe details include sections, ingredients, hands-on and unattended time,
source attribution, and ordered read-only instructions from SQLite.

**Why:** The application should be useful while cooking, not merely redirect to potentially
changing websites.

**Consequence:** Instructions are required for verification; times may be `unknown`. Instructions
may be rewritten clearly during true-up.

**Revisit when:** Interactive cooking mode or source refresh is scoped.

## D023 — Keep list provenance compact but available

**Decision:** Group groceries by store section and expose contributing recipes behind an expand
action.

**Why:** Provenance is essential for trust, but displaying it constantly would reduce shopping
scanability.

**Consequence:** Contribution rows are first-class data. The default list remains compact.

**Revisit when:** Mobile use indicates another presentation is clearer.

## D024 — Weekly list edits do not alter recipes

**Decision:** Allow arbitrary additions, removal, completion, and quantity overrides on the
week's list only.

**Why:** The list needs to behave like a practical checklist without turning incidental weekly
choices into corpus mutations.

**Consequence:** Manual additions use `Other`; generated edits retain underlying contribution
provenance. Recipe correction UI is deferred.

**Revisit when:** A deliberate promotion flow from list correction to recipe correction is
scoped.

## D025 — No explicit week lock

**Decision:** Assume users settle recipes before shopping, but do not add a lock/unlock state.

**Why:** The natural workflow already minimizes recipe/list collisions. Enforcing it adds
ceremony for an edge case.

**Consequence:** Recipe changes recompute generated contributions with simple, predictable
best effort. Sophisticated reconciliation is deferred.

**Revisit when:** Real usage demonstrates repeated destructive conflicts.

## D026 — Current week is manually created

**Decision:** The user explicitly generates a week; the application assigns the current
Sunday. It does not auto-roll over or expose future/past week interfaces.

**Why:** Automatic lifecycle behavior adds state and edge cases without improving the core v1
flow.

**Consequence:** Historical rows are retained for future use but not shown.

**Revisit when:** Weekly use makes history or automatic rollover valuable.

## D027 — Responsive Linear-inspired UI

**Decision:** Use a polished, compact, dark, professional visual direction inspired by Linear.
Both planning and shopping must work well on iPhone.

**Why:** The current UI felt like data administration. The target should look serious, move
cleanly, and avoid explanatory clutter.

**Consequence:** React and TypeScript are the preferred frontend; transitions are restrained,
and mobile is a first-class layout.

**Revisit when:** User testing identifies a clearer visual direction.

## D028 — Go, SQLite, Goose, and sqlc

**Decision:** Build the backend in Go with local SQLite, Goose migrations, and sqlc queries.

**Why:** This is the desired durable backend direction and supports a future Google ADK for Go
integration without making AI part of v1.

**Consequence:** Existing Python code is not the implementation base. Docker, hosting, and
single-binary packaging are deferred.

**Revisit when:** Only if implementation discovery reveals a blocking technical constraint.

## D029 — No authentication or offline mode locally

**Decision:** Trust clients on the local network and require the host computer to remain
available.

**Why:** Authentication, offline synchronization, and deployment do not test the core product.

**Consequence:** Local iPhone access is sufficient. Public exposure is unsupported.

**Revisit when:** Hosting or offline shopping becomes an explicit phase.

## D030 — Keep a manual injectable corpus snapshot

**Decision:** Commit migrations and a manually refreshed SQL corpus snapshot, not the live
SQLite database.

**Why:** The runtime database should remain local while the verified starting corpus remains
recoverable and reviewable enough for this phase.

**Consequence:** Snapshot refresh is an intentional true-up closeout action. Automatic export,
backup, and synchronization are deferred.

**Revisit when:** Formal backup/export or release automation is scoped.

## D031 — Keep deferred ideas visible but out of v1

**Decision:** Track all “not yets” in `UP_NEXT.md` rather than leaving placeholders in schema,
navigation, or implementation.

**Why:** The interview identified valuable future work, but designing it now would repeat the
scope expansion that made the prototype incoherent.

**Consequence:** Deferred work needs a new phase and observed justification before promotion.

**Revisit when:** An item meets the promotion rule in `UP_NEXT.md`.

## D032 — Suggestions do not replace recipe requirements

**Decision:** Keep a generic approved requirement when the source allows multiple ways to
satisfy it, and preserve a preferred option as a short suggestion note.

**Why:** Mapping `2 cups cooked chicken` to `Rotisserie Chicken` would force one purchase choice
before substitutions exist. The shopper should retain the source quantity and decide how to
satisfy it.

**Consequence:** The Chicken and Biscuits pilot displays `2 cups Cooked Chicken` with the note
`suggestion: rotisserie chicken`. A suggestion is not a substitution relationship and does not
change aggregation identity or quantity.

**Revisit when:** First-class substitutions and alternatives move into scope.
