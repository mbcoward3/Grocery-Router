# v1 Scope Interview — Raw Transcript

> **Historical and non-authoritative.** This is the user-visible interview transcript,
> excluding hidden reasoning, tool calls, and tool output. Decisions were distilled into
> `V1_SPEC.md`, `PRODUCT_DECISIONS.md`, `TRUE_UP_PLAN.md`, and `UP_NEXT.md`. Those active
> documents govern when this transcript differs or contains ambiguity.

Session: `01a00d39-a2cc-733c-a8e3-081542d51454`  
Started at the v1 scope-reset interview.

---

## User

let's consolidate behind phase 1 then. Personally I think the current recipes as markdown and our conversion to get to that point has been sloppy. Today when trying to use the python web app I noticed that it was doing a terrible job of estimating quantities. I also found the attempts it was making to assume what I'd already have in stock to be pretty bad - such that I'd prefer to just remove that from v1 scope. For v1, Let's really scope this down to something basic as hell. I think we should aim to "databaseize" my family's corpus(this will likely require analysis on current state of things compared to linked recipes and source material pdf to ensure quantities are logical) so that it's extensible as we add future feature sets. I'm fine just building it for local sqlite db today while we build out v1. Basically, I want a UI that I can interact with that offers me a week of groceries based on my corpus. I don't even care to have the fancy suggestions of new recipes stuff for now. I want a bulletproof schema and UI for picking a week of recipes and we can build from there. Plase interview me relentlessly until we share a clear scope and goal for this phase.

## Assistant

Agreed. I’m treating this as a reset of v1, with previous architecture and feature decisions no longer binding unless reaffirmed.

## My provisional interpretation

A deliberately small local app that:

1. Stores a carefully verified family recipe corpus in SQLite.
2. Lets you choose recipes for the coming week.
3. Produces a deterministic grocery list from those recipes.
4. Makes **no assumptions about pantry inventory**.
5. Does no recipe discovery, acquisition, behavioral scoring, or elaborate AI planning.
6. Prioritizes trustworthy quantities, provenance, and correction workflows over automation.

I’ll challenge this interpretation through several interview rounds. Please answer by number; short answers are fine.

# Round 1 — What exactly is the product?

### 1. Describe the ideal weekly interaction

Which is closest?

**A. Manual selection:** Browse/search the corpus and explicitly choose every recipe.

**B. Suggested starting week:** The app selects a basic starting set from the corpus, then you keep, remove, or replace recipes.

**C. Constrained generation:** You specify nights, effort, guests, etc., and it generates a week.

My recommendation for basic-as-hell v1 is **B**, using transparent deterministic rotation rather than AI.

### 2. If the app offers a starting week, how should it choose?

Possible inputs:

- random selection
- least recently cooked
- avoid repeating last week
- protein/cuisine variety
- favorites
- explicit household rules
- model-generated reasoning

Which matter in v1? Is “shuffle five eligible recipes” sufficient?

### 3. What is a “week”?

Is it:

- a pool of, for example, five recipes with no assigned days
- recipes assigned to specific days
- a configurable number of meals
- always the same number of meals

Do leftovers count as planned meals?

### 4. What must the final grocery list include?

You said no inventory assumptions. Should it therefore include:

- every ingredient named by every selected recipe
- salt, pepper, water, cooking oil, and other common staples
- optional toppings and garnishes
- ingredients marked “to taste”
- ingredients with no stated quantity

My conservative default is: **include everything the recipe says, visibly flag anything unresolved, and let you manually uncheck/remove it.**

### 5. What does “quantity accuracy” mean to you?

These are different products:

**A. Recipe requirements:** “These meals require 5.3 lb of chicken total.”

**B. Purchase guidance:** “Buy 6 lb of chicken.”

**C. Product/package guidance:** “Buy three 2 lb packages.”

Which should v1 produce?

My recommendation is to guarantee **A** first. Only produce B or C where an explicit, reviewed purchase rule exists.

### 6. Should quantities be scaled?

For each selected recipe, should the UI ask:

- how many people are eating
- how many portions you want
- whether you want leftovers
- nothing—use the recipe’s original batch

If the source says “serves 6,” can we scale that mathematically, or must your household verify what “serves 6” means in practice?

### 7. What should happen when a quantity cannot be trusted?

Examples:

- “one onion,” with no size
- “chicken, as needed”
- “2 cups cooked chicken,” but you must buy raw chicken
- recipe yield is absent
- source says “one package”
- two sources disagree

Choose the posture:

**A. Block the grocery list until corrected.**

**B. Include the original text under “Needs attention.”**

**C. Make a clearly labeled estimate.**

My recommendation is **B**, with no estimates unless you explicitly approve a conversion rule.

### 8. What is authoritative for each recipe?

Potential sources currently include:

- linked web recipe
- source PDF
- typed/transcribed input
- current recipe Markdown
- your household’s corrected version

If they disagree, what wins?

My proposed hierarchy is:

1. Explicit household correction
2. Original source material
3. Existing Markdown only when traceable to that source
4. Otherwise unresolved

Do you agree?

### 9. Are household adaptations first-class data?

Examples:

- use less broth than the linked recipe
- always add cheese to hamburgers
- substitute chicken thighs
- make a double batch
- omit an ingredient

Should the database preserve both:

- the source recipe, unchanged
- your household’s adopted version

Or do you only care about the current household version?

### 10. How rigorous should corpus verification be before UI work?

Options:

**A. Verify all existing recipes against links/PDF before building the app.**

**B. Build the schema and correction UI first, then verify recipe-by-recipe through it.**

**C. Verify a representative subset, build the app, then migrate the remainder.**

I recommend **C**, with several difficult recipes chosen as schema tests.

### 11. What recipe information must v1 store and expose?

Mark each as required, optional, or out:

- title
- source URL/document/page
- original ingredient text
- normalized ingredient name
- exact quantity and unit
- preparation note (“diced,” “cooked,” etc.)
- recipe yield
- instructions
- active time
- total time
- category/cuisine
- tags
- family notes
- last cooked
- favorite/disliked
- variants
- sides

### 12. Is the corpus only dinner mains?

Should v1 support:

- sides
- breakfasts
- lunches
- desserts
- sauces
- non-recipe grocery items

Or should the schema be capable of those later while the v1 UI only handles dinner recipes?

### 13. How should ingredients aggregate?

If one recipe needs `1 onion` and another needs `8 oz onion`, should v1:

