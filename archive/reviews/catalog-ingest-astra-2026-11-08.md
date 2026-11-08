# Catalog ingest spec — Astra reviews (2026-11-08)

Historical review record for `CATALOG_INGEST_SPEC.md`. Non-authoritative. Section references in
Appendix A point to the first draft; Appendix B points to revision 2. Both drafts exist only in this
review history; their outcomes are folded into the current specification.

## Appendix A — Astra review (2026-11-08)

Retained verbatim until this revision is approved. Section references inside it refer to the first
draft.

**Recommendation: revise before authorizing implementation of the full pipeline.** The architecture
is credible, but the proposed quality contract is not yet sufficient to call its output complete or
grocery-ready. This review is commentary on the draft, not approval, a replacement specification,
or authorization to implement the recommendations below.

Review basis: the current product and authentication contracts, current ingestion and aggregation
code, and the pilot branch at `a4f676825ddfd572e1f74b6dc1189720eb2d467c`. The pilot was inspected
read-only. No fresh crawl, model benchmark, or end-to-end pipeline run was performed. Consequently,
the publication rate, accuracy, cost, and throughput targets remain unproven.

### 20.1 Overall assessment

The strong parts should survive revision: offline-only AI, deterministic grocery generation,
source evidence, bounded acquisition, shared catalog versus household ownership, explicit
rejections, and staged delivery. PostgreSQL and ordinary Go are reasonable foundations for this
scale; the difficult part is interpreting recipes faithfully and recovering from mistakes.

The current draft is likely to produce a large catalog of structurally valid recipes. It has not
established that those recipes will reliably produce correct groceries or complete cooking
instructions. Those are different success conditions. The central mistake is substituting
mechanical completeness for the human judgment that made the original corpus trustworthy.

Automated publication can be a sound direction. It needs empirical qualification of the pipeline,
conservative handling of ambiguity, and a way to stop or withdraw bad output. Per-recipe human
approval does not need to return, but quality measurement must have operational consequences.

### 20.2 Blocker: preserving every line does not preserve every requirement

Sections 9.2, 11, and 18 protect ingredient-line presence, but a line can contain multiple
requirements. The pilot already demonstrates the failure:

- `catalog/candidates/banana-bread.candidate.json` preserves the source line
  `½ cup chopped walnuts (plus 2 tablespoons for topping)` but structures only `1/2 cup`.
  The additional two tablespoons survive as a warning, not a grocery contribution. The cooking
  instructions explicitly use them. Under the current US unit table, the complete requirement is
  `5/8 cup`, or two separately traceable contributions totaling that amount.
- The same candidate has butter `plus more for the pan`; the additional unspecified requirement
  also needs an explicit representation or an explicit unsupported outcome.
- `catalog/candidates/sausage-and-peppers.candidate.json` proposes thyme for a source line containing
  both thyme and rosemary. Its review correctly flags ambiguity, but merely retaining rosemary in
  source text would not put rosemary on the grocery list. The instructions also mention salt and
  pepper absent from the displayed ingredient list.

These examples do not prove that the new parser will repeat the errors. They prove that the
proposed gates would not reliably detect them, and that the pilot's blockers are not all harmless
administrative friction.

**Required change:** define source-line-to-requirement coverage. One line may produce multiple
structured requirements, each traceable to the original line. Distinguish additive amounts,
alternatives, shared amounts, and explanatory equivalents. Unrepresentable compounds must remain
unresolved instead of becoming a plausible single item. Add end-to-end fixtures that assert the
actual grocery output, including these pilot examples. Define how ingredient requirements found
only in instructions are detected and handled; do not claim complete recipes based solely on
`recipeIngredient` coverage.

### 20.3 Blocker: missing information is being converted into apparently valid data

Section 9.2 permits `1 garlic` to become `unspecified` when the unit cannot be resolved. The
current quantity model then cannot retain the numeric amount in structured form. That is an
unresolved parse being converted into an intentionally unspecified requirement, which can pass
publication gates. The source did not say “to taste.”

