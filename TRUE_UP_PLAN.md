# Grocery Router v1 — Corpus True-Up Plan

Status: **required v1 delivery plan**  
Product and data contract: [`V1_SPEC.md`](V1_SPEC.md)

This document governs the one-time migration from the raw family recipe evidence into the
verified SQLite corpus. The PDF defines membership; `V1_SPEC.md` defines what verified data
means. This plan defines how the work is executed, reviewed, tracked, and closed.

## Progress

- **Gate 1 complete:** `trueup/recipes.csv` inventories all 25 PDF recipes and passes the
  repository evidence audit.
- **Gate 7 in progress:** the Goose schema models sources, exact quantities, packages,
  canonical grocery items, instructions, review flags, and verification guards. The strict
  Markdown parser and transactional importer are working.
- **Fifteen recipes verified:** Chicken and Biscuits Casserole, Hamburgers, Sausage and
  Peppers, 3-Ingredient Teriyaki Chicken, Chicken Veggie Stir Fry, Crock Pot Italian Beef
  Sandwiches, Meatloaf, Beef Stew with Carrots and Potatoes, Easy Salmon Dinner, Chili,
  Enchiladas, Chicken Noodle Soup, Tacos, Pork Loin and Rice, and Cheesy Pasta are approved
  under `corpus/recipes/` and import successfully into a fresh migrated database.
- **Readable bootstrap enforced:** `corpus-render` generates recipe details, ingredients,
  instructions, grocery preview, and approved decisions from strict front matter;
  `corpus-audit` rejects any drift.
- **Next:** true up linked recipes directly from authoritative website recipe data, asking the
  household only about material ambiguities or backfilled choices.

## 1. Execution gates

True-up is a tracked migration, not an informal rewrite of the legacy Markdown recipes. Each
approved result is retained in a new strict Markdown bootstrap format and ingested into SQLite
through validated Go code. It proceeds in the following gates.

### Gate 1 — Inventory the PDF

Create a machine-readable ledger of every recipe represented by `sources/Recipes.pdf`. The
ledger is the completeness boundary and must include:

- stable inventory key;
- PDF name and page/location;
- linked URL, if present;
- available raw supporting evidence;
- proposed database recipe name;
- source relationship (`source` or `adapted-from`);
- workflow status;
- verification date; and
- a concise disposition note.

Allowed workflow statuses are:

- `inventoried`;
- `source-found`;
- `drafted`;
- `reviewable`;
- `changes-requested`;
- `verified`; and
- `excluded`.

`excluded` is exceptional and requires an explicit user decision. A blank or hard-to-read
PDF page is not an exclusion; it is a source-recovery task.

The inventory must be produced from the PDF itself. Existing corpus tables and recipe
filenames may help identify entries but may not define membership.

### Gate 2 — Establish source evidence

For each inventoried recipe:

1. Open and inspect the PDF entry.
2. If it has a link, retrieve and inspect the linked recipe. Prefer machine-readable
   schema.org Recipe data, and verify it against the visible recipe card before drafting.
3. Record the URL and the date on which it was checked.
4. Compare available typed inputs, transcripts, and legacy recipe data for useful
   household-specific evidence.
5. If the link is unavailable or the recipe has no link, search for a credible reference
   recipe and label the result `adapted-from`.
6. Transcribe authoritative linked facts without asking the household to restate them.
7. Bring only material ambiguity to the user: source conflicts, missing quantities,
   alternatives that v1 must resolve, uncertain purchased form, or genuinely backfilled
   household choices.

Raw website HTML does not need to become a permanent second recipe store. The approved
source URL, relationship, source ingredient lines, and review record are the durable
trace.

### Gate 3 — Draft the complete recipe

Build a database draft through the same ingestion/domain services intended for future
onboarding. For each recipe, draft:

- name and source relationship;
- source URL and optional image;
- source yield when available;
- hands-on and unattended time, allowing `unknown`;
- ordered ingredient sections;
- exact source ingredient text;
- structured quantity and package details;
- preparation and optional state;
- canonical grocery-item mapping;
- store section and shopping mode; and
- ordered, readable cooking instructions.

Agent-generated additions must be identifiable in the review packet. After approval, the
strictly structured Markdown review becomes the committed bootstrap record and SQLite stores
the current runtime truth. The draft must not become selectable.

### Gate 4 — Normalize grocery identity