- convert and combine them
- keep them as separate lines unless a reviewed conversion exists
- force corpus cleanup before allowing such mixed units

My recommendation: combine only through explicit item-specific conversions; otherwise show both requirements.

### 14. What list editing is allowed?

Can you:

- add arbitrary groceries
- change a generated quantity
- remove an ingredient
- mark an ingredient already owned
- check items off while shopping

Should edits change only this week’s list, or be promotable into permanent recipe corrections?

### 15. Where will you use it?

Please confirm:

- planning device: desktop, phone, or both?
- shopping device: iPhone Safari?
- must it work away from home?
- must it work without network access?
- is running a local server on your laptop acceptable during v1?
- one user only, or should multiple family members use it concurrently?

### 16. Does v1 contain any AI at all?

Potential uses:

- choosing the initial week
- parsing source recipes during import
- normalizing ingredients
- no AI in runtime; perhaps AI-assisted migration reviewed by a person

My recommendation: **no AI in the trusted runtime path**. AI may assist corpus migration, but every imported fact must be reviewable before acceptance.

### 17. What is explicitly out of scope?

Please approve or amend:

- recipe discovery
- new-recipe suggestions
- pantry/inventory inference
- store integration
- pricing
- package/SKU matching
- automatic purchase rounding
- authentication
- multi-household support
- cloud deployment
- Kubernetes/Talos
- behavioral scoring
- self-improvement metrics
- profile claims
- meal feedback
- model-written reasons
- sophisticated leftovers planning

### 18. What proves v1 is done?

My proposed acceptance test:

> Starting from a verified SQLite corpus, Matt can select five known recipes, adjust their batch sizes, inspect every unresolved quantity, generate a traceable grocery list containing every recipe requirement with no pantry subtraction, make weekly-only edits, and use/check the list on an iPhone.

What is missing or unnecessary in that sentence?

## User

can we do questions one at a time?

## Assistant

Absolutely.

**Question 1:** How should you pick the week?

**A.** Manually choose every recipe.  
**B.** Start with a suggested set, then keep, remove, or replace recipes.  
**C.** Provide constraints and have the app generate the week.

Which is closest?

## User

c

## Assistant

**Question 2:** What information should you provide when generating a week?

For example: number of dinners, guests, busy nights, desired leftovers, effort level, proteins, or specific must-include recipes. What matters to you?

## User

number of dinners and number of guests is probably fine for now. I'm even hesitant about number of guests because it leans on us being good at quantity calculations.

## Assistant

Agreed. For v1, we’ll use **number of dinners only** and defer guest scaling until quantity calculations are trustworthy.

**Question 3:** How should the app generate those dinner choices?

**A.** Randomly from eligible recipes  
**B.** Prefer recipes not cooked recently  
**C.** Use an AI model  
**D.** Something else

## User

a, random for now. will add agent soon.

## Assistant

Good: **random selection now, agent-based planning later**.

**Question 4:** After generating the random set, should you be able to manually replace any selected recipe with a specific recipe from the corpus?

## User

yes, you should be able to add, remove, swap with random, swap with specific any of the recipes.

## Assistant

Understood. The week editor needs four direct actions: **add, remove, swap randomly, and swap with a specific recipe**.

**Question 5:** Should selected recipes be assigned to particular days, or remain an unordered pool for the week?

## User

nah, not for now. Our weeks are pretty dynamic.

## Assistant

So the week is an **unordered pool**.

**Question 6:** Should “number of dinners” mean the number of recipes selected, regardless of whether leftovers cover additional nights?

## User

let's ignore the leftover bit for now.

## Assistant

Got it: **one selected recipe equals one dinner slot; leftovers are out of scope**.

**Question 7:** Should the grocery list include every ingredient named by the selected recipes, including common staples like salt, pepper, oil, and spices?

## User

for now include everything.

## Assistant

Understood: **include everything, with no pantry or staple assumptions**.

**Question 8:** Should v1 report the exact recipe requirements, such as `5.3 lb chicken`, rather than trying to round them into suggested purchase quantities like `buy 6 lb`?

## User

ya don't try to be fancy, pass the recipe values along, we need to make sure the recipe is correct.

## Assistant

Agreed: preserve and aggregate verified recipe quantities without purchase estimates or rounding.

**Question 9:** For v1, should each selected recipe use its original batch quantities exactly, with no scaling for household size or guests?

## User

I don't have a good answer for this. If we don't let folks select # guests, we should just stick wiht baseline for now.

## Assistant

That makes sense. We’ll defer scaling and use a stored baseline batch.

**Question 10:** Should that baseline be the recipe source’s original quantities, or your family’s usual adjusted version when it differs from the source?

## User

yes?

## Assistant

Let me make that more concrete:

If a source recipe serves 8, but your family normally cooks half of it, which quantities should v1 store as the default?

**A.** The source’s full recipe  
**B.** Your family’s usual half-batch  
**C.** Preserve both, with your family’s version as the default

## User

how will we know my family's usual quantity? Just use what's in the pdf

## Assistant

Right—we should not invent a household adjustment. The **PDF quantities are the baseline** unless you explicitly correct them later.

**Question 11:** When the PDF includes a link to an online recipe, should the PDF’s written quantities still win if the website currently differs?

## User

website wins

## Assistant

Understood: **the linked website is authoritative; the PDF is the fallback/source index**.

**Question 12:** If a linked webpage is unavailable or has changed so much that the PDF entry cannot be matched confidently, should that recipe be marked unresolved and excluded from week generation until reviewed?

## User

this is a one time true up so we should just come to consensus if there is something missing together.

## Assistant

Got it. We’ll perform a **one-time, recipe-by-recipe true-up**, bringing ambiguities to you for a decision rather than automatically excluding or guessing.

**Question 13:** For each ingredient, should the database preserve both:

1. the exact source text, such as `2 cups cooked chicken, shredded`, and  
2. structured fields like quantity `2`, unit `cup`, item `chicken`, preparation `cooked, shredded`?

I recommend keeping both for traceability.

## User

err, both I ugess

## Assistant

We’ll keep both: the original line for verification and structured fields for list generation.

**Question 14:** If you correct a source ingredient later, should we retain the original value and record your correction separately, or simply replace it?

## User

I'd keep a source of truth

## Assistant

By “source of truth,” do you mean the verified database value should simply become authoritative, without maintaining a full correction/version history?

## User

yes

## Assistant

Understood: SQLite holds the current verified truth; no correction history in v1.

