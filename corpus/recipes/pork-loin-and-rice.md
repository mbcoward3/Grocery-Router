---
format_version: 1
key: pork-loin-and-rice
name: Pork Loin and Rice
status: verified
approved_on: 2026-08-17
source:
  relationship: source
  attribution: Household notes in Recipes.pdf, page 10
  checked_on: 2026-08-17
hands_on:
  min: 10
  max: 10
unattended:
  min: 50
  max: 65
ingredient_sections:
  - name: Pork and Rice
    ingredients:
      - source_text: pork loin
        grocery_item:
          key: pork-loin
          name: Pork Loin
          store_section: {key: meat, name: Meat}
          shopping_mode: measured
        quantity: {kind: exact, amount: "2", unit: lb}
      - source_text: White rice
        grocery_item:
          key: white-rice
          name: White Rice
          store_section: {key: pantry, name: Pantry}
          shopping_mode: measured
        quantity: {kind: exact, amount: "2", unit: cup}
        preparation: uncooked
      - source_text: water, according to the rice package directions
        grocery_item:
          key: water
          name: Water
          store_section: {key: non-shopping, name: Non-shopping}
          shopping_mode: measured
        quantity: {kind: unspecified}
        non_shopping: true
      - source_text: Soup sauce
        grocery_item:
          key: soy-sauce
          name: Soy Sauce
          store_section: {key: condiments, name: Condiments}
          shopping_mode: measured
        quantity: {kind: unspecified}
        preparation: for serving with the rice only
instruction_sections:
  - name: Method
    steps:
      - Preheat the oven to 375°F and place the pork loin in a baking dish.
      - Bake for 45 to 60 minutes, until the thickest part reaches 145°F.
      - Remove the pork from the oven and let it rest for at least 3 minutes before slicing.
      - While the pork bakes, cook 2 cups uncooked white rice with water according to the rice package directions.
      - Serve the sliced pork with the cooked rice and add soy sauce to the rice as desired.
review:
  - field: ingredient_sections[0].ingredients[0].quantity
    kind: backfilled
    note: Household review set the baseline pork loin requirement to two pounds.
    approved: true
  - field: ingredient_sections[0].ingredients[1].quantity
    kind: backfilled
    note: Household review set the baseline rice requirement to two cups uncooked white rice.
    approved: true
  - field: ingredient_sections[0].ingredients[2]
    kind: backfilled
    note: Added the package-directed water needed to cook the rice and explicitly marked it non-shopping without inventing a water ratio.
    approved: true
  - field: ingredient_sections[0].ingredients[3]
    kind: conflict-resolved
    note: Household review corrected the source typo "Soup sauce" to soy sauce and clarified that it is used only on the rice, with no fixed quantity.
    approved: true
  - field: hands_on_and_unattended
    kind: backfilled
    note: Estimated 10 minutes hands-on and 50 to 65 minutes unattended from the completed baked-pork method.
    approved: true
  - field: instruction_sections
    kind: backfilled
    note: The household source contains no method. Household review selected baking; the completed method uses 375°F, the safe 145°F endpoint, and a minimum three-minute rest without introducing additional food ingredients.
    approved: true
---

# Pork Loin and Rice

> Approved bootstrap recipe. YAML front matter is ingested into SQLite; the sections below
> are the checked human-readable view.

## Recipe details

- Source: Household notes in Recipes.pdf, page 10 (`source`)
- Source checked: 2026-08-17
- Yield: unknown
- Hands-on: 10 minutes
- Unattended: 50–65 minutes

## Ingredients

### Pork and Rice

- pork loin
  - Shopping: 2 lb Pork Loin — Meat
- White rice
  - Shopping: 2 cup White Rice — Pantry
  - Preparation: uncooked
- water, according to the rice package directions
  - Recipe only: Water — not added to the grocery list
- Soup sauce
  - Shopping: Soy Sauce — Condiments
  - Preparation: for serving with the rice only

## Instructions

### Method

1. Preheat the oven to 375°F and place the pork loin in a baking dish.
2. Bake for 45 to 60 minutes, until the thickest part reaches 145°F.
3. Remove the pork from the oven and let it rest for at least 3 minutes before slicing.
4. While the pork bakes, cook 2 cups uncooked white rice with water according to the rice package directions.
5. Serve the sliced pork with the cooked rice and add soy sauce to the rice as desired.

## One-batch grocery preview

### Condiments

- Soy Sauce

### Meat

- 2 lb Pork Loin

### Pantry

- 2 cup White Rice

## Approved true-up decisions

- `ingredient_sections[0].ingredients[0].quantity` — **backfilled:** Household review set the baseline pork loin requirement to two pounds.
- `ingredient_sections[0].ingredients[1].quantity` — **backfilled:** Household review set the baseline rice requirement to two cups uncooked white rice.
- `ingredient_sections[0].ingredients[2]` — **backfilled:** Added the package-directed water needed to cook the rice and explicitly marked it non-shopping without inventing a water ratio.
- `ingredient_sections[0].ingredients[3]` — **conflict-resolved:** Household review corrected the source typo "Soup sauce" to soy sauce and clarified that it is used only on the rice, with no fixed quantity.
- `hands_on_and_unattended` — **backfilled:** Estimated 10 minutes hands-on and 50 to 65 minutes unattended from the completed baked-pork method.
- `instruction_sections` — **backfilled:** The household source contains no method. Household review selected baking; the completed method uses 375°F, the safe 145°F endpoint, and a minimum three-minute rest without introducing additional food ingredients.
