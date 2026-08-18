---
format_version: 1
key: hamburgers
name: Hamburgers
status: verified
approved_on: 2026-08-17
source:
  relationship: source
  attribution: Household notes in Recipes.pdf, page 10
  checked_on: 2026-08-17
hands_on:
  min: 25
  max: 30
unattended:
  min: 0
  max: 0
ingredient_sections:
  - name: Burgers
    ingredients:
      - source_text: 2lb ground beef
        grocery_item:
          key: ground-beef
          name: Ground Beef
          store_section: {key: meat, name: Meat}
          shopping_mode: measured
        quantity: {kind: exact, amount: "2", unit: lb}
      - source_text: Onion
        grocery_item:
          key: onion
          name: Onion
          store_section: {key: produce, name: Produce}
          shopping_mode: counted
        quantity: {kind: exact, amount: "1", unit: each}
        preparation: sliced into rings for topping
      - source_text: Buns
        grocery_item:
          key: hamburger-buns
          name: Hamburger Buns
          store_section: {key: bakery, name: Bakery}
          shopping_mode: counted
        quantity:
          kind: exact
          amount: "1"
          package: {type: package, size: "8", unit: each}
      - source_text: Mayo
        grocery_item:
          key: mayonnaise
          name: Mayonnaise
          store_section: {key: condiments, name: Condiments}
          shopping_mode: presence-only
        quantity: {kind: unspecified}
      - source_text: Mustard
        grocery_item:
          key: mustard
          name: Mustard
          store_section: {key: condiments, name: Condiments}
          shopping_mode: presence-only
        quantity: {kind: unspecified}
      - source_text: Ketchup
        grocery_item:
          key: ketchup
          name: Ketchup
          store_section: {key: condiments, name: Condiments}
          shopping_mode: presence-only
        quantity: {kind: unspecified}
      - source_text: Salt
        grocery_item:
          key: salt
          name: Salt
          store_section: {key: spices, name: Spices}
          shopping_mode: presence-only
        quantity: {kind: unspecified}
        preparation: sprinkled on formed patties
      - source_text: Pepper
        grocery_item:
          key: black-pepper
          name: Black Pepper
          store_section: {key: spices, name: Spices}
          shopping_mode: presence-only
        quantity: {kind: unspecified}
        preparation: sprinkled on formed patties
      - source_text: Garlic powder
        grocery_item:
          key: garlic-powder
          name: Garlic Powder
          store_section: {key: spices, name: Spices}
          shopping_mode: presence-only
        quantity: {kind: unspecified}
        preparation: sprinkled on formed patties
      - source_text: 8 slices Colby Jack cheese
        grocery_item:
          key: sliced-colby-jack-cheese
          name: Sliced Colby Jack Cheese
          store_section: {key: dairy, name: Dairy}
          shopping_mode: counted
        quantity: {kind: exact, amount: "8", unit: slice}
instruction_sections:
  - name: Method
    steps:
      - Divide the ground beef into patties of the preferred size, handling the meat lightly. Press a shallow indentation into the center of each patty.
      - Sprinkle the tops of the patties with salt, black pepper, and garlic powder.
      - Heat a grill or skillet over medium-high heat.
      - Cook the patties until browned on the first side, then flip and continue cooking until the centers reach 160°F. Add Colby Jack during the final minute for any cheeseburgers.
      - Briefly toast the cut sides of the buns on the grill or skillet if desired.
      - Serve on buns with onion rings, mayonnaise, mustard, and ketchup.
review:
  - field: hands_on
    kind: backfilled
    note: Estimated a 25 to 30 minute hands-on range from the approved grill or skillet method.
    approved: true
  - field: ingredient_sections[0].ingredients[1]
    kind: backfilled
    note: Set the shopping requirement to one onion and clarified that it is sliced into rings for topping.
    approved: true
  - field: ingredient_sections[0].ingredients[2]
    kind: backfilled
    note: Set the shopping requirement to one 8-count package of buns without claiming a burger yield.
    approved: true
  - field: ingredient_sections[0].ingredients[3:9].quantity
    kind: backfilled
    note: Preserved condiments and seasonings as presence-only because the household source gives no quantities.
    approved: true
  - field: ingredient_sections[0].ingredients[9]
    kind: backfilled
    note: Added eight slices of Colby Jack cheese as the approved current default.
    approved: true
  - field: instruction_sections
    kind: backfilled
    note: Added a basic grill-or-skillet method; seasonings are sprinkled onto formed patties.
    approved: true
---

# Hamburgers

> Approved bootstrap recipe. YAML front matter is ingested into SQLite; the sections below
> are the checked human-readable view.

## Recipe details

- Source: Household notes in Recipes.pdf, page 10 (`source`)
- Source checked: 2026-08-17
- Yield: unknown
- Hands-on: 25–30 minutes
- Unattended: 0 minutes

## Ingredients

### Burgers

- 2lb ground beef
  - Shopping: 2 lb Ground Beef — Meat
- Onion
  - Shopping: 1 Onion — Produce
  - Preparation: sliced into rings for topping
- Buns
  - Shopping: 1 × 8-count package Hamburger Buns — Bakery
- Mayo
  - Shopping: Mayonnaise — Condiments
- Mustard
  - Shopping: Mustard — Condiments
- Ketchup
  - Shopping: Ketchup — Condiments
- Salt
  - Shopping: Salt — Spices
  - Preparation: sprinkled on formed patties
- Pepper
  - Shopping: Black Pepper — Spices
  - Preparation: sprinkled on formed patties
- Garlic powder
  - Shopping: Garlic Powder — Spices
  - Preparation: sprinkled on formed patties
- 8 slices Colby Jack cheese
  - Shopping: 8 slices Sliced Colby Jack Cheese — Dairy

## Instructions

### Method

1. Divide the ground beef into patties of the preferred size, handling the meat lightly. Press a shallow indentation into the center of each patty.
2. Sprinkle the tops of the patties with salt, black pepper, and garlic powder.
3. Heat a grill or skillet over medium-high heat.
4. Cook the patties until browned on the first side, then flip and continue cooking until the centers reach 160°F. Add Colby Jack during the final minute for any cheeseburgers.
5. Briefly toast the cut sides of the buns on the grill or skillet if desired.
6. Serve on buns with onion rings, mayonnaise, mustard, and ketchup.

## One-batch grocery preview

### Bakery

- 1 × 8-count package Hamburger Buns

### Condiments

- Mayonnaise
- Mustard
- Ketchup

### Dairy

- 8 slices Sliced Colby Jack Cheese

### Meat

- 2 lb Ground Beef

### Produce

- 1 Onion

### Spices

- Salt
- Black Pepper
- Garlic Powder

## Approved true-up decisions

- `hands_on` — **backfilled:** Estimated a 25 to 30 minute hands-on range from the approved grill or skillet method.
- `ingredient_sections[0].ingredients[1]` — **backfilled:** Set the shopping requirement to one onion and clarified that it is sliced into rings for topping.
- `ingredient_sections[0].ingredients[2]` — **backfilled:** Set the shopping requirement to one 8-count package of buns without claiming a burger yield.
- `ingredient_sections[0].ingredients[3:9].quantity` — **backfilled:** Preserved condiments and seasonings as presence-only because the household source gives no quantities.
- `ingredient_sections[0].ingredients[9]` — **backfilled:** Added eight slices of Colby Jack cheese as the approved current default.
- `instruction_sections` — **backfilled:** Added a basic grill-or-skillet method; seasonings are sprinkled onto formed patties.