Before proposing a new grocery item, ingestion must search the existing canonical item set.
The reviewer must check both failure directions:

- **false split** — two source phrases represent the same purchased item and should map
  together; and
- **false merge** — related words represent different purchased forms and must remain
  separate.

Preparation normally stays on the recipe ingredient. Purchased form stays in grocery-item
identity. Every newly introduced grocery item receives an approved display name, store
section, and shopping mode.

After any canonical-item change, corpus-wide validation and affected aggregation fixtures
must run; changing a shared item can alter previously approved lists.

### Gate 5 — Mechanical validation

A validation command moves a draft to `reviewable` only if all §4.5 and §11.13 invariants
hold. Its output must be readable and identify:

- missing required fields;
- unmapped ingredients;
- invalid quantities or packages;
- unsupported units;
- duplicate ordering;
- missing instructions;
- source phrases that map to the same grocery item within the recipe; and
- every agent-backfilled or materially rewritten field awaiting human approval.

Validation cannot approve a recipe.

### Gate 6 — One-recipe review

Recipes are reviewed **one at a time**. Each review packet must show:

1. PDF identity and source/reference link;
2. whether the recipe is sourced or adapted;
3. recipe name, yield, and times;
4. every ingredient section in order;
5. source text beside structured grocery interpretation;
6. package sizes, optional flags, preparation, store section, and shopping mode;
7. ordered instructions;
8. all backfilled, inferred, rewritten, or conflict-resolved fields; and
9. a preview of the grocery list produced by one baseline batch.

The user may approve, request changes, or explicitly exclude the recipe. Approval is
recorded in the ledger and promotes the recipe to `verified` transactionally. Requested
changes return it to draft/reviewable state.

### Gate 7 — Pilot before bulk migration

The schema and ingestion path must first be exercised against a deliberately difficult
pilot set. It should include, at minimum:

- a linked recipe with structured sections and package quantities;
- a PDF-only list such as Hamburgers that requires approved backfilling;
- a recipe involving a purchased-form question such as raw, cooked, or rotisserie chicken;
- a recipe with optional ingredients;
- a recipe with presence-only groceries;
- a recipe with source alternatives that require one v1 default; and
- a recipe whose ingredients exercise exact cross-unit aggregation.

Schema changes are expected during the pilot. Bulk one-at-a-time review begins only after
the pilot demonstrates that source text, structured quantities, packages, sections,
instructions, mappings, and list output can all be represented without recipe-specific
hacks.

### Gate 8 — Corpus-wide audit

After the last recipe review, run a complete audit that proves:

- every PDF inventory row has a disposition;
- every included recipe is verified;
- no draft/reviewable recipe is selectable;
- no verified ingredient is unmapped;
- all grocery items have valid store sections and shopping modes;
- all verified recipes have instructions;
- all units participate only in exact allowed conversions;
- one-baseline-batch list output exists for every recipe;
- no ingredient contribution is silently lost; and
- all schema, ingestion, and aggregation tests pass.

The audit should also report counts by status, source relationship, unknown time/yield,
shopping mode, store section, unit, and unresolved warning. Counts are diagnostics, not
quality scores.

### Gate 9 — Snapshot and closeout

Once the corpus audit passes:

1. ensure every included ledger row points to one committed approved Markdown recipe;
2. create a fresh local application database through Goose migrations;
3. ingest all approved recipe files in one transaction;
4. verify the fresh database against the complete-corpus audit;
5. record the import date and audit result; and
6. remove temporary migration artifacts that are neither approved bootstrap recipes, raw
   evidence, nor reusable onboarding fixtures.

The approved Markdown corpus, inventory ledger, ingestion rules, difficult fixtures, and
validation queries remain as future-onboarding and future-agent context. The running
application reads only SQLite.

## 2. Deliverables

True-up is complete only when the repository contains:

- the PDF inventory/disposition ledger;
- reusable ingestion and verification commands;
- the approved SQLite corpus;
- one strictly structured, individually approved Markdown bootstrap file per recipe;
- canonical grocery items, sections, modes, and exact units;
- representative ingestion and aggregation fixtures;
- a complete-corpus audit command; and
- concise future-agent/onboarding guidance derived from actual migration decisions.

## 3. Completion authority

True-up is complete only when the corpus acceptance requirements in `V1_SPEC.md` §15.6 and
the v1 definition of done both pass. Progress counts or successful imports do not substitute
for one-at-a-time approval and the final corpus-wide audit.