**Question 15:** Does v1 need to store cooking instructions, or only the recipe name, source link, and verified ingredients needed for the grocery list?

## User

um, would kinda be nice once we have a good UI to be able to click into an acctual recipe wiht step by step.

## Assistant

Then recipes should include instructions and support a recipe-detail view.

**Question 16:** Should the app display the cooking steps directly from SQLite, or is opening the original website acceptable for v1?

## User

direct from sqlite as a nice in app page.

## Assistant

Got it: v1 stores and displays ordered cooking steps directly from SQLite.

**Question 17:** Should recipes support ingredient sections such as **“Sauce,” “Filling,” and “Topping,”** or is one flat ingredient list sufficient?

## User

sections are important

## Assistant

Agreed: ingredient sections are first-class and retain their source ordering.

**Question 18:** If the source marks an ingredient as optional, should it still appear on the grocery list by default, clearly labeled optional?

## User

sure?

## Assistant

We’ll include optional ingredients by default and label them clearly.

**Question 19:** When multiple selected recipes use the same ingredient in compatible units—for example, `1 lb beef` and `2 lb beef`—should the grocery list combine them into `3 lb beef`?

## User

for 18, optional ingredients also need to be optional in the shopping list. for 19 - definitely.

## Assistant

Understood on combining compatible quantities.

**Question 20:** For an optional ingredient, should it initially be excluded from the shopping total until you explicitly include it, or included but visually marked as optional?

## User

include but visibly optional

## Assistant

Got it: optional ingredients are included by default and visibly labeled.

**Question 21:** If the same ingredient appears in incompatible units—such as `1 onion` and `1 cup onion`—should the list keep them as separate lines rather than estimating a conversion?

## User

um, that is tricky. I'd like to consolidate

## Assistant

Consolidation requires a conversion assumption. For example, we must decide how many cups of chopped onion equal one onion.

**Question 22:** Should the true-up establish explicit, reviewed conversions for individual ingredients—such as `1 medium onion = 1 cup chopped`—and only consolidate when such a verified rule exists? Otherwise, the list would flag the conflict for your review.

## User

I don't wanna overcomplicate this massively. just display em separately for now.

## User

I don't wanna overcomplicate this massively. just display em separately for now unless you have some way to guarantee conversions can happen accurately over time and not be annoying

## Assistant

Agreed. We’ll keep incompatible units separate and leave room for reviewed item-specific conversions later.

**Question 23:** Should universally exact unit conversions still combine automatically—for example, `1 cup broth + 8 fluid ounces broth`, or `1 tablespoon salt + 3 teaspoons salt`?

## User

yes

## Assistant

Good: exact conversions combine; ingredient-specific estimates do not.

**Question 24:** How should the grocery list be organized in v1: alphabetically, by recipe, or by manually assigned store sections such as Produce, Meat, and Dairy?

## User

Store sections. Each ingredient should say what recipe(s) it links to.

## Assistant

Understood: group by store section, with every line showing its contributing recipes.

**Question 25:** During the corpus true-up, should we manually assign each canonical ingredient to a store section, rather than inferring sections automatically?

## User

ya I guess so

## Assistant

We’ll manually verify store sections as corpus data.

**Question 26:** Should differently worded source ingredients map to one canonical grocery item when clearly equivalent—for example, `yellow onion, diced` and `diced yellow onion` both mapping to `yellow onion`—while preserving each original source line?

## User

yes. Are you thinking that there is some lift here that needs LLM involvement here at recipe selection time or do we at corups ingest/true up time assign the actual grocery equivalent for each item?

## Assistant

The grocery equivalent should be assigned at **corpus ingest/true-up time**, not recipe-selection time.

The LLM may help propose mappings during migration, but you approve them once. Runtime behavior then stays deterministic:

- Source line: `2 yellow onions, diced`
- Canonical grocery item: `yellow onion`
- Quantity: `2`
- Unit: `each`
- Preparation: `diced`
- Store section: `Produce`

Recipe selection only reads these verified mappings and aggregates them. No LLM is needed.

**Question 27:** If a recipe somehow contains an unmapped ingredient, should grocery-list generation stop and require correction rather than guess?

## User

all selectable recipes must have been ingested correctly so this is impossible.

## Assistant

Excellent. We’ll enforce a hard boundary: **only fully verified recipes are selectable**. Draft/incomplete recipes cannot enter a week.

**Question 28:** Must every selectable recipe have all of the following?

- authoritative source
- ordered ingredient sections
- structured quantity/unit for every ingredient
- canonical grocery-item mapping
- store section for every grocery item
- ordered cooking steps

Should any of these be optional?

## User

no, as you can see in the raw recipes pdf this isn't preseent for a lot of my stuff. basically we need ot be able to onboard just a list of ingredients as an entry and backfill everything else. For example my burger recipe is jus a grocery list - but to standardize we need to be able to llm augmented way to backfill it.

## Assistant

Understood. “Correctly ingested” does not mean the source was complete. It means:

1. Import whatever evidence exists.
2. Use an LLM to propose missing structure, mappings, and possibly instructions.
3. Review/correct the proposal.
4. Promote it to a selectable, verified recipe.

The database then becomes the authoritative version.

**Question 29:** Must you explicitly approve all LLM-backfilled information before a recipe becomes selectable?

## User

for 28 - ingested correctly means the recipe is complete, not that the source was ever complete. for 29 - probably.

## Assistant

Exactly: a selectable recipe must be complete, even if completing it required reviewed backfilling. We’ll require approval before promotion.

**Question 30:** May the LLM propose missing ingredient quantities, or must any missing quantity be supplied or explicitly confirmed by you?

## User

yes

## Assistant

The LLM may propose missing quantities, but they remain drafts until you approve them.

**Question 31:** May it also propose entirely missing ingredients—for example, cheese for hamburgers—or only quantities and structure for ingredients already present in the source?

## User

yes

## Assistant

Understood. The LLM may propose missing ingredients and instructions, but nothing becomes part of a selectable recipe until you approve it.

**Question 32:** Should the generated grocery list allow you to add unrelated household items—such as paper towels or milk—even when no selected recipe requires them?

## User

not yet.

## Assistant

Out of scope for now: the list contains recipe-derived groceries only.

**Question 33:** Should you be able to manually remove a generated ingredient from a particular week’s grocery list because you already have it?

## User

yknow, I guess the grocery list can just be an add, remove, done tyep of list.

