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
yield: 4 sandwiches
hands_on:
  min: 30
  max: 30
unattended:
  min: 0
  max: 0
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
      amount: '4'
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
      amount: '8'
      maximum: '12'
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
      amount: '4'
      maximum: '8'
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
      amount: '8'
      maximum: '12'
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
      amount: '4'
      maximum: '8'
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
      amount: '4'
      maximum: '8'
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
      amount: '4'
      maximum: '8'
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
      amount: '8'
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
      amount: '8'
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
      amount: '8'
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
      amount: '4'
      unit: tbsp
instruction_sections:
- name: Method
  steps:
  - Combine the tuna, mayonnaise, yellow mustard, chopped pickles, pickle juice, pepperoncini, diced red onion, optional celery seed, salt, and a generous amount of black pepper until creamy but still textured.
  - Assemble four sandwiches by layering bread with tuna mixture, pickle slices, thinly sliced red onion, cheddar, and top bread.
  - Melt butter in a cast-iron skillet over medium heat. Working in batches as needed, add the sandwiches, press lightly, and cook until golden.
  - Flip, cover the skillet, and continue cooking until the second sides are golden and the cheese is melted.
review:
- field: yield_and_sandwich_quantities
  kind: backfilled
  note: Household re-review replaced the delegated one-sandwich baseline with a four-sandwich family baseline, scaling all quantified source contributions exactly.
  approved: true
- field: ingredient_sections[1].ingredients[3:5]
  kind: conflict-resolved
  note: Kept cheddar from cheese of choice and butter rather than oil while scaling both to four sandwiches.
  approved: true
- field: instruction_sections
  kind: conflict-resolved
  note: Kept the covered-skillet finish rather than the oven alternative and adjusted the method to cook four sandwiches in batches as needed.
  approved: true
---

# Tuna Melt

> Approved bootstrap recipe. YAML front matter is ingested into SQLite; the sections below
> are the checked human-readable view.

## Recipe details

- Source: @mealssheeats Instagram story captured in Recipes.pdf, pages 26–28 (`source`)
- Source checked: 2026-08-17
- Yield: 4 sandwiches
- Hands-on: 30 minutes
- Unattended: 0 minutes

## Ingredients

### Tuna Mix

- 1 can spicy yellowfin tuna, lightly drained
  - Shopping: 4 × 5 oz cans Spicy Yellowfin Tuna — Pantry
  - Preparation: lightly drained
- 2-3 tbsp mayo
  - Shopping: 8–12 tbsp Mayonnaise — Condiments
- 1-2 tsp yellow mustard
  - Shopping: 4–8 tsp Yellow Mustard — Condiments
- 2-3 tbsp finely chopped pickles
  - Shopping: 8–12 tbsp Dill Pickles — Condiments
  - Preparation: finely chopped
- 1-2 tsp pickle juice
  - Shopping: 4–8 tsp Pickle Juice — Condiments
- 1-2 tbsp finely chopped pepperoncini
  - Shopping: 4–8 tbsp Sliced Pepperoncini — Condiments
  - Preparation: finely chopped
- 1-2 tbsp finely diced red onion
  - Shopping: 4–8 tbsp Red Onion — Produce
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
  - Shopping: 8 slices Sandwich Bread — Bakery
- Boar's Head pickle slices
  - Shopping: 8 Dill Pickle Slices — `source brand: Boar's Head` — Condiments
  - Note: source brand: Boar's Head
- Thinly sliced red onion
  - Shopping: Red Onion — Produce
  - Preparation: thinly sliced
- Cheese of choice
  - Shopping: 8 slices Sliced Cheddar Cheese — Dairy
- Butter or oil
  - Shopping: 4 tbsp Butter — Dairy

## Instructions

### Method

1. Combine the tuna, mayonnaise, yellow mustard, chopped pickles, pickle juice, pepperoncini, diced red onion, optional celery seed, salt, and a generous amount of black pepper until creamy but still textured.
2. Assemble four sandwiches by layering bread with tuna mixture, pickle slices, thinly sliced red onion, cheddar, and top bread.
3. Melt butter in a cast-iron skillet over medium heat. Working in batches as needed, add the sandwiches, press lightly, and cook until golden.
4. Flip, cover the skillet, and continue cooking until the second sides are golden and the cheese is melted.

## One-batch grocery preview

### Bakery

- 8 slices Sandwich Bread

### Condiments

- 8–12 tbsp Mayonnaise
- 4–8 tsp Yellow Mustard
- 8–12 tbsp Dill Pickles
- 4–8 tsp Pickle Juice
- 4–8 tbsp Sliced Pepperoncini
- 8 Dill Pickle Slices — `source brand: Boar's Head`

### Dairy

- 8 slices Sliced Cheddar Cheese
- 4 tbsp Butter

### Pantry

- 4 × 5 oz cans Spicy Yellowfin Tuna

### Produce

- 4–8 tbsp Red Onion
- Red Onion

### Spices

- Salt
- Black Pepper
- Celery Seed — optional

## Approved true-up decisions

- `yield_and_sandwich_quantities` — **backfilled:** Household re-review replaced the delegated one-sandwich baseline with a four-sandwich family baseline, scaling all quantified source contributions exactly.
- `ingredient_sections[1].ingredients[3:5]` — **conflict-resolved:** Kept cheddar from cheese of choice and butter rather than oil while scaling both to four sandwiches.
- `instruction_sections` — **conflict-resolved:** Kept the covered-skillet finish rather than the oven alternative and adjusted the method to cook four sandwiches in batches as needed.
