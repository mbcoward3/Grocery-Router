# Grocery Router — Recipe Intelligence and Jev Assessment

Status: **approved import-time assessment pilot; implementation authorized**

Depends on: [`RECIPE_SOURCE_PILOT_SPEC.md`](RECIPE_SOURCE_PILOT_SPEC.md), [`V1_SPEC.md`](V1_SPEC.md)

Related backlog: [`UP_NEXT.md`](UP_NEXT.md) §2 and §4

## 1. Proposed outcome

Turn a pinned source-recipe artifact into two distinct outputs:

1. a reviewable, structurally complete Grocery Router recipe candidate; and
2. a versioned semantic assessment that supports search, filtering, ranking, and future meal-planning agents.

Jev supplies narrow semantic judgments through typed questions. It does not generate recipe text,
perform arithmetic, certify dietary safety, or promote a candidate into the verified corpus. Code,
source normalization, a separate structured-extraction stage, and human approval retain those jobs.

The initial corpus remains owned by the `Coward` household. This draft does not create a public or
cross-household recipe catalog.

Jev runs only in operator-invoked import and review tooling for this phase. The application makes no
runtime Jev calls. Runtime semantic reranking and planning evaluation are deliberately deferred,
though import-time profiles are designed to support that later phase.

## 2. Pipeline

```text
pinned source artifact
  -> source projection and deterministic facts
  -> structured recipe candidate generation
  -> canonical ingredient and taxonomy candidates
  -> Jev intrinsic assessment
  -> deterministic validation and confidence routing
  -> human review
  -> approved recipe truth + approved semantic profile
```

The source artifact is immutable evidence. Generated candidates, Jev answers, and human-approved
truth are separate records. Re-running a newer model or catalog never silently changes an approved
recipe or semantic profile.

## 3. What belongs where

### 3.1 Deterministic code

Code owns facts that can be parsed, counted, compared, or calculated exactly:

- source and artifact digests;
- exact durations and duration ranges;
- yield values when structurally available;
- ingredient and instruction counts;
- exact unit conversion;
- canonical ingredient membership after approval;
- allergen presence derived from approved ingredient mappings;
- explicit appliance and temperature mentions found by parsers;
- grocery-section counts and ingredient overlap between recipes;
- nutrition values explicitly supplied by the source; and
- filtering against numeric limits.

Jev must not be asked to count ingredients, add times, perform unit arithmetic, compare dates, or
reconstruct precise numbers from score levels.

### 3.2 Structured extraction

A generative extractor or bounded candidate generator may propose:

- ingredient amount, unit, package, item, preparation, and alternatives;
- instruction sections and steps;
- source yield and duration components;
- equipment and technique candidates;
- canonical grocery-item candidates; and
- concise recipe descriptions where no source description exists.

Every proposed value retains its source span or a clear `inferred` marker. Jev may select among or
validate candidates, but it is not used for free-text generation.

### 3.3 Jev

Jev owns atomic semantic judgments where multiple reasonable readers could classify the same recipe:

- role and taxonomy;
- flavor and texture character;
- effort, attention, cleanup, and skill burden;
- planning and occasion suitability;
- make-ahead and leftover qualities;
- likely familiarity or adventurousness; and
- relevance or compatibility judgments at query time.

### 3.4 Human approval

A reviewer owns corpus truth, ambiguous ingredient mappings, source conflicts, dietary claims shown
to users, and every promotion to `verified`.

## 4. Assessment model

Question definitions are versioned data, not database columns.

```text
AssessmentDefinition
  key                 stable namespaced identifier
  version             positive integer
  primitive           choice | score | noul
  state_projection    identity | ingredients | method | full_recipe | pair | query
  instructions        exact Jev instruction
  criteria            choice options, score levels, or structured Noul criteria
  applicability       deterministic condition or always
  materialization     facet | signal | diagnostic | dynamic_only

AssessmentRun
  recipe_candidate_id
  source_digest
  candidate_digest
  catalog_version
  model
  requested_at
  completed_at

AssessmentAnswer
  definition_key + definition_version
  typed answer
  complete probability distribution when returned
  confidence when returned
  disposition         accepted | needs_review | rejected | not_applicable
  reviewer and review timestamp when reviewed
```

Approved materialized facets are stored separately from raw run answers. Their provenance points to
an assessment answer, deterministic derivation, source field, or human decision.