Likewise, §9.1 explicitly removes references to absent notes without rejecting the recipe. A
missing note might contain an optional serving suggestion, but it might also contain essential
preparation, an ingredient amount, or instructions for the selected alternative. Removing the
reference makes the result look more complete than the evidence supports.

**Required change:** separate `intentionally_unspecified`, `unresolved`, and `unsupported` in
pipeline state. Only the first is publishable as an unspecified quantity. Do not authorize an AI
to infer a garlic head through the undefined phrase “literally implied.” Preserve missing-note
references and reject unresolved essential dependencies. If JSON-LD-only acquisition cannot
recover required information, rejecting the recipe is consistent with this phase's scope.

The same distinction is needed for `for serving` versus optional, `and` versus `or`, and
`divided` versus additional amounts. Selecting the first alternative is acceptable only when its
quantity and cooking instructions remain coherent. A local ingredient-line decision cannot
always establish that.

### 20.4 Blocker: the quality target is too weak for the user experience

Section 18 allows 95% ingredient-line accuracy. As an illustration, if a ten-ingredient recipe
had independent 95%-accurate lines, only `0.95^10`, approximately 60%, would have every line
correct. This is not a prediction of actual pipeline performance; errors will not necessarily be
independent. It shows why line accuracy is not a sufficient product acceptance measure.

The proposed sample also excludes instruction completeness and can be dominated by easy lines,
popular sites, and repeated cached mappings. Fifty random recipes are a useful initial audit,
not strong evidence of reliability across thousands of recipes and unfamiliar sources.

**Required change:** measure recipe-level grocery correctness, missing/extra requirements,
quantity and package correctness, purchased-form correctness, and instruction completeness.
Separate material errors from cosmetic differences. Evaluate by site and by difficult parsing
category, including AI-created items and non-shopping decisions. Maintain an independently
reviewed holdout set that is not used to tune rules, plus a regression set for discovered defects.

Replace “spot-checks never block publication” with pipeline-release qualification and regression
stop rules. Recipes may still publish automatically under a qualified release. A release or site
that fails quality checks should stop publishing until corrected. This avoids individual approval
while preserving accountability. Define a withdrawal policy for already-published affected data.

Treat the 70% publication and 85% rule-resolution figures as measurements until supported by
evidence. Neither is inherently a quality goal. Publication quotas can reward permissive parsing;
rule coverage can reward brittle special cases. Clarify whether an AI-origin cached result counts
as AI, a cache hit, or a rule; it should retain its original provenance.

### 20.5 Blocker: unchanged ingestion reuse is not supported by current code

The reuse promised in §§9, 11, and 17 needs an explicit refactoring boundary:

- `internal/ingest/document.go` requires `Status == "verified"` and a valid `ApprovedOn` date.
  Those fields currently describe a human-approved bootstrap document. Catalog automation should
  not fabricate a human approval date to satisfy them.
- `Document.Validate` checks unit-key syntax, but actual unit existence is checked later during
  import. It does not prove semantic fidelity, known units, or full database importability by
  itself. Rational values can also pass document parsing before failing 64-bit storage conversion.
- `internal/ingest/import.go: Import` requires an empty household recipe table. It cannot directly
  adopt one recipe into the existing family's collection.
- Grocery-item reuse currently rejects conflicting names, sections, or shopping modes. Matching
  catalog and household items by key alone does not resolve those conflicts.

**Required change:** share recipe-content validation while separating bootstrap human approval
from automated catalog publication provenance. Define a versioned serialized content contract and
catalog-specific validation. The later adoption phase needs a transactional single-recipe import
path and explicit mapping/conflict policy. It need not be implemented now, but “reuse unchanged”
must be removed and compatibility should be exercised through a small integration fixture.

Do not weaken the existing household bootstrap validator to accommodate the catalog.

### 20.6 High risk: vocabulary and cache decisions can spread errors globally

Section 9.3 removes preparation and size adjectives before resolving a name. Some descriptors are
dispensable preparation; others define what must be purchased. A generic normalization rule needs
explicit protection for distinctions such as fresh versus dried, canned versus fresh, sweetened
versus unsweetened, and whole versus ground. A string-similarity shortlist is not evidence that
two foods are interchangeable.