## Assistant

That suggests a simple editable checklist: generated lines can be added, removed, and marked done.

**Question 34:** Does “add” include arbitrary groceries like paper towels after all, or only restoring recipe-derived lines that were removed?

## User

sure

## Assistant

Okay: the weekly grocery list supports arbitrary additions, removal, and completion.

**Question 35:** Should those edits affect only that week’s list, with recipe data unchanged unless you deliberately edit the recipe itself?

## User

for now yes

## Assistant

Good: list edits are week-specific.

**Question 36:** If you change the selected recipes after editing the grocery list, should the app rebuild the recipe-derived lines while preserving your manual additions and completed-item state where possible?

## User

probably edits should cascade in such a way that this works.

## Assistant

We’ll model each generated line’s recipe contributions so adding, removing, or swapping a recipe updates only its contribution while preserving unrelated edits.

**Question 37:** Should you be able to manually change a generated quantity for that week, or only remove the line and add your own replacement?

## User

edit is fine, don't let the edit affect the underlying recipe

## Assistant

Understood: weekly quantity edits are overrides and never modify the recipe.

**Question 38:** Must v1 include a UI for adding and editing recipes, or can the initial corpus be loaded through a separate migration/true-up tool?

## User

no just use the badly constructed corpus that exists in the repo for this phase.

## Assistant

So recipe onboarding/editing UI is out of scope. We’ll true up the existing corpus during development and load the approved result into SQLite.

**Question 39:** For reviewing LLM-backfilled recipe data, are you comfortable approving it through our development conversation and generated audit reports rather than through the application UI?

## User

to be clear on 38 - many of our true up tasks will be similar to real onbarding tasks once we add that so it needs to be well done. for 39 - that's fine but again this should be exensible for future stuff.

## Assistant

Understood. The true-up pipeline should use the same validation and promotion services that a future onboarding UI will call; only the UI is deferred.

**Question 40:** Which existing recipes belong in the initial selectable corpus: every recipe represented in the PDF/repository, or only recipes your family has already cooked and wants to keep?

## User

ya every recipe in there.

## Assistant

All existing recipes will be true-upped and selectable.

**Question 41:** Should v1 treat every recipe equally, or distinguish mains from sides and other recipe types?

## User

all equal for now.

## Assistant

No recipe roles in v1; everything is simply a recipe.

**Question 42:** Can the same recipe appear more than once in a generated week, or must every selected recipe be unique?

## User

all unique suggested - user can swap in a dupe if hte wanna.

## Assistant

Random generation will choose unique recipes, while manual actions may intentionally add duplicates.

**Question 43:** If you add the same recipe twice, should its ingredient quantities count twice in the grocery list?

## User

ya

## Assistant

Correct—duplicate selections represent two batches.

**Question 44:** On the week-selection screen, what information must each recipe show besides its name? For example: ingredient summary, cooking time, source, or nothing else for v1.

## User

cook time is fine.

## Assistant

**Question 45:** Should “cook time” be one total duration, or should v1 distinguish hands-on preparation time from unattended cooking time?

## User

um, the two separately would be useful

## Assistant

We’ll store and show **hands-on time** and **unattended time** separately.

**Question 46:** Does the recipe picker need search in v1, or is browsing the entire corpus sufficient?

## User

no search for now.

## Assistant

No search or filtering for v1.

**Question 47:** Besides swapping individual recipes randomly, should there be a single action to discard and randomly regenerate the entire week?

## User

eh, probably not for now. regen is free with this not involving an llm yet.

## Assistant

Just to clarify: because regeneration is free, do you want a **“Regenerate week”** button, or should users only swap recipes individually?

## User

both

## Assistant

We’ll support both full regeneration and individual random swaps.

**Question 48:** When manually adding a grocery item, should you choose its store section, or should it go into a generic “Other” section automatically?

## User

other

## Assistant

Manual items will go into **Other**.

**Question 49:** When an item is marked done, should it remain visible and crossed out, or move into a collapsed completed section?

## User

visible and crossed out, maybe get slightly smaller?

## Assistant

Done items remain visible, crossed out, and visually de-emphasized.

**Question 50:** For v1, is it acceptable to run the app locally on your computer and access it from your iPhone over the same network, with no offline support?

## User

ya that 's fine.

## Assistant

Local server plus iPhone browser is sufficient; offline support is out of scope.

**Question 51:** Should SQLite retain past weeks and their final grocery lists, or only the current week?

## User

ya, just retain it.

## Assistant

We’ll retain every week and its final state.

**Question 52:** Should each week be identified by a calendar start date, and if so, which day starts your week?

## User

Sunday

## Assistant

Weeks run Sunday through Saturday.

**Question 53:** Should v1 allow planning future weeks, or only the current Sunday-to-Saturday week?

## User

nah

## Assistant

Only the current week can be planned.

**Question 54:** Should past weeks be viewable in the UI, or merely retained in SQLite for future features?

## User

not v1

## Assistant

Past weeks will be retained but not exposed in the v1 UI.

**Question 55:** On the recipe detail page, should the original source link be visible alongside the in-app ingredients and instructions?

## User

sure

## Assistant

The recipe page will include source attribution and a link.

**Question 56:** Should planning and shopping be separate screens, with recipe details opening from either one?

## User

ya separate

## Assistant

Planning and shopping will be separate screens.

**Question 57:** Can a verified ingredient intentionally have a nonnumeric quantity such as `to taste`, `as needed`, or `for serving`, as long as that wording is explicit and approved?

## User

that's fine

## Assistant

Those are valid verified quantities, not incomplete data.

**Question 58:** If two recipes use the same item with nonnumeric quantities—such as both saying `salt to taste`—should they combine into one `salt — to taste` line listing both recipes?

## User

they should combing to just one salt at a shoppable quantity.

## Assistant

That introduces a small distinction between recipe quantity and shopping quantity.

For `salt to taste`, what should the combined list say?

**A.** `Salt — quantity not specified`  
**B.** `Salt — 1 container`  
**C.** Just `Salt`, with no displayed quantity

Which feels right?

## User

unspecified should land as c

## Assistant

Good: unspecified recipe quantities become one consolidated item name with no shopping quantity.

**Question 59:** If one recipe needs `1 tsp salt` and another says `salt to taste`, should the combined line show `1 tsp salt` and list both recipes, without adding an estimate for the second?

## User

well salt is a bad example because it is such a rare buy, I'd want it to just say salt here too on my grocery list