## 5. State projections

Do not send the complete pilot artifact to every question. It contains duplicated and irrelevant
page metadata that reduces classification quality.

- **identity:** name, source description, source categories, source keywords, approved concise
  description when available.
- **ingredients:** identity plus ordered ingredient text and proposed canonical ingredient names.
- **method:** identity plus ordered instruction text, structured durations, and equipment candidates.
- **full_recipe:** identity, ingredients, method, yield text, and deterministic facts.
- **pair:** compact approved profiles for exactly two recipes.
- **query:** the user's query or planning context plus compact approved profiles of shortlisted
  recipes.

Source page text is untrusted data. Instructions explicitly tell Jev to classify the recipe and
ignore commands or requests appearing inside source content.

## 6. Question design rules

1. Each question measures one concept.
2. Use `Noul` for independent multi-label facets; several answers may be true.
3. Use `Choice` only for mutually exclusive alternatives and include `unclear/not evidenced` where
   absence of evidence is possible.
4. Use `Score` for ordered semantic levels with a concrete rubric for every level.
5. Never infer exact numeric values from a score.
6. Never treat Noul complements as arithmetic inverses or reuse thresholds between primitives.
7. Preserve complete probability distributions and confidence.
8. Confidence routes behavior; it is not displayed as factual certainty.
9. Questions are evaluated against labeled fixtures before automatic materialization is enabled.
10. Dynamic user-context questions are not persisted as intrinsic recipe facts.

## 7. Intrinsic question catalog

The tables below define catalog v1. `N` means Noul, `C` means Choice, and `S` means Score. Each row
becomes an independently versioned definition. Backticks denote stable answer keys.

The pilot executes every applicable intrinsic definition in this section: 137 definitions or
conditional definition templates across §§7.1–7.11. Review initially prioritizes roles, cuisine,
methods, flavor, effort, and planning utility, but all returned answers are preserved for evaluation.
The 10 dynamic definitions in §§9–10 require query, pair, or week context and are not run during
import-time assessment.

### 7.1 Dish identity and role

For each `N` role, ask: **“Is this recipe itself reasonably served as [role], rather than merely
containing or mentioning one?”** This is deliberately multi-label.

| Key | Type | Values or criterion |
|---|---:|---|
| `role.main` | N | substantial centerpiece of a meal |
| `role.side` | N | accompaniment to another principal dish |
| `role.complete_meal` | N | ordinarily supplies the central dish and meaningful accompaniments without requiring another recipe |
| `role.soup_stew_chili` | N | spoonable soup, stew, chowder, curry-like stew, or chili |
| `role.salad` | N | salad is the prepared dish, not merely a garnish |
| `role.sandwich_wrap` | N | sandwich, burger, taco, wrap, or filled handheld |
| `role.pasta_noodle` | N | pasta or noodles are structurally central |
| `role.rice_grain_bowl` | N | rice, grain, or composed bowl is structurally central |
| `role.breakfast_brunch` | N | conventionally suitable as breakfast or brunch |
| `role.appetizer_snack` | N | conventionally served as appetizer, party bite, or snack |
| `role.dessert` | N | sweet concluding dish |
| `role.baked_good` | N | bread, cake, cookie, pastry, muffin, or similar baked item |
| `role.sauce_condiment_component` | N | primarily a sauce, dressing, condiment, dough, stock, filling, or reusable component |
| `role.beverage` | N | prepared drink |
| `role.potluck_dish` | N | naturally portionable and presentable for a shared spread; transport is assessed separately |
| `identity.primary_format` | C | `plated`, `one_vessel`, `bowl`, `handheld`, `shareable_platter`, `slice_or_piece`, `beverage`, `component`, `unclear` |
| `identity.temperature` | C | `hot`, `warm`, `room_temperature`, `cold`, `flexible`, `unclear` |

### 7.2 Culinary taxonomy

Cuisine labels describe culinary resemblance, not authorship or cultural authenticity. The first
question selects a broad family. Conditional child Choices refine it through a versioned hierarchy;
beam search may retain several plausible paths. `cross_cultural_or_fusion` and `not_distinctive`
prevent false precision.

