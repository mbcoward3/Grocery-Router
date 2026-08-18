---
format_version: 1
key: tuna-melt
name: Tuna Melt
status: verified
approved_on: '2026-08-17'
source:
  relationship: source
  attribution: '@mealssheeats Instagram story captured in Recipes.pdf, pages 26–28'
  checked_on: '2026-08-17'
yield: 1 sandwich
hands_on:
  min: 15
  max: 15
unattended:
  min: 5
  max: 5
ingredient_sections:
- name: Tuna Mix
  ingredients:
  - source_text: 1 can spicy yellowfin tuna, lightly drained
    grocery_item:
      key: spicy-yellowfin-tuna
      name: Spicy Yellowfin Tuna
      store_section:
        key: pantry
        name: Pantry
      shopping_mode: counted
    quantity:
      kind: exact
      amount: '1'
      package:
        type: can
        size: '5'
        unit: oz
    preparation: lightly drained
  - source_text: 2-3 tbsp mayo
    grocery_item:
      key: mayonnaise
      name: Mayonnaise
      store_section:
        key: condiments
        name: Condiments
      shopping_mode: measured
    quantity:
      kind: range
      amount: '2'
      maximum: '3'
      unit: tbsp
  - source_text: 1-2 tsp yellow mustard
    grocery_item:
      key: yellow-mustard
      name: Yellow Mustard
      store_section:
        key: condiments
        name: Condiments
      shopping_mode: measured
    quantity:
      kind: range
      amount: '1'
      maximum: '2'
      unit: tsp
  - source_text: 2-3 tbsp finely chopped pickles
    grocery_item:
      key: dill-pickles
      name: Dill Pickles
      store_section:
        key: condiments
        name: Condiments
      shopping_mode: measured
    quantity:
      kind: range
      amount: '2'
      maximum: '3'
      unit: tbsp
    preparation: finely chopped
  - source_text: 1-2 tsp pickle juice
    grocery_item:
      key: pickle-juice
      name: Pickle Juice
      store_section:
        key: condiments
        name: Condiments
      shopping_mode: measured
    quantity:
      kind: range
      amount: '1'
      maximum: '2'
      unit: tsp
  - source_text: 1-2 tbsp finely chopped pepperoncini
    grocery_item:
      key: sliced-pepperoncini
      name: Sliced Pepperoncini
      store_section:
        key: condiments
        name: Condiments
      shopping_mode: measured
    quantity:
      kind: range
      amount: '1'
      maximum: '2'
      unit: tbsp
    preparation: finely chopped
  - source_text: 1-2 tbsp finely diced red onion
    grocery_item:
      key: red-onion
      name: Red Onion
      store_section:
        key: produce
        name: Produce
      shopping_mode: counted
    quantity:
      kind: range
      amount: '1'
      maximum: '2'
      unit: tbsp
    preparation: finely diced
  - source_text: Salt
    grocery_item:
      key: salt
      name: Salt
      store_section:
        key: spices
        name: Spices
      shopping_mode: measured
    quantity:
      kind: unspecified
  - source_text: Black pepper (generous)
    grocery_item:
      key: black-pepper
      name: Black Pepper
      store_section:
        key: spices
        name: Spices
      shopping_mode: measured
    quantity:
      kind: unspecified
    preparation: generous
  - source_text: Celery seed (optional)
    grocery_item:
      key: celery-seed
      name: Celery Seed
      store_section:
        key: spices
        name: Spices
      shopping_mode: measured
    quantity:
      kind: unspecified
    optional: true
- name: Sandwich
  ingredients:
  - source_text: Bread of choice
    grocery_item:
      key: sandwich-bread
      name: Sandwich Bread
      store_section:
        key: bakery
        name: Bakery
      shopping_mode: counted
    quantity:
      kind: exact
      amount: '2'
      unit: slice
  - source_text: Boar's Head pickle slices
    grocery_item:
      key: dill-pickle-slices
      name: Dill Pickle Slices
      store_section:
        key: condiments
        name: Condiments
      shopping_mode: counted
    quantity:
      kind: exact
      amount: '2'
      unit: each
    note: 'source brand: Boar''s Head'
  - source_text: Thinly sliced red onion
    grocery_item:
      key: red-onion
      name: Red Onion
      store_section:
        key: produce
        name: Produce
      shopping_mode: counted
    quantity:
      kind: unspecified
    preparation: thinly sliced
  - source_text: Cheese of choice
    grocery_item:
      key: sliced-cheddar-cheese
      name: Sliced Cheddar Cheese
      store_section:
        key: dairy
        name: Dairy
      shopping_mode: counted
    quantity:
      kind: exact
      amount: '2'
      unit: slice
  - source_text: Butter or oil
    grocery_item:
      key: butter
      name: Butter
      store_section:
        key: dairy
        name: Dairy
      shopping_mode: measured
    quantity:
      kind: exact
      amount: '1'
      unit: tbsp