A cached name error can affect thousands of recipes. A provisional flag merely reports that
risk; it does not constrain it. AI-selected `presence-only` mode can hide quantitative shopping
requirements, and an incorrect non-shopping decision can remove a requirement altogether.

Identical line text also need not have identical interpretation across source contexts. The
existing volume table explicitly uses US measures. A multi-site catalog needs declared source
measurement conventions rather than assuming every `cup` or `tbsp` means the seeded unit. Recipes
may also refer to an ingredient prepared elsewhere in the same recipe; resolving an isolated
name loses that context.

**Required change:** define protected identity attributes, explicit measurement conventions, and
a narrowly allowed non-shopping policy. Restrict automatic presence-only assignments. Cache keys
must include every dependency that can change interpretation: relevant source context, parser
and prompt versions, model configuration, vocabulary/alias policy, and unit conventions. Do not
blindly share ambiguous decisions globally.

Provide correction and invalidation operations that identify all affected candidates. Define
operator override precedence so reprocessing does not undo corrections. Item merges must update
embedded document JSON, derived item rows, resolutions, and digests consistently. Incorrect
merges also need a recovery path; a merge-only vocabulary cannot repair mistaken conflation.

### 20.7 High risk: de-duplication is too destructive and underspecified

Section 12 suppresses recipes with the same normalized name and 90% grocery-item overlap. Two
recipes can share all their ingredients while having substantially different proportions,
instructions, yields, or cooking methods. Conversely, syndicated copies can have different names.
“Overlap” also needs a defined denominator and treatment of repeated items.

**Required change:** automatically collapse only well-supported exact equivalents. Record fuzzy
matches as duplicate candidates or variant groups until the equivalence rule has been validated.
If choosing a representative, prefer completeness and evidence quality before site priority.
Preserve every source attribution. Dedupe should not erase meaningful alternatives needed by the
future recommender.

Canonical URLs are untrusted source assertions, not unconditional primary keys. Define conflict
handling for missing, multiple, cross-origin, or incorrect canonical links. Separate fetched-page
identity, source-node identity, duplicate grouping, and stable catalog identity. A JSON array
position is not a stable node identity when extraction changes or source nodes move.

### 20.8 High risk: reprocessing and recovery lack a complete contract

Section 13 promises that versioned stages can rerun without fetching. Stored JSON-LD permits
standardization and node-extraction replay, but omitted HTML prevents replaying fixes to HTML
script discovery, canonical-link extraction, or other page-level acquisition behavior. A body
hash proves identity only when the body is available; it cannot reconstruct the evidence.

The statement in §14 that evidence can be rebuilt by fetching is also too strong. A later fetch
can return changed content, a block, or no page. It creates new evidence, not the old evidence.

**Required change:** either retain bounded compressed HTML, potentially with a retention policy,
or explicitly narrow the replay guarantee and accept that some fixes require new evidence.
Define backup and restore expectations for the original evidence, paid AI resolutions, operator
corrections, and publication state. This does not require a separate cluster or an elaborate
storage system.

Also specify what happens when a previously published recipe fails a new build: retain the last
known-good release, withdraw it, or apply a failure-class-specific rule. Store enough publication
provenance to reproduce and roll back a bad release. Candidate history may provide much of this;
it does not require a generalized event-sourcing architecture.

### 20.9 High risk: resumable AI work needs persisted request state and budget reservations