| Key | Type | Values or criterion |
|---|---:|---|
| `cuisine.family` | C | `african`, `caribbean`, `central_asian`, `east_asian`, `eastern_european`, `latin_american`, `mediterranean`, `middle_eastern`, `north_american`, `south_asian`, `southeast_asian`, `western_european`, `cross_cultural_or_fusion`, `not_distinctive`, `unclear` |
| `cuisine.<family>.region` | C | direct children from the pinned cuisine taxonomy plus `mixed`, `not_distinctive`, `unclear` |
| `tradition.comfort_food` | N | strongly resembles familiar, hearty comfort food |
| `tradition.restaurant_style` | N | resembles a dish commonly sought as a restaurant-style meal |
| `tradition.holiday_celebration` | N | naturally associated with a holiday or celebratory table independent of one named holiday |
| `tradition.backyard_cookout` | N | naturally suits grilling, barbecue, or backyard cookout service |

A regional label is never promoted solely because a source keyword claims it. Low-separation paths
route to broader-family materialization or review.

### 7.3 Ingredient composition

These questions classify semantic prominence. Exact ingredient presence comes from approved
canonical mappings, not Jev.

| Key | Type | Values or criterion |
|---|---:|---|
| `composition.center` | C | `red_meat`, `poultry`, `seafood`, `eggs`, `dairy`, `legumes`, `vegetables`, `grain_or_pasta`, `mixed`, `other`, `unclear` |
| `composition.plant_forward` | N | vegetables, legumes, fruit, fungi, or whole grains are central rather than incidental |
| `composition.protein_forward` | N | a concentrated protein is the principal identity of the dish |
| `composition.produce_heavy` | N | produce materially defines the dish and occupies a large conceptual share |
| `composition.starch_forward` | N | bread, pasta, rice, potato, or another starch materially defines the dish |
| `composition.sauce_forward` | N | sauce, gravy, dressing, broth, or cooking liquid is central to the experience |
| `composition.cheese_forward` | N | cheese is a defining component rather than garnish or minor seasoning |
| `composition.uses_alcohol_materially` | N | alcohol contributes meaningful volume or defining flavor, not a negligible optional splash |
| `composition.fresh_herb_forward` | N | fresh herbs materially define the dish rather than acting only as optional garnish |
| `composition.pantry_heavy` | N | most defining ingredients are shelf-stable pantry or freezer items |

### 7.4 Cooking methods

For each method ask whether successfully completing the recipe materially uses that method. A mere
mention does not qualify.

| Key | Type | Method |
|---|---:|---|
| `method.raw_assembly` | N | assembly without meaningful cooking |
| `method.baking` | N | dry oven baking |
| `method.roasting` | N | oven roasting or high-heat browning |
| `method.broiling` | N | direct overhead heat |
| `method.stovetop_saute` | N | sautéing, pan cooking, or shallow frying |
| `method.searing_browning` | N | deliberate surface browning as a meaningful step |
| `method.deep_frying` | N | deep-fat frying |
| `method.boiling_simmering` | N | boiling, poaching, or simmering in liquid |
| `method.braising` | N | browning followed by covered or moist slow cooking |
| `method.slow_cooker` | N | slow cooker is a primary cooking path |
| `method.pressure_cooker` | N | pressure cooker is a primary cooking path |
| `method.grilling` | N | outdoor or indoor grill is a primary path |
| `method.smoking` | N | smoke cooking materially defines the method |
| `method.steaming` | N | steam is a primary cooking method |
| `method.microwave` | N | microwave is materially used, not merely for melting or reheating |
| `method.blending_pureeing` | N | blending or pureeing is required for intended texture |
| `method.kneading_shaping` | N | dough handling or repeated shaping is a meaningful task |
| `method.marination_brining` | N | advance marination or brining materially affects the result |
| `method.chilling_setting` | N | chilling or resting is needed to set, firm, or finish the dish |
| `method.reduction_emulsion` | N | reduction or emulsion requires active technique |
| `method.cookware_pattern` | C | `no_cook`, `one_pan_or_pot`, `sheet_pan`, `single_appliance_vessel`, `multiple_vessels`, `unclear` |

### 7.5 Equipment burden

Explicit mentions become extraction candidates. Jev determines whether equipment is genuinely
required for the intended result or merely convenient.

