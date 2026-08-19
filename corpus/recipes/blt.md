---
format_version: 1
key: blt
name: BLT
status: verified
approved_on: '2026-08-17'
source:
  relationship: source
  attribution: Household notes in Recipes.pdf, pages 10–11
  checked_on: '2026-08-17'
yield: 4 sandwiches
hands_on:
  min: 20
  max: 20
unattended:
  min: 0
  max: 0
ingredient_sections:
- name: Sandwiches
  ingredients:
  - source_text: bread
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
  - source_text: Bacon
    grocery_item:
      key: bacon
      name: Bacon
      store_section:
        key: meat
        name: Meat
      shopping_mode: counted
    quantity:
      kind: exact
      amount: '1'
      package:
        type: package
  - source_text: Lettuce
    grocery_item:
      key: lettuce
      name: Lettuce
      store_section:
        key: produce
        name: Produce
      shopping_mode: counted
    quantity:
      kind: exact
      amount: '1'
      unit: each
    preparation: head; leaves washed and dried
  - source_text: Tomato
    grocery_item:
      key: tomato
      name: Tomato
      store_section:
        key: produce
        name: Produce
      shopping_mode: counted
    quantity:
      kind: exact
      amount: '2'
      unit: each
    preparation: sliced
  - source_text: Mayo
    grocery_item:
      key: mayonnaise
      name: Mayonnaise
      store_section:
        key: condiments
        name: Condiments
      shopping_mode: measured
    quantity:
      kind: unspecified
instruction_sections:
- name: Method
  steps:
  - Cook the bacon until crisp, then drain it.
  - Toast the bread if desired.
  - Spread mayonnaise on the bread and layer with lettuce, sliced tomato, and bacon.
  - Close the sandwiches and serve.
review:
- field: yield_and_quantities
  kind: backfilled
  note: Household re-review kept the four-sandwich baseline with 8 bread slices but changed shopping requirements to one package bacon, one head lettuce, and 2 tomatoes; mayonnaise remains unquantified.
  approved: true
- field: instruction_sections
  kind: backfilled
  note: The household source has no method; added a minimal assembly method without introducing ingredients.
  approved: true
---

# BLT

> Approved bootstrap recipe. YAML front matter is ingested into SQLite; the sections below
> are the checked human-readable view.

## Recipe details

- Source: Household notes in Recipes.pdf, pages 10–11 (`source`)
- Source checked: 2026-08-17
- Yield: 4 sandwiches
- Hands-on: 20 minutes
- Unattended: 0 minutes

## Ingredients

### Sandwiches

- bread
  - Shopping: 8 slices Sandwich Bread — Bakery
- Bacon
  - Shopping: 1 package Bacon — Meat
- Lettuce
  - Shopping: 1 Lettuce — Produce
  - Preparation: head; leaves washed and dried
- Tomato
  - Shopping: 2 Tomato — Produce
  - Preparation: sliced
- Mayo
  - Shopping: Mayonnaise — Condiments

## Instructions

### Method

1. Cook the bacon until crisp, then drain it.
2. Toast the bread if desired.
3. Spread mayonnaise on the bread and layer with lettuce, sliced tomato, and bacon.
4. Close the sandwiches and serve.

## One-batch grocery preview

### Bakery

- 8 slices Sandwich Bread

### Condiments

- Mayonnaise

### Meat

- 1 package Bacon

### Produce

- 1 Lettuce
- 2 Tomato

## Approved true-up decisions

- `yield_and_quantities` — **backfilled:** Household re-review kept the four-sandwich baseline with 8 bread slices but changed shopping requirements to one package bacon, one head lettuce, and 2 tomatoes; mayonnaise remains unquantified.
- `instruction_sections` — **backfilled:** The household source has no method; added a minimal assembly method without introducing ingredients.