## Assistant

Understood. This is not a pantry assumption; it is a display rule for items whose recipe measurements are not useful purchase quantities.

**Question 60:** Should each canonical grocery item have a reviewed shopping mode such as:

- **measured** — aggregate and show quantity (`3 lb ground beef`)
- **counted** — aggregate and show count (`4 onions`)
- **presence-only** — show the item once with no quantity (`salt`)

That would let salt remain `Salt` without guessing whether you own it.

## User

that sounds complex but it also sounds correct

## Assistant

We’ll keep it explicit and small: every item gets one reviewed mode during true-up, with no runtime guessing.

**Question 61:** For packaged ingredients, should package size remain part of the grocery requirement—for example, `2 × 14.5 oz cans diced tomatoes`—rather than displaying only `2 cans`?

## User

ya that's important for shopping

## Assistant

Package count and package size will both be preserved.

**Question 62:** If recipes request different package sizes of the same item—such as one `28 oz can` and two `14.5 oz cans`—should they remain separate rather than being converted into a total weight?

## User

ya separate em

## Assistant

Different package sizes remain separate.

**Question 63:** Should ingredients that are purchased differently remain distinct canonical grocery items—for example, `raw chicken breast`, `cooked chicken`, and `rotisserie chicken`—even if they are related foods?

## User

ya

## Assistant

Purchased form is part of canonical item identity.

**Question 64:** If a source says only `chicken` without identifying a purchasable form, must we resolve that with you during true-up before the recipe is selectable?

## User

ya, in future state some things can sub in for other things especially in this fhicken case - but for now just resolve it

## Assistant

We’ll resolve one concrete purchasable form per ingredient now and defer substitution relationships.

**Question 65:** Are you comfortable treating the current web UI as disposable and designing the v1 interface from scratch, while reusing only validated parsing or aggregation logic where it fits?

## User

everything in here other than raw corupos is completely disposable and frankly I'd like you to get rid of as much as possible to start wiht good context from this conversation.

## Assistant

Understood. We’ll plan a clean rebuild and preserve only raw source evidence—not existing code, architecture, specs, or derived recipe data unless revalidated.

**Question 66:** Which repository content counts as raw corpus to preserve: the original PDF, linked source URLs, typed/transcribed source inputs, and existing household week/history records—or only the PDF and its links?

## User

the original PDF is gospel, some of the other stuff may be helpful though, up to you if you wnana re derive some of that

## Assistant

I’ll preserve the PDF as the canonical inventory of family recipes and retain other source material only as migration evidence.

One apparent conflict to settle:

**Question 67:** Is this correct?

- The PDF decides **which recipes belong in the corpus**.
- When the PDF links to a website, the website decides **the recipe’s ingredients, quantities, and instructions**.
- Unlinked or incomplete PDF recipes are completed with reviewed LLM proposals.

## User

yep, and for the ones that don't link to websites, we need to backfill - maybe by finding a similar but different recipe online and linking to that one but adjusting as needed?

## Assistant

Yes. For unlinked entries, we can:

1. Search for a credible matching recipe.
2. Present the closest candidate to you.
3. Import its ingredients and instructions as a starting point.
4. Adjust it to match what your family actually means.
5. Approve the completed recipe before it becomes selectable.

We should not imply the final recipe exactly matches that webpage.

**Question 68:** Should the database distinguish `source` from `adapted from`, so these backfilled recipes can credit the reference without claiming to reproduce it exactly?

## User

sure

## Assistant

We’ll track whether a recipe is reproduced from a source or adapted from a reference.

**Question 69:** Do you have a preferred implementation stack for the rebuild, or should I choose a simple SQLite-backed stack based on maintainability and UI quality?

## User

I really want to write this in go for the backend and whatever frontend let's us model after "linear" as our frontend inspiration. I plan to link to to open ai models with google's adk for go.

## Assistant

Direction recorded:

- **Go backend**
- **SQLite**
- likely **React + TypeScript** for a Linear-inspired interface
- future OpenAI-backed agents through **Google ADK for Go**

**Question 70:** Should this v1 phase actually integrate Google ADK for corpus true-up/backfilling, or only design a clean service boundary so agent integration can be added immediately afterward?

## User

no, just do all llming in this agent but as we work through those steps we should build out context or "skills" for future agents via the sdks.

## Assistant

Understood. v1 has **no runtime LLM or ADK dependency**. We’ll use this agent during true-up while preserving reusable prompts, rules, fixtures, and validation context for future SDK agents.

**Question 71:** Should the shipped v1 application contain no AI-generated week planning at all—only random selection—even if future agent seams are documented?

## User

no ai generation beyond our ingest steps.

## Assistant

Confirmed: AI is limited to development-time corpus ingestion and backfilling.

**Question 72:** Should v1 have no authentication and trust anyone who can reach the local server?

## User

for now yes

## Assistant

No authentication for local v1.

**Question 73:** What specifically do you want to borrow from Linear: visual polish, compact information density, keyboard-first interaction, command menus, fast transitions, dark mode, or something else?

## User

visual polish, compacct info density, nice transitions, dark mode, looks good but also means business.

## Assistant

The direction is polished, compact, dark, fast, and professional—not decorative or copy-heavy.

**Question 74:** Should the planning interface primarily target desktop, while the grocery checklist is designed mobile-first for iPhone?

## User

in the end they both need to work great on iphone.

## Assistant

Both planning and shopping must be fully usable on iPhone; neither is a desktop-only surface.

**Question 75:** Should SQLite become the sole runtime source of truth for recipes, with no generated Markdown recipe files kept in sync?

## User

correct. I think we should have skills and queries eventualy that can build agent context out of the db just the same.

## Assistant

SQLite will be the sole recipe source of truth. Future agents can receive controlled database queries or exported context instead of reading Markdown.

**Question 76:** Should the populated SQLite database itself be committed to Git, or should Git contain migrations plus verified seed data that deterministically builds the local database? My recommendation is migrations plus seed data, avoiding opaque binary diffs.

## User

nah, schema should

## Assistant

We’ll commit the schema and migrations, not the populated database.

**Question 77:** After the one-time true-up, how should the verified corpus be backed up or moved to another machine: a SQLite backup file, a JSON export/import command, or both?

## User

goose for migrations

## Assistant

We’ll use **Goose** for versioned SQLite schema migrations.

**Question 78:** Is database backup/export out of scope for v1, provided the local SQLite file can be copied manually?

## User