| Key | Type | Criterion |
|---|---:|---|
| `equipment.oven` | N | conventional oven is required |
| `equipment.stovetop` | N | stovetop or portable burner is required |
| `equipment.slow_cooker` | N | slow cooker is required for the represented method |
| `equipment.pressure_cooker` | N | pressure cooker is required |
| `equipment.outdoor_grill` | N | outdoor grill is required |
| `equipment.food_processor_blender` | N | powered processor, blender, or immersion blender is required |
| `equipment.stand_or_hand_mixer` | N | powered mixer is required for intended result |
| `equipment.specialty_bakeware` | N | specialty pan, mold, stone, or similarly uncommon bakeware is required |
| `equipment.thermometer_helpful` | N | thermometer materially improves reliable doneness or safety control |
| `equipment.specialty_tool_required` | N | a tool outside ordinary knife, board, bowls, pots, pans, sheet pan, and utensils is required |
| `equipment.burden` | S | `ordinary basics only`; `one common appliance`; `several common tools`; `one specialty tool`; `multiple specialty tools or setup` |

### 7.6 Flavor, texture, and sensory profile

All scores use five recipe-specific, textual levels. They describe the finished dish, not objective
chemical measurements.

| Key | Type | Ordered levels 0 → 4 |
|---|---:|---|
| `flavor.heat` | S | no chile heat; background warmth; noticeable mild heat; clearly spicy; intense chile heat |
| `flavor.sweetness` | S | not sweet; faint sweetness; balanced sweet note; distinctly sweet; dessert-level sweetness |
| `flavor.richness` | S | very light; light; moderate; rich; exceptionally rich/heavy |
| `flavor.acidity` | S | no evident acidity; faint; balancing; bright/tangy; sharply acidic |
| `flavor.savoriness` | S | mild; lightly savory; clearly savory; deeply savory; intensely umami/savory |
| `flavor.smokiness` | S | none; trace; supporting; prominent; defining |
| `flavor.herbaceousness` | S | none; light; noticeable; prominent; defining fresh-herb character |
| `flavor.spice_aromatic_intensity` | S | neutral; lightly seasoned; aromatic; strongly aromatic; assertive/complex spice profile |
| `texture.crispness` | S | none; minor accent; meaningful contrast; prominent; defining crisp/crunchy texture |
| `texture.creaminess` | S | none; slight; moderate; prominent; defining creamy texture |
| `texture.brothiness_sauciness` | S | dry; lightly coated; saucy; very saucy/brothy; liquid-forward |
| `texture.tender_softness` | S | firm/chewy; some tenderness; tender; very soft; fall-apart/pureed |
| `experience.familiarity` | S | highly unfamiliar to a typical US home diner; somewhat unfamiliar; mixed; familiar; highly familiar/classic |
| `experience.flavor_assertiveness` | S | very mild; mild; balanced; bold; highly assertive |

These are ranking signals, not user preferences. Household feedback may later learn that a person
likes or dislikes a given profile.

### 7.7 Effort and execution profile

Numeric time stays deterministic. These scores describe the nature of the work.

| Key | Type | Ordered levels 0 → 4 |
|---|---:|---|
| `effort.prep_complexity` | S | almost none; simple; several routine tasks; many or intricate tasks; elaborate multi-part prep |
| `effort.cooking_skill` | S | assembly/basic reheating; beginner; comfortable home cook; advanced technique; expert precision |
| `effort.active_attention` | S | set-and-forget; occasional checks; periodic attention; frequent attention; nearly continuous attention |
| `effort.timing_sensitivity` | S | highly forgiving; forgiving; some timing matters; timing-sensitive; narrow timing window across components |
| `effort.cleanup` | S | minimal; light; moderate; substantial; extensive multi-vessel cleanup |
| `effort.coordination` | S | one linear path; mostly linear; a few coordinated components; several parallel components; complex synchronized finish |
| `effort.ingredient_prep` | S | open/measure only; limited chopping; routine chopping; substantial prep; intricate cutting/shaping |
| `effort.failure_sensitivity` | S | difficult to spoil; very forgiving; ordinary care; easy to degrade; technically fragile |
| `effort.interruption_tolerance` | S | can pause freely; mostly pausable; some fixed windows; difficult to interrupt; continuous precise execution |
| `effort.overall` | S | trivial; easy; moderate; involved; project recipe |

`effort.overall` is retained as a user-facing summary but never replaces the atomic dimensions in
planning. Composite scores are computed in code with explicit weights.

### 7.8 Planning utility