The proposed working tables omit explicit batch/request lifecycle records. A crash after batch
submission but before local completion can cause duplicate work and charges. Batches can finish
partially, expire, and return results in a different order from their inputs. Official documentation
requires matching results through `custom_id` and describes partial results on expiration:
[OpenAI Batch API](https://developers.openai.com/api/docs/guides/batch).

**Required change:** persist request identity, submission state, provider batch IDs, per-request
outcomes, usage, and reconciliation state. Resume by recovering existing work before submitting
new work. Define retryable errors separately from rejected content and budget deferral.

Reserve estimated spend for submitted/in-flight work before admitting more work, account for
retries, and reconcile actual usage. A ceiling checked only after results arrive cannot bound a
large outstanding batch. Define whether the monthly budget is enforced across runs; currently
only per-run flags are specified.

The claim that cost scales only with distinct ingredient names is incomplete: §10 also pays to
parse distinct unresolved ingredient lines. Measure both, along with reprocessing cost and human
maintenance time. Do not build a complicated parser solely to maximize the percentage of rules
without comparing its accuracy and maintenance cost against bounded offline AI fallback.

### 20.10 High risk: structured output is not a semantic or injection guarantee

The sentence in §10 that the output schema keeps injected instructions from having any effect is
incorrect. A model can return schema-valid but wrong quantities, grocery mappings, or non-shopping
decisions. The official documentation explicitly states that structured outputs can still contain
mistakes: [OpenAI Structured Outputs](https://developers.openai.com/api/docs/guides/structured-outputs).

Keeping models offline and tool-free limits the consequences, but the remaining consequence is
catalog corruption. Require evidence-grounded decisions, explicit abstention, constrained identity
choices, conservative non-shopping rules, and adversarial fixtures. Retain the model output and
validation outcome needed to diagnose a bad resolution, subject to a documented retention policy.

### 20.11 Medium risk: acquisition needs several operational decisions before scaling

The pilot provides useful safe-fetch code, but sitemap discovery is a new workload. Specify:

- maximum sitemap/index depth, URL count, decoded size, and cycle handling;
- allowed origins for sitemap entries and redirects, including explicit subdomain policy;
- behavior when robots retrieval fails or is unavailable;
- rate limits shared across redirects, robots requests, sitemaps, and overlapping operator runs;
- persistent site pauses and explicit resume policy, so restarting a command does not immediately
  undo a response to blocking or throttling;
- crash recovery, work leases or a single-run lock, atomic publication, and bounded database use;
- a source eligibility record covering intended storage/display/reuse, plus removal and withdrawal
  procedures before catalog content becomes user-visible.

The last point is an unresolved product/operating requirement, not a legal conclusion that the
listed sources may or may not be used. Crawl politeness alone does not settle the intended reuse
policy.

There is also a factual correction: the pilot manifest contains 16 source URLs across 13 distinct
origins, not 16 distinct sites. The initial registry should deduplicate those origins and discover
the actual sitemap and crawl policy for each.

### 20.12 Product risk: catalog size can outrun proof of household value

The catalog-only boundary is sensible, but tens of thousands of recipes should not be the first
proof of success. The family needs useful new meals with trustworthy groceries. A large catalog
can still contain unsuitable yields, side dishes, desserts, near-identical variants, or instructions
that make sense only alongside the source page.

Preserve source category, cuisine, yield, timing, and other useful evidence now; the raw node is
already suitable for much of this. There is no need to build embeddings or taxonomy in this phase.
Before scaling, verify a small representative set through the existing grocery aggregation path
and review the results as someone actually planning, shopping, and cooking would use them.

The future adoption contract also needs explicit trust semantics. D040 preserves individual
approval for the household corpus, while D043 anticipates automatic suggestions entering a week.
Clarify whether adoption confers selectability without human approval and how its automated origin
is represented. Do not silently reinterpret `verified` to mean both kinds of trust.

Mapping by grocery key needs conflict handling, and a trial recipe needs a household snapshot
before it can generate groceries under the current tenant model. “Joins when kept” can later mean
collection membership, but it cannot leave the week's ingredients without an owned snapshot.
Finally, a recipe not swapped out is evidence of retention, not proof it was cooked or liked.
Those are later-phase decisions, but this spec must not promise compatibility without naming them.

### 20.13 Recommended delivery change

1. **Qualify the contract first.** Build a reviewed evaluation set from the family corpus, pilot
   failures, and varied new source samples. Include full expected grocery output and instruction
   completeness judgments. Set material-error criteria before tuning for publication rate.
2. **Acquire a bounded corpus.** Implement discovery and evidence capture for a few sources and a
   few hundred recipes. Exercise interruption, retry, and replay behavior. Measure source coverage.
3. **Compare interpretation approaches.** Evaluate deterministic parsing and bounded AI fallback
   on the same held-out data. Establish source-line coverage, context-sensitive caching, and
   conservative rejection. Measure cost per correct usable recipe, not just cost per publication.
4. **Qualify publication and recovery.** Exercise import/aggregation compatibility, vocabulary
   correction, dedupe precision, bad-release withdrawal or rollback, and crash recovery. Publish
   automatically only under a qualified pipeline release.
5. **Scale after demonstrated usefulness.** Expand sources when the resulting groceries and
   cooking instructions are reliable. Leave recommendation, user-facing adoption, and household
   editing to separately approved phases, but settle their data/trust boundaries before freezing
   the catalog contract.

Smaller consistency fixes: §11 lists `ai_budget_deferred` as a rejection despite §10 requiring
deferral; §16 places publication in slice 3 before catalog publication tables in slice 4; and the
new D039–D043 decision records should clearly distinguish proposed directions from approved
decisions while this specification remains a draft. These should be corrected alongside the
substantive issues.

**Approval threshold:** resolve requirement coverage, ambiguity handling, release-level quality
control, ingestion compatibility, and correction/recovery semantics before authorizing the full
build. The long-term goal is an automatically maintained catalog that earns trust through measured
results. The current draft has a useful foundation, but its strongest quality claims exceed its
specified checks and the available pilot evidence.

---

## Appendix B — Astra second-pass review of revision 2 (2026-11-08)

**Verdict: materially improved; a credible direction for a bounded implementation, but several
publication-critical contracts still need revision.** This is a review, not authorization to
implement. Section references below refer to revision 2. No fresh crawl, live-model evaluation,
or implementation benchmark was performed.

The requirement model, interpretation states, retained HTML, exact-only duplicate suppression,
content/approval separation, proposed-decision labels, and usefulness review are substantive
improvements. Postponing Batch is also sensible. The first review's architectural objections are
largely addressed. The remaining issues concern how these policies interact, including several
new rules that can undermine the intended guarantees. The disposition table's “Accepted” entries
should not yet be read as “fully resolved.”

### B.1 High — Instruction matching must not create unsupported requirements (§9.5)

The new cross-check promotes salt and black pepper mentions to intentionally unspecified
requirements. That is too broad:

- “Do not add salt” contains the item name but requires no salt.
- “Add 2 teaspoons salt” contains an explicit amount, which the rule would discard.
- “Serve with pepper if desired” contains conditional language and does not establish that
  generic pepper means black pepper.
- Mentions of alternatives, equipment names, serving suggestions, and recipe-produced components
  require interpretation; string occurrence alone cannot determine ingredient use.

The matching vocabulary is also incomplete by construction: an unknown ingredient cannot be
found by searching only known grocery names. Passing this check is therefore not proof of
instruction/ingredient consistency. Automatically rejecting every unmatched known name can also
reject complete recipes because of incidental mentions.

**Change:** treat lexical matches as possible discrepancies. Either reject ambiguous
instruction-only requirements in this first phase, or pass them through the same quantity,
optionality, identity, and source-grounding contract as ingredient lines. Auto-add only a narrowly
specified affirmative usage pattern. Preserve explicit quantities. Use typed provenance such as
`ingredient_line` or `instruction_step`, with a location and span; §12 currently requires a source
line position even though instruction-derived requirements have no ingredient line. Add positive,
negative, conditional, and quantified instruction fixtures. Describe the cross-check as a useful
detector with measured limitations, not a completeness guarantee.

### B.2 High — Qualification can pass with almost no usable output (§13)

Qualification scores only holdout recipes the release publishes, and regression recipes may all
be rejected. A release that publishes one easy holdout recipe correctly and rejects the rest can
satisfy the written criteria. A release publishing none has an undefined denominator. A recipe
that previously worked can become unsupported without causing qualification failure.

The lighter sample is reasonable for a pilot, but its implications need to be explicit. With 30
published holdout recipes, one material error still passes 95%. The subsequent audit permits
1 error in 10 site samples and 2 in 20 release samples without stopping, both observed error rates
of 10%. That is inconsistent with a 95% correctness target. Missing requirements get special
treatment during qualification but no equivalent immediate stop during audits.

**Change:** define a minimum number of published holdout recipes, an empty-denominator failure,
and a small must-pass set covering supported cases such as the additive walnut requirement.
Keep separate expected-reject fixtures. Unsupported complexity can still reject safely; rejecting
previously supported examples must be reported and deliberately accepted rather than silently
counted as regression success. Require site/category coverage for the scope being qualified.

For the initial small pilot, zero observed material errors in the qualification sample is a
clearer release criterion, while explicitly making no population-level accuracy claim. Align
audit actions with qualification severity, and define the action on a single missing requirement
or systematic mapping defect. State when audits are due, what happens if they are not completed,
and whether a stopped site can return only after its own failure cases pass.

The checked-in holdout is a benchmark, not automatically a blind test. Once its expected answers
inform fixes, promote those examples to regression and replenish a final independent sample.
This is especially relevant when coding agents can read all checked-in evaluation files.

### B.3 High — Context-sensitive caching still depends on the model noticing the problem (§10.5)

A global result can be reused before the AI sees the second recipe. Consequently, relying on the
AI to report context dependence on the first occurrence cannot protect later contexts. First-option
selection explicitly depends on instructions under §9.3, but the global line cache key contains no
instruction context. The same alternatives can be coherent in one recipe and incoherent in another.

Recording only vocabulary items considered also misses new aliases, newly introduced better
matches, normalization-policy changes, and changes to non-shopping rules. None necessarily changes
an item in the old shortlist. There is an additional provenance issue: normalized text may share a
cache entry across raw strings with different whitespace or entities, making cached raw character
spans invalid for the second source line.

**Change:** cache context-free parsing separately from recipe-level decisions. Recheck alternative
coherence and instruction dependencies for every recipe. Scope AI interpretation by relevant
recipe context by default unless a defined context-free contract permits broader reuse. Version
the alias, normalization, measurement, and shopping policies as cache dependencies. Rebind spans
to each original source through a defined normalization mapping. Withhold global reuse when
that mapping or interpretation is uncertain.

### B.4 High — Releases need an immutable dependency boundary (§5, §10, §14)

A release includes a vocabulary snapshot, but the build creates provisional vocabulary items.
The draft does not say when the snapshot freezes or whether later catalog-build runs can grow it
under the same qualified release. Qualification can otherwise cover one vocabulary while
publication uses another. Grocery merges also rewrite references and digests before “the next
release,” leaving unclear whether published or historical data is being changed in place.

**Change:** define a working build followed by a frozen release manifest and qualification of that
manifest. State which changes require requalification. Qualified dependencies and prior candidate
documents must remain immutable. Stage vocabulary corrections and derived documents together,
then switch them together at publication. If one qualified pipeline configuration may process
new recipes continuously, distinguish that configuration from publication batches and define how
new provisional items and new sites are admitted. Either design can work; the spec needs one.

### B.5 High — Rollback and export do not yet restore a complete state (§14.3–§14.4)

Suppose release R1 publishes A, and R2 changes A and introduces B. “Republish the previous
release's documents” can restore A but has no previous document for B. B must be withdrawn or
otherwise removed from the active publication set. Status changes, attribution, duplicate
representatives, vocabulary changes, and derived item rows also need consistent restoration.

Rollback must not undo a subsequent operator/site block and resurrect content explicitly made
ineligible. Exact deduplication also means a recipe can have multiple eligible sources; withdrawing
one source needs a defined effect on the representative and remaining attribution.

A publication log containing only keys and digests cannot itself reconstruct documents.
Candidates might retain the required bytes, but the retention/reference contract is unstated.
Export currently includes only AI results and that log, with no restore command or procedure.
It does not establish recovery of evidence, published documents, identities, or vocabulary.

**Change:** record sufficient immutable before/after state or publication membership to restore
new, changed, and withdrawn keys, including dependencies. Preserve current eligibility blocks
through rollback. Define retention for referenced documents and evidence. Choose either a tested
database backup/restore procedure or a self-contained export/import format for the intended
recovery scope. A manual bounded database backup is sufficient for the pilot; a new generalized
backup platform is unnecessary. Acceptance should restore into an empty database and verify
catalog identity, documents, and generated groceries.

### B.6 High — Restore an explicit per-candidate publication gate (§12, §20)

Revision 1 had a dedicated gate list. Revision 2 distributes checks across sections and never
fully restates the gate that every candidate must pass. Non-English content remains excluded but
has no specified detection outcome. The acceptance criteria prohibit unresolved requirements,
but not explicitly unresolved source lines with zero requirements. A recorded outcome of
`unresolved` is still an outcome, so line accounting alone is insufficient.

**Change:** add one normative publication predicate covering shared content validation, supported
language, nonempty ingredients and instructions, known units and storage bounds, every source
line's publishable disposition, every requirement's provenance and valid state, alternative/note
checks, vocabulary consistency, and release/site eligibility. Only a validated heading may yield
zero requirements without an explicit supported reason. Reference this predicate from publishing
and acceptance tests. Release qualification must never substitute for candidate validation.

This need not restore arbitrary ingredient/step count thresholds from revision 1; resource bounds
and semantic completeness can be specified separately.

### B.7 Medium — A closed shopping policy still needs context (§10.4)

The closed list is safer than allowing arbitrary AI decisions, but `ice` is not universally
non-shopping. Ice used to chill a bowl and an explicit purchased bag of ice are different cases.
Likewise, generic water cannot erase an explicitly purchased water requirement. Item identity
and use must preserve that distinction.

Assigning a new item's shopping mode from its first observed unit is undefined for an
intentionally unspecified quantity and package quantities, and can depend on processing order
when an item appears both counted and measured. The existing aggregator can handle compatible
quantities without inventing new grocery identities for each unit.

**Change:** constrain non-shopping to explicit supported item/use rules, with tests proving that
purchased forms remain shopping requirements. Define a stable shopping-mode policy for unknown
quantities, packages, and mixed usage. Preserve original numeric recipe requirements even when
shopping display is presence-only. Also narrow §13.2's claim that preparation phrasing is
cosmetic: rewording is cosmetic only when it preserves meaning; whole versus minced or a changed
preparation order can change what is cooked.

### B.8 Medium — Resolve operational contradictions without expanding the platform

- §7.1 allows 50 MiB decoded sitemaps while §7.2 and §8.1 impose a 5 MiB decoded fetch/evidence
  limit. Define separate bounded fetch profiles, or intentionally accept a smaller sitemap limit.
- §7.3 treats robots 4xx as unrestricted while §7.4 persistently pauses on 403/429. State which
  rule takes precedence for robots responses, and check robots per actual allowed origin.
- The five-second rate limiter needs a defined host/site key when allowed hosts overlap; otherwise
  two registry entries can independently schedule the same host.
- A local AI request record does not by itself resolve a crash after remote completion but before
  local persistence. Specify an `outcome_unknown` policy: retain its spend reservation and either
  recover the response where supported or permit a bounded, accounted retry. Do not assume that
  merely naming a field `idempotency key` guarantees provider-side deduplication.
- Define exact duplicate “requirement sets” as preserving multiplicity. One cup of sugar in each
  of two sections is not equivalent to one cup once. Grocery equivalence must not erase cooking
  section allocation or preparation differences.

These are implementation contracts to settle while writing the affected slices; they do not
justify a distributed workflow engine or further speculative infrastructure.

### B.9 Recommended next decision

I would now support authorizing only the evaluation/content-contract slice and bounded
acquisition once their immediate contradictions are resolved. I would hold automatic publication
until B.1–B.6 are incorporated and demonstrated by fixtures. That is a bounded path forward,
not a request for another complete architecture rewrite.

Keep the operational burden visible: the written audit policy implies up to 130 initial site
checks plus 20 release checks, in addition to evaluation-set review. Decide whether that initial
effort is worthwhile, and let overlapping samples count where appropriate. Start with a smaller
site subset if needed; the 13-origin registry need not mean qualifying all 13 simultaneously.

The central question has improved from “will this accept plausible but wrong recipes?” to “does
this release process actually enforce the quality and recovery guarantees it names?” Revision 2
is substantially closer. Small, explicit contracts and executable failure examples should now
replace further broad feature additions.
