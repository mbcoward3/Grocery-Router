# Regression cases

Empty at the start of slice 1. **Every material defect lands here** (CATALOG_INGEST_SPEC.md
§6.3, §6.5): when an audit, a review, or a holdout result finds a material error (§6.2) in a
published recipe, add that recipe here as a case directory before the corrected release is
qualified. A holdout recipe that informs a fix also moves here and is replaced in the holdout.

A regression case uses the same format as every other case (`source.json` + `expected.yaml`; see
`../README.md`). Its outcome may be `publish` or `reject`. Because a regression recipe got here by
publishing, `catalog-qualify` treats a regression case that now rejects as a failure unless the
accepted-regressions manifest lists it with a reason.
