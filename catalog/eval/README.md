# Catalog evaluation set

The evaluation set for the shared recipe catalog (CATALOG_INGEST_SPEC.md §4 slice 1, §6.3).
`catalog-qualify` scores a candidate set against it. `internal/catalog/qualify` is the strict
parser and harness; every rule below is enforced there.

## Parts

| Directory | Holds | Qualification rule (§6.3) |
| --- | --- | --- |
| `must-pass/` | supported cases that must publish correctly (~10) | publishes with zero material errors, unless listed as an accepted regression |
| `expected-reject/` | cases that must reject (~10) | is rejected |
| `regression/` | every later material defect (empty at start; see its README) | publishes correctly or rejects as expected; rejecting a `publish` case needs an accepted regression |
| `development/` | free tuning data (~20) | reported only; correct publications count toward the per-site minimum |
| holdout (outside the repo) | ~20 final recipes kept by the operator | at least 15 publish, zero material errors among them; an empty holdout fails |

Each active site in `catalog/sites.yaml` needs at least 3 evaluation recipes (any part, including
the holdout) that publish with zero material errors. Every case must be reviewed by the household
(`reviewed: true`) before a candidate set can qualify.

## The holdout

The holdout lives **outside this repository**, kept and backed up by the operator so development
and coding agents cannot tune to it. Pass it with `--holdout DIR`. `DIR` holds case directories
directly (plus an optional `README.md`), in exactly the case format below. Case IDs must not
collide with the checked-in set. Without `--holdout`, qualification fails. A holdout recipe that
informs a fix moves to `regression/` and is replaced.

## Case format (version 1)

A case is a directory named by its case ID (lowercase kebab-case, unique across all parts and the
holdout; by convention `<site-id>-<recipe-slug>`). It contains exactly two files.

### `source.json` — the source node input

```json
{
  "format_version": 1,
  "site": "love-and-lemons",
  "url": "https://www.loveandlemons.com/banana-bread/",
  "note": "optional: how the node was captured",
  "node": { "@type": "Recipe", "recipeIngredient": ["..."], "...": "..." }
}
```

- `site` must be a registry site ID; `url` must be HTTPS on one of that site's hosts.
- `node` is the raw Schema.org `Recipe` object. Its `recipeIngredient` strings are the source lines.
- Unknown fields are rejected.

### `expected.yaml` — the expectation

Common fields:

| Field | Meaning |
| --- | --- |
| `format_version` | `1` |
| `reviewed` | `true` only after the household reviews the case |
| `draft_note` | required while `reviewed: false` (what to check); forbidden once reviewed |
| `note` | optional lasting context |
| `outcome` | `publish` or `reject` (`must-pass` is always `publish`; `expected-reject` always `reject`) |
| `categories` | optional extra report categories: `additive`, `compound`, `alternative`, `ai_created_item`, `non_shopping`, `presence_only`. Categories visible in the expected document are added automatically. |
| `instructions_complete` | the household's judgment that the steps are complete enough to cook from; must be `true` for `publish` |
| `instructions_note` | optional explanation of that judgment |

For `outcome: publish`:

- `document` — the expected interpretation in the **catalog content format v1**
  (`internal/ingest/catalog.go`). It must pass the publishability check, and its `source_lines` must
  equal the node's `recipeIngredient` list exactly.
- `grocery` — the expected grocery output: the result of the existing aggregation
  (`internal/grocery.Aggregate`) for the document added once to an empty week. The parser
  recomputes it and rejects the case if the recorded lines differ, so the lines are always the
  aggregator's output, never hand-computed. Amounts are reduced rationals (`"5/8"`), and
  `source_lines` lists the source line behind every contribution.

For `outcome: reject`:

- `reject.codes` — issue or reason codes the rejection is expected to report, for example
  `missing_note_dependency`, `incoherent_alternative`, `instruction_ingredient_discrepancy`, or an
  unresolved requirement's reason. A rejection that misses a listed code is reported for review;
  qualification only requires that the case rejects.

## Candidate sets

A candidate set is a directory of files named `<case-id>.yaml`, one per case:

```yaml
format_version: 1
document: { ... catalog content format v1 ... }   # the interpretation, if any
rejections:                                        # recipe-level rejections, if any
  - code: missing_note_dependency
    detail: optional text
```

A candidate publishes only if it has a document, no rejections, and the document passes
`ingest.CheckPublishable`. A case with no candidate file counts as rejected (`no_candidate`).

## Material and cosmetic differences

A published candidate is compared with the expected document per source line (requirements are
matched by grocery item, then in order) and per instruction step (aligned in order).

Material errors (§6.2): `missing_requirement`, `extra_requirement`, `wrong_item`,
`wrong_shopping_mode`, `wrong_amount`, `wrong_unit`, `wrong_range`, `wrong_package`,
`wrong_optional`, `wrong_unspecified`, `wrong_non_shopping`, `wrong_preparation` (words or numbers
changed), `missing_step`, `wrong_step` (an extra or changed step), `wrong_source_lines`,
`wrong_grocery_output`, and `unexpected_publication` (an expected rejection published).

Cosmetic differences are reported but never fail: item name, store section, relationship, span,
note, section layout, recipe metadata, and preparation or step text that differs only in case,
spacing, or punctuation.

## Accepted regressions

`--accepted-regressions FILE` names must-pass or regression cases allowed to reject under this
release. It is honored only for cases it lists:

```yaml
format_version: 1
accepted:
  - case: love-and-lemons-banana-bread
    reason: stricter alternative rule rejects the butter-or-oil line until format v2
```

## Running

```sh
task catalog-qualify -- --candidates DIR --holdout HOLDOUT_DIR [--accepted-regressions FILE]
```

The command prints the criteria, results by site and by category, and every case's differences,
and exits non-zero unless every criterion passes.