instruction_sections:
- name: Method
  steps:
  - Combine the tuna, mayonnaise, yellow mustard, chopped pickles, pickle juice, pepperoncini, diced red onion, optional celery seed, salt, and a generous amount of black pepper until creamy but still textured.
  - Layer one bread slice with the tuna mixture, pickle slices, thinly sliced red onion, cheddar, and the second bread slice.
  - Melt the butter in a cast-iron skillet over medium heat. Add the sandwich, press lightly, and cook until golden.
  - Flip, cover the skillet, and continue cooking until the second side is golden and the cheese is melted.
review:
- field: yield_and_sandwich_quantities
  kind: backfilled
  note: Under delegated household review, backfilled one sandwich using a 5-ounce tuna can, 2 bread slices, 2 pickle slices, 2 cheddar slices, and 1 tablespoon butter. Logged for revisit in trueup/CONTROVERSIAL_CALLS.md.
  approved: true
- field: ingredient_sections[1].ingredients[3:5]
  kind: conflict-resolved
  note: Selected cheddar from cheese of choice and butter rather than oil.
  approved: true
- field: instruction_sections
  kind: conflict-resolved
  note: Selected the source-listed covered-skillet finish rather than its oven alternative and preserved all tuna-mix ranges.
  approved: true
---

# Tuna Melt

> Approved bootstrap recipe. YAML front matter is ingested into SQLite; the sections below
> are the checked human-readable view.

## Recipe details

- Source: @mealssheeats Instagram story captured in Recipes.pdf, pages 26–28 (`source`)
- Source checked: 2026-08-17
- Yield: 1 sandwich
- Hands-on: 15 minutes
- Unattended: 5 minutes

## Ingredients

### Tuna Mix

- 1 can spicy yellowfin tuna, lightly drained
  - Shopping: 1 × 5 oz can Spicy Yellowfin Tuna — Pantry
  - Preparation: lightly drained
- 2-3 tbsp mayo
  - Shopping: 2–3 tbsp Mayonnaise — Condiments
- 1-2 tsp yellow mustard
  - Shopping: 1–2 tsp Yellow Mustard — Condiments
- 2-3 tbsp finely chopped pickles
  - Shopping: 2–3 tbsp Dill Pickles — Condiments
  - Preparation: finely chopped
- 1-2 tsp pickle juice
  - Shopping: 1–2 tsp Pickle Juice — Condiments
- 1-2 tbsp finely chopped pepperoncini
  - Shopping: 1–2 tbsp Sliced Pepperoncini — Condiments
  - Preparation: finely chopped
- 1-2 tbsp finely diced red onion
  - Shopping: 1–2 tbsp Red Onion — Produce
  - Preparation: finely diced
- Salt
  - Shopping: Salt — Spices
- Black pepper (generous)
  - Shopping: Black Pepper — Spices
  - Preparation: generous
- Celery seed (optional)
  - Shopping: Celery Seed — optional — Spices
  - Optional: yes

### Sandwich

- Bread of choice
  - Shopping: 2 slices Sandwich Bread — Bakery
- Boar's Head pickle slices
  - Shopping: 2 Dill Pickle Slices — `source brand: Boar's Head` — Condiments
  - Note: source brand: Boar's Head
- Thinly sliced red onion
  - Shopping: Red Onion — Produce
  - Preparation: thinly sliced
- Cheese of choice
  - Shopping: 2 slices Sliced Cheddar Cheese — Dairy
- Butter or oil
  - Shopping: 1 tbsp Butter — Dairy

## Instructions

### Method

1. Combine the tuna, mayonnaise, yellow mustard, chopped pickles, pickle juice, pepperoncini, diced red onion, optional celery seed, salt, and a generous amount of black pepper until creamy but still textured.
2. Layer one bread slice with the tuna mixture, pickle slices, thinly sliced red onion, cheddar, and the second bread slice.
3. Melt the butter in a cast-iron skillet over medium heat. Add the sandwich, press lightly, and cook until golden.
4. Flip, cover the skillet, and continue cooking until the second side is golden and the cheese is melted.

## One-batch grocery preview

### Bakery

- 2 slices Sandwich Bread

### Condiments

- 2–3 tbsp Mayonnaise
- 1–2 tsp Yellow Mustard
- 2–3 tbsp Dill Pickles
- 1–2 tsp Pickle Juice
- 1–2 tbsp Sliced Pepperoncini
- 2 Dill Pickle Slices — `source brand: Boar's Head`

### Dairy

- 2 slices Sliced Cheddar Cheese
- 1 tbsp Butter

### Pantry

- 1 × 5 oz can Spicy Yellowfin Tuna

### Produce

- 1–2 tbsp Red Onion
- Red Onion

### Spices

- Salt
- Black Pepper
- Celery Seed — optional

## Approved true-up decisions

- `yield_and_sandwich_quantities` — **backfilled:** Under delegated household review, backfilled one sandwich using a 5-ounce tuna can, 2 bread slices, 2 pickle slices, 2 cheddar slices, and 1 tablespoon butter. Logged for revisit in trueup/CONTROVERSIAL_CALLS.md.
- `ingredient_sections[1].ingredients[3:5]` — **conflict-resolved:** Selected cheddar from cheese of choice and butter rather than oil.
- `instruction_sections` — **conflict-resolved:** Selected the source-listed covered-skillet finish rather than its oven alternative and preserved all tuna-mix ranges.