These are intrinsic suitability judgments assuming the recipe is prepared as written. Household
schedule, available equipment, weather, and preferences are supplied only in dynamic planning.

| Key | Type | Criterion |
|---|---:|---|
| `utility.weeknight_friendly` | N | reasonably practical on an ordinary work night considering active work, attention, and coordination—not merely total time |
| `utility.busy_night_friendly` | N | practical when the cook has little uninterrupted attention |
| `utility.weekend_project` | N | effort or technique makes leisurely cooking part of the appeal |
| `utility.make_ahead` | N | substantial work can be completed earlier without meaningful quality loss |
| `utility.assemble_ahead` | N | dish can be assembled earlier and cooked or finished later |
| `utility.batch_cooking` | N | naturally supports making a large batch without delicate per-serving work |
| `utility.leftovers` | N | ordinarily retains good eating quality after refrigeration and reheating or cold service |
| `utility.freezer_friendly` | N | finished dish or a clearly defined prepared stage ordinarily freezes and recovers well |
| `utility.meal_prep` | N | portions and retains well for planned later meals |
| `utility.scales_up` | N | can serve more people without proportionally increasing delicate or serial work |
| `utility.scales_down` | N | can be reduced without awkward package, vessel, or indivisible-component problems |
| `utility.crowd_friendly` | N | service and broad practicality suit a group meal |
| `utility.entertaining` | N | presentation and execution suit hosting when the cook wants a notable meal |
| `utility.kid_approachable` | N | flavor, format, and texture are broadly approachable for children; this is never a guarantee for a specific child |
| `utility.portable` | N | transports without likely leakage, collapse, or immediate quality loss |
| `utility.holds_for_service` | N | can wait after completion without a narrow quality window |
| `utility.quick_serving` | N | once cooking begins, individual portions do not require slow serial assembly |
| `utility.lunch_suitable` | N | practical and natural as lunch, including packed or reheated lunch where applicable |
| `utility.warm_weather` | N | preparation and eating experience naturally suit warm weather |
| `utility.cold_weather` | N | preparation and eating experience naturally suit cold weather |
| `utility.comforting` | N | likely to be sought for warmth, familiarity, or emotional comfort |
| `utility.special_occasion` | N | has enough ceremony, presentation, or distinctiveness for a special meal |
| `utility.low_supervision` | N | most cooking proceeds safely with infrequent checks |
| `utility.flexible_serving_time` | N | serving can move by roughly ordinary household delay without significant degradation; no exact duration is implied |

### 7.9 Shopping and sourcing burden

Cost and availability vary by household and market, so these are broad semantic signals rather than
price claims.

| Key | Type | Ordered levels 0 → 4 |
|---|---:|---|
| `shopping.specialty_burden` | S | standard staples; mostly common; one less-common item; several specialty items; specialty sourcing defines recipe |
| `shopping.perishability` | S | mostly shelf-stable; low; moderate; several perishables; highly perishable/time-sensitive basket |
| `shopping.package_waste_risk` | S | little likely remainder; small; moderate; several partial packages; many specialty remainders likely |
| `shopping.relative_cost` | S | very economical; economical; moderate; expensive; premium ingredients dominate |
| `shopping.substitution_flexibility` | S | many easy swaps; several; some; few; defining ingredients are difficult to replace |
| `shopping.pantry_likelihood` | S | mostly ordinary pantry staples; many likely staples; mixed; mostly planned purchases; specialty fresh purchases dominate |

Exact pantry ownership is never inferred from these scores.

### 7.10 Dietary adaptation potential

Approved ingredient mappings deterministically report what the recipe contains. Jev only evaluates
how structurally difficult an adaptation would be. Each question assumes the named restriction and
asks about preserving the dish's recognizable identity.

| Key | Type | Ordered levels 0 → 4 |
|---|---:|---|
| `adapt.vegetarian` | S | already compatible; one trivial omission/swap; straightforward changes; major redesign; incompatible with core identity |
| `adapt.vegan` | S | already compatible; one trivial omission/swap; straightforward changes; major redesign; incompatible with core identity |
| `adapt.gluten_free` | S | already compatible by ingredients; one reliable swap; several straightforward swaps; technically difficult; incompatible with core identity |
| `adapt.dairy_free` | S | already compatible; one reliable swap; several straightforward swaps; major texture/flavor redesign; incompatible with core identity |
| `adapt.lower_heat` | S | no adaptation needed; omit garnish/condiment; simple reduction; several flavor-balancing changes; chile heat defines dish |
| `adapt.alcohol_free` | S | no alcohol; omit without impact; straightforward substitute; major flavor/method impact; alcohol defines dish |