sqlc for slq.

## Assistant

Recorded: **Goose for migrations and sqlc for typed SQL access**.

**Question 78:** Is manual copying of the SQLite file sufficient for v1 backup, with formal export/import deferred?

## User

for 77 - dont worry about it for now, for 78 - ya manual ccopy is fine.

## Assistant

Backup and portability tooling are deferred; manual SQLite file copies are sufficient.

**Question 79:** Are you comfortable standardizing the frontend on React and TypeScript, or do you have another preference?

## User

back to 78 yknow maybe we commit something that is injectable into sqlite to the git repo as a snapshot for now. for 79 - whatever you think is most agent friendly and can get us closest to my desired look.

## Assistant

Revised direction:

- Commit Goose migrations.
- Commit a deterministic SQL corpus snapshot that can initialize SQLite.
- SQLite remains the runtime source of truth; the snapshot is only bootstrap/backup material.
- Use **React + TypeScript**, likely Vite, because it is agent-friendly and well suited to the polished responsive UI you want.

**Question 80:** Should updating a verified recipe deliberately regenerate and commit the SQL corpus snapshot, so corpus changes remain reviewable in Git?

## User

nah, justsomething we manually keep up to date.

## Assistant

Understood. The SQL snapshot is a manually refreshed checkpoint, not an automatically synchronized mirror.

**Question 81:** Should recipe images be included in v1, or should the interface rely on typography and structured recipe information only?

## User

um, ya probably try to include but no biggie if it's too hard

## Assistant

Recipe images are optional polish, not a v1 completion requirement.

**Question 82:** On the recipe detail page, do cooking steps need interactive checkboxes/timers, or is a clear read-only ordered list sufficient?

## User

nah read only is fine for now.

## Assistant

Recipe details will be read-only.

**Question 83:** Should store sections have a fixed shopping order—such as Produce, Meat, Dairy, Pantry, Frozen—or is alphabetical section ordering sufficient?

## User

should be reorderable but alpha is fine for start

## Assistant

Sections default alphabetically.

**Question 84:** Is section reordering required in v1, or should the schema support a future custom order while the v1 UI remains alphabetical?

## User

eh keep v1 simple

## Assistant

Custom section ordering is deferred.

**Question 85:** Should each grocery line show contributing recipe names directly beneath it, or hide them behind an expand action to keep the list compact?

## User

behind an expand

## Assistant

Recipe provenance will be available through expansion.

**Question 86:** Suppose recipes generate `3 lb chicken`, you override it to `4 lb`, then add another recipe requiring `1 lb`. Should the list become `5 lb` by preserving your `+1 lb` adjustment, or remain manually fixed at `4 lb`?

## User

hmm, this should trigger a warning to reconcile. for v1 just keep this becomes 5lb

## Assistant

We’ll store overrides as an adjustment to the generated total. In that example, `+1 lb` remains, producing `5 lb`. Reconciliation warnings can come later.

**Question 87:** If you remove a generated grocery line and then add another recipe that also requires that item, should the item reappear with the newly recalculated total?

## User

yaa

## Assistant

A recipe change reactivates a previously removed item when it is required again.

**Question 88:** If a checked-off item’s quantity changes because you modify the week, should it automatically become unchecked? respondent?

## User

bacck to 86 and 87 I think basilaly we should assume users don't edit the shopping list at all until they have locked in the recipes so most of this is moot.

## Assistant

That is a much cleaner rule. We can make the workflow:

1. Build and adjust the recipe pool.
2. **Lock the week.**
3. Generate the grocery list.
4. Edit and shop from the list.

**Question 88:** After locking, should changing recipes require an explicit **“Unlock week”** action that warns the grocery list will be regenerated?

## User

no I don't wna a "lcok unlock" paradigm I'm just saying that's how the workflow should assumed to work for v1

## Assistant

Understood—no lock state or extra ceremony. We’ll optimize for that natural sequence while keeping recipe changes deterministic.

**Question 88:** If a completed grocery item changes quantity after a recipe edit, should it become unchecked so the change is visible?

## User

too in the weeds on this non issue

## Assistant

Agreed. I’ll stop drilling into unlikely list-edit collisions and use unsurprising defaults.

**Question 89:** After the one-time corpus true-up, should recipes remain fixed in SQLite until manually changed, with no automatic synchronization against their source websites?

## User

ep

## Assistant

Recipes will not automatically sync with source websites.

**Question 90:** Are these the only required v1 screens?

1. **Week** — generate and adjust the recipe pool  
2. **Groceries** — grouped editable checklist  
3. **Recipe detail** — ingredients, times, instructions, and source link

Is anything missing?

## User

yep

## Assistant

Those three screens define the v1 application.

**Question 91:** Even though v1 will not scale recipes, should we store each recipe’s stated yield or serving count when available for future scaling?

## User

sure

## Assistant

We’ll store source yield as informational data without using it for v1 calculations.

**Question 92:** On Sunday, should the app automatically start a fresh current week while retaining the previous week in SQLite?

## User

sure

## Assistant

A new week begins automatically each Sunday.

**Question 93:** What should the default number of dinners be when generating a week?

## User

actually back to 92 - no for now - let user generate the week manually. for 93 - see new 92

## Assistant

Revised: no automatic rollover. A week exists only when you explicitly generate it, and you choose the dinner count then.

**Question 94:** When generating, should the app automatically assign the current week’s Sunday date, without asking you to choose a date?

## User

sure

## Assistant

The generated week will automatically use the current Sunday’s date.

**Question 95:** Must hands-on and unattended times be known before a recipe is selectable, or may either display as “unknown”? respondent?

## User

should be derivable or backfillable but unknown is fine.

## Assistant

Times should be derived or backfilled where reasonable, but `unknown` remains valid.

**Question 96:** Are cooking instructions required for every selectable recipe, even when they must be backfilled and approved from a reference recipe?

## User

I think so

## Assistant

Cooking instructions are required before a recipe becomes selectable.

**Question 97:** If a source gives an ingredient choice such as “butter or oil,” should true-up resolve it to one concrete default rather than model alternatives in v1? respondent?

## User

for v1 pick one - eventually this falls into the substitution bucket where an item can be one of many things.

## Assistant

We’ll choose one approved default in v1 and defer alternatives to a future substitution model.

**Question 98:** Should random generation treat every recipe equally, without considering previous weeks or cooking history?

## User

ya that's fine for now

## Assistant

Random selection will be uniform and history-blind.