These answers must never be presented as allergy or medical safety advice. “Already compatible” is
validated against approved canonical ingredients before materialization.

### 7.11 Search-oriented concepts

These facets cover recurring intents that lexical ingredient search does not answer well.

| Key | Type | Criterion |
|---|---:|---|
| `search.cozy` | N | warm, comforting, homey eating experience |
| `search.fresh_bright` | N | fresh, crisp, herbaceous, or acidic character dominates over richness |
| `search.light_meal` | N | likely experienced as a lighter meal rather than rich or heavy |
| `search.hearty` | N | substantial, filling, and robust |
| `search.indulgent` | N | richness, sweetness, fried character, or abundance makes indulgence part of appeal |
| `search.family_style` | N | naturally served communally rather than individually composed |
| `search.impressive` | N | likely to feel notably impressive relative to ordinary home cooking |
| `search.casual` | N | naturally fits an informal meal |
| `search.hands_off` | N | unattended cooking materially dominates active intervention |
| `search.craveable` | N | bold comfort, texture, richness, or familiarity gives strong treat-like appeal; retained only after evaluation because this is subjective |
| `search.customizable_at_table` | N | diners can meaningfully choose toppings, fillings, or components at service |
| `search.good_for_picky_eaters` | N | simple recognizable format with separable or mild components; household feedback must eventually override this generic signal |

## 8. Safety and evidence facets not delegated to Jev

The following are computed from approved ingredient ontology and explicit source evidence:

- contains milk, egg, fish, crustacean shellfish, tree nuts, peanuts, wheat, soy, sesame, or another
  configured allergen;
- contains meat, poultry, seafood, alcohol, caffeine, or added sugar;
- vegetarian/vegan/gluten-free status under an explicitly versioned policy;
- exact active, unattended, and total durations;
- explicit temperatures and food-safety instructions;
- nutrition amounts supplied by the source; and
- whether an ingredient mapping or alternative remains unresolved.

Unknown and possible cross-contact remain explicit. Grocery Router never infers “allergy safe.”

## 9. Future dynamic search questions (deferred)

This section reserves likely runtime questions and is not part of the import-time implementation.
Arbitrary natural-language search cannot be solved by a fixed tag catalog alone. A future search
phase may use a hybrid pipeline:

1. deterministic filters for exact constraints and approved facets;
2. lexical retrieval over name, description, ingredients, and instructions;
3. a bounded shortlist; and
4. Jev scoring against the user's query.

| Key | Type | Ordered levels 0 → 4 |
|---|---:|---|
| `dynamic.search_relevance` | S | unrelated; weak connection; plausible; strong match; direct excellent match |
| `dynamic.constraint_evidence` | C | `satisfies`, `violates`, `not_enough_evidence` |
| `dynamic.intent_role` | C | `find_specific_recipe`, `browse_by_ingredient`, `browse_by_occasion`, `browse_by_effort`, `browse_by_diet`, `browse_by_flavor`, `open_ended_inspiration`, `unclear` |

Jev cannot override a deterministic violation. For example, semantic relevance cannot admit a recipe
that exceeds an exact user time limit or contains an excluded approved ingredient.

## 10. Future dynamic planning questions (deferred)

This section reserves likely runtime questions and is not part of the import-time implementation.
Future planning would evaluate context without rewriting intrinsic facets. Code would first remove
hard-constraint violations and compute exact history, time, and ingredient-overlap facts. Jev would
then evaluate the shortlist.

| Key | Type | Ordered levels or values |
|---|---:|---|
| `dynamic.context_fit` | S | poor; weak; acceptable; strong; excellent fit for supplied occasion and constraints |
| `dynamic.household_appeal` | S | strong evidence against; somewhat unlikely; unknown/mixed; likely; strong positive evidence |
| `dynamic.week_contribution` | S | makes proposed week worse; little value; acceptable; improves balance; fills an important gap |
| `dynamic.recipe_pairing` | S | clashes; weak pairing; workable; complementary; especially coherent pairing |
| `dynamic.redundancy` | S | very distinct; somewhat distinct; moderate overlap; strongly repetitive; near duplicate experience |
| `dynamic.workload_compatibility` | S | impractical together; awkward; workable; compatible; naturally coordinated |
| `dynamic.special_request_fit` | C | `satisfies`, `partially_satisfies`, `does_not_satisfy`, `not_enough_evidence` |

Planner explanations are assembled from accepted atomic signals and deterministic facts. Jev is not
asked to generate a persuasive explanation.

## 11. Confidence and review routing

Thresholds are learned per question and primitive from labeled fixtures; this draft intentionally
does not invent universal numbers.

- High-confidence, fixture-validated intrinsic answers may be materialized automatically when they
  do not affect safety or recipe truth.
- Medium-confidence answers remain searchable only as provisional internal signals or route to
  review, according to the definition.
- Low-confidence answers are retained diagnostically but not materialized.
- Cuisine leaf labels require both sufficient confidence and separation from the second path;
  otherwise the broader accepted ancestor is used.
- Dietary, allergen, grocery mapping, and source-conflict decisions always follow their stronger
  deterministic/human approval rules.
- Model errors, timeouts, malformed responses, or missing answers leave the facet unknown. They do
  not block preserving the source artifact.

## 12. Evaluation set

Before production use, create a labeled set containing:

- all current approved household recipes;
- all successful pilot recipes;
- intentionally ambiguous multi-role dishes;
- culturally ambiguous and fusion dishes;
- recipes with optional meat, dairy, alcohol, and garnishes;
- recipes with long unattended but short active time;
- one-pot recipes with substantial prep;
- recipes that freeze poorly despite being batch-friendly;
- recipes whose source keywords overclaim category or diet;
- ingredient alternatives and dual-unit source lines; and
- adversarial source text that contains instructions directed at a model.

Evaluate each definition independently. Record agreement, confusion patterns, coverage, abstention,
and review rate. Search and planning questions additionally require ranked-query fixtures and
pair/week scenarios. Catalog changes that alter accepted materializations require a replay report.

## 13. Initial product capabilities enabled

Once approved and implemented, the import-time catalog supports:

- persisted facets for search by role, cuisine, method, flavor, texture, effort, equipment, and
  planning utility;
- deterministic compound filtering such as “cozy, hands-off, and reheats well”;
- transparent “why this matched” summaries made from approved individual facets;
- precomputed signals for later context-aware suggestions, week balancing, pairing, and preference
  learning; and
- targeted human review when model confidence is insufficient.

It does not yet authorize runtime Jev calls, free-form semantic reranking, a runtime planner, public
recipe onboarding, automatic recipe approval, nutrition estimation, pantry inference,
substitutions, or medical/dietary guarantees.

## 14. Decisions required before approval

1. Which structured extraction model or service proposes unbounded recipe fields before Jev.
2. Whether generic signals such as `kid_approachable`, `craveable`, and relative cost are valuable
   enough to retain before household feedback exists.
3. The pinned cuisine hierarchy and policy for multi-cuisine/fusion recipes.
4. Which assessment answers require human review versus automatic semantic materialization.
5. Retention and cost policy for raw Jev responses.
6. Whether semantic profiles are household-owned or may eventually back a separately governed
   reusable catalog.

Resolved for this phase: Jev is limited to operator-invoked import and review tooling. Runtime search
reranking and planning assessment require a later approved phase.

## 15. Pilot command and outputs

```sh
task recipe-intelligence-pilot
```

The command reads successful artifacts from the newest completed run beneath
`.local-run/recipe-source-pilot/`, the approved
Markdown corpus, and `pilot/recipe-intelligence.yaml`. It requires `TYPESAFE_API_KEY`, pins the
catalog's versioned model, and writes a new immutable run beneath
`.local-run/recipe-intelligence/`. Each run contains an index, one JSON artifact per assessed
candidate, and `review.md`. Artifacts preserve projected state, question definitions, complete
answers, probabilities, confidence, model identity, token usage, and input/catalog digests. They
never contain the API key.

Automated tests use an injected evaluator or local HTTP server and make no live TypeSafe requests.
A failed candidate is recorded, remaining candidates are attempted, and the command exits non-zero
when any candidate fails. Pilot output is local evidence only: it is not ingested into PostgreSQL and
cannot alter approved corpus truth.