**Question 99:** Should grocery quantities display as familiar fractions where exact—such as `¾ cup`—rather than decimals like `0.75 cup`? respondent? cont?

## User

I don't care

## Assistant

I’ll choose a consistent, readable quantity format without making it a product feature.

**Question 100:** Should the Go server embed the built frontend so v1 runs as one executable plus its SQLite file? respondent? con? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont? respondent? cont?

## User

sure?

## Assistant

We’ll use readable fractions where convenient without making formatting a major feature.

**Question 100:** If two recipes need the same grocery item with different preparations—such as one diced onion and one sliced onion—should the shopping list combine them, with preparation details visible only in the expanded recipe contributions? respondent? cont?

## User

yes ingest time should map to an ingrednent htat is shoppable and list creation time should see them as the same thing.

## Assistant

Exactly: source ingredient lines map to canonical shoppable items during ingestion; list generation aggregates canonical items.

**Question 101:** During true-up, may we lightly rewrite cooking instructions for clarity and consistent step structure, or should source wording be preserved verbatim whenever available? respondent? cont?

## User

yes rewrite away

## Assistant

Instructions may be rewritten into clear, consistent steps during true-up.

**Question 102:** For the clean start, should we preserve the existing Git history but replace the active tree with the minimal v1 project, or create an entirely new repository? respondent? cont?

## User

sure

## Assistant

Which do you prefer?

**A.** Keep this repository and its history, but delete obsolete files from the active branch.  
**B.** Start a completely new repository with no old history.

## User

a

## Assistant

We’ll keep the repository history and replace the active tree with a minimal, authoritative v1 project.

**Question 103:** Is a single local command that starts the Go server and serves the built frontend sufficient, with Docker and deployment infrastructure out of scope? respondent? cont?

## User

ya docker comes later

## Assistant

Docker and deployment are deferred.

**Question 104:** Should production-style local use be one Go binary with the compiled frontend embedded, even if development runs separate Go and frontend dev servers? respondent? cont?

## User

i don't really care

## Assistant

I’ll choose the simplest development and packaging approach without making single-binary delivery a requirement.

**Question 105:** During corpus true-up, do you want to approve recipes one at a time, or review them in small batches with every changed/backfilled field clearly identified? respondent? cont?

## User

one at a time

## Assistant

We’ll true up and approve every recipe individually.

**Question 106:** Should we design the schema using a few difficult representative recipes first, then lock it before migrating the full corpus? respondent? cont?

## User

sure

## Assistant

We’ll validate the schema against difficult recipes—especially burgers, cooked chicken, package-sized ingredients, sections, and ambiguous alternatives—before migrating everything.

**Question 107:** Is this complete exclusion list correct for v1?

No pantry tracking, guest scaling, leftovers,

## User

well actually on 106 I trust you. after 106 questions you should understand the plan.

## Assistant

You’re right. We have enough.

# Shared v1 goal

Build a trustworthy local application that converts a verified family recipe corpus into:

1. an editable, randomly generated weekly recipe pool; and
2. an accurate, recipe-derived grocery checklist.

The product is fundamentally **corpus → week → groceries**.

## v1 product scope

### Week

- User manually generates the current Sunday-based week.
- User chooses only the number of dinners.
- Recipes are selected randomly and uniformly.
- Initial suggestions are unique.
- The week is an unordered pool, not a calendar.
- Actions:
  - add a specific recipe
  - remove
  - swap randomly
  - swap with a specific recipe
  - regenerate the whole pool
- Users may intentionally add duplicate recipes; ingredients count twice.
- No guests, scaling, leftovers, history weighting, or AI planning.

### Grocery list

- Includes every ingredient; no pantry or inventory assumptions.
- Grouped by manually verified store sections.
- Contributing recipes appear behind an expansion control.
- Compatible exact units aggregate automatically.
- Ingredient-specific estimated conversions do not happen.
- Different package sizes and incompatible units remain separate.
- Optional ingredients are included and visibly optional.
- Canonical grocery items have a reviewed shopping mode:
  - measured
  - counted
  - presence-only
- Package sizes remain visible.
- Preparation differences do not prevent aggregation when the purchased item is identical.
- Users can add arbitrary items, remove lines, override weekly quantities, and mark items done.
- Manual additions enter `Other`.
- Completed lines remain visible, crossed out, and de-emphasized.
- Weekly edits never modify recipes.

### Recipe details

- Ingredients organized into sections.
- Read-only ordered cooking instructions.
- Hands-on and unattended time, with `unknown` allowed.
- Stated source yield retained for future use but not used for scaling.
- Source attribution/link.
- Images are optional polish.

### Required screens

1. Week
2. Groceries
3. Recipe detail

All three must work well on iPhone.

## Corpus true-up

- The PDF determines which recipes belong.
- A linked website is authoritative for its recipe details.
- Unlinked recipes may be completed from a similar online reference and adjusted.
- Such recipes are marked `adapted from`, not falsely represented as copies.
- Development-time LLM assistance may propose:
  - missing ingredients
  - quantities
  - mappings
  - times
  - instructions
- You approve each recipe individually.
- Only complete, approved recipes become selectable.
- Every source ingredient retains:
  - original text
  - structured quantity/unit
  - canonical shoppable item
  - preparation
  - optional status
  - section and ordering
- Ambiguous alternatives get one approved v1 default.
- Substitutions are deferred.
- SQLite becomes authoritative after approval.
- No source synchronization or correction history in v1.
- True-up tooling should become the foundation of future onboarding.

## Technical direction

- Go backend
- SQLite
- Goose migrations
- sqlc
- React + TypeScript frontend, likely Vite
- Linear-inspired: polished, compact, dark, responsive, restrained transitions
- No runtime AI
- No authentication
- No offline mode
- No Docker or deployment work
- Local computer hosts the app; iPhone accesses it over the network
- SQLite is the runtime source of truth
- A manually refreshed SQL corpus snapshot may be committed for bootstrap/recovery

## Explicitly out of scope

- Recipe discovery
- AI meal planning
- Guest and serving scaling
- Leftovers
- Pantry inference
- Shopping purchase estimates
- Ingredient substitutions
- Recipe-management UI
- Past-week UI
- Search and filtering
- Store integration, prices, and SKUs
- Authentication and multi-household support
- Deployment infrastructure
- Self-improvement and behavioral scoring

## Repository consolidation

We will retain Git history but replace the active tree with a minimal v1 project. Preserve:

- the original PDF
- useful raw source evidence
- repository history

Everything else—existing Python code, deployment work, superseded specifications, contradictory context, and current UI—is disposable.

## Execution order

1. Write one authoritative v1 scope document from this interview.
2. Clean the active tree and scaffold the Go/React/SQLite project.
3. Design the schema using several difficult representative recipes.
4. Build deterministic ingestion validation and corpus snapshot tooling.
5. True up recipes one at a time with your approval.
6. Build the three-screen application.
7. Validate grocery output against the approved corpus.

No further product interview is needed before starting.

## User

ya before we are done here I want a comprehensive spec doc for v1

## Assistant

Created the comprehensive v1 specification at:

**`V1_SPEC.md`**

It covers:

- Product goals and principles
- Complete v1 scope and exclusions
- Corpus authority and recipe-by-recipe true-up
- Agent-assisted ingestion rules
- Week, grocery list, and recipe-detail behavior
- Quantity and aggregation rules
- Logical SQLite schema and invariants
- Go, Goose, sqlc, React, and TypeScript architecture
- UI direction and mobile requirements
- Failure behavior and test requirements
- Repository consolidation
- Deferred extension seams
- Definition of done

It is explicitly marked as canonical and superseding the old specifications and prototype decisions. No existing files were deleted or modified while creating it.

## User

does this cover our true up?

## Assistant

**Partially.** It defines the true-up contract, authority rules, approval requirements, and completion criteria, but it does not yet include the operational true-up plan.

Still needed:

- Complete PDF recipe inventory and disposition ledger
- One-recipe-at-a-time review template
- Source comparison and conflict procedure
- Rules for agent-backfilled fields
- Approval recording method
- Canonical grocery-item reuse checks
- Import order and difficult pilot recipes
- Corpus progress/audit commands
- Final SQLite seed snapshot procedure

I recommend adding a detailed **True-Up Execution Plan** appendix to `V1_SPEC.md`, then creating the recipe ledger during PDF analysis.

## User

ya we need a plan for the true up as well as the repo cleanup. I'd also like you to track all of the "not yets" from our interview. Things that we need to do but that will muddy v1.

## Assistant

Updated **`V1_SPEC.md`** with three comprehensive additions:

### True-up execution plan

- PDF inventory/disposition ledger
- Source recovery and authority process
- Drafting and canonical grocery mapping
- Mechanical validation
- One-recipe-at-a-time approval packets
- Difficult pilot recipe set
- Corpus-wide audit
- Injectable SQL snapshot and closeout
- Reusable context for future onboarding agents

### Repository cleanup plan

- Preserve dirty work safely before deletion
- Consolidate onto one rebuild branch
- Classify raw, supporting, derived, and obsolete files
- Remove Python, deployment, and superseded context separately
- Establish the Go/React/SQLite scaffold
- Close obsolete PRs and branches
- Validate that a fresh clone contains one coherent direction

### Deferred “not yets” backlog

Tracked by category:

- Planning intelligence and ADK agents
- Scaling, substitutions, inventory, and purchase optimization
- Recipe onboarding and correction UI
- Agent skills and database context tools
- History, future weeks, and list reconciliation
- Export, Apple Notes, and offline shopping
- Docker, hosting, authentication, and multi-household support
- Optional UI and recipe-image polish

The specification is now **1,255 lines** and includes these items in its definition of done.

## User

oh I was thinking we have a few different docs, one for cleanup, one for up nexts, and one for v1 imp

## Assistant

Agreed. I split the monolithic specification into four focused documents:

- **`V1_SPEC.md`** — product behavior, data model, architecture, testing, and definition of done
- **`TRUE_UP_PLAN.md`** — PDF inventory, source recovery, recipe drafting, individual approval, corpus audit, and SQL snapshot
- **`REPO_CLEANUP_PLAN.md`** — preservation, branch consolidation, deletion, scaffold, and closeout
- **`UP_NEXT.md`** — every deferred “not yet,” grouped by planning, quantities, corpus management, lifecycle, platform, and polish

The documents cross-reference each other, while `V1_SPEC.md` remains the product authority. No existing project files have been deleted or committed yet.

## User

great, so what do you think else we should make durable out of that interview? I feel like we had a lot of useful back and forth on that. Do we persist the whole thing in a raw folder?

## Assistant

The interview contains useful **rationale** that the specifications intentionally compress. I would preserve three layers:

## 1. Authoritative documents

Already created:

- `V1_SPEC.md`
- `TRUE_UP_PLAN.md`
- `REPO_CLEANUP_PLAN.md`
- `UP_NEXT.md`

Future agents should start here.

## 2. Product decision record

Add **`PRODUCT_DECISIONS.md`** containing concise entries with:

- decision
- reason
- consequence
- rejected alternative
- condition for revisiting it

Important examples:

- Source completeness versus recipe completeness
- PDF controls membership; linked websites control content
- LLM assistance only during true-up
- Human approval before selection
- Canonical grocery mapping happens during ingestion
- No pantry assumptions
- Exact recipe requirements rather than purchase estimates
- No scaling until quantities and yields are trustworthy
- SQLite as sole runtime truth
- Why the existing implementation is disposable
- Why v1 uses random selection
- Why there is no lock/finalize workflow

This preserves the valuable “why” without forcing agents to reread 106 questions.

## 3. Raw interview archive

Keeping the transcript is reasonable, but I would not put it in `sources/raw/`; that could confuse product discussion with recipe evidence.

Use something like:

```text
archive/interviews/v1-scope-interview.md
```

Mark it clearly:

> Historical, non-authoritative. Decisions were distilled into the current specification and decision record.

The transcript should not be referenced by `AGENTS.md` or loaded as routine agent context. It exists for recovering nuance or resolving ambiguity.

## Additional durable artifacts

During implementation, also preserve:

- **Recipe true-up ledger** — status and approval for every PDF recipe
- **Ingestion decision fixtures** — difficult mappings and why they were accepted
- **Canonical grocery-item notes** — especially false merges and purchased-form distinctions
- **Schema decision records** — only for choices whose rationale is not evident from migrations
- **Representative golden grocery lists** — executable examples of expected behavior

My recommendation: create the concise product decision record and archive the raw transcript separately. The decision record will likely be more useful than the transcript day to day.

## User

I'm down for this plan.

## User

and then please proceed with repo cleanup so that we are left only with our raw pre-cleanup corpus and this new, and useful context documentation.

