---
format_version: 1
key: chicken-and-dumplings
name: Chicken and Dumplings
status: verified
approved_on: '2026-08-17'
source:
  relationship: source
  attribution: Lil Luna recipe captured in Recipes.pdf, pages 13–15
  checked_on: '2026-08-17'
yield: 6 servings
hands_on:
  min: 10
  max: 10
unattended:
  min: 25
  max: 25
ingredient_sections:
- name: Soup
  ingredients:
  - source_text: 4 c chicken broth
    grocery_item:
      key: chicken-broth
      name: Chicken Broth
      store_section:
        key: pantry
        name: Pantry
      shopping_mode: measured
    quantity:
      kind: exact
      amount: '4'
      unit: cup
  - source_text: 1 10.75 oz can cream of chicken soup
    grocery_item:
      key: cream-of-chicken-soup
      name: Cream of Chicken Soup
      store_section:
        key: pantry
        name: Pantry
      shopping_mode: counted
    quantity:
      kind: exact
      amount: '1'
      package:
        type: can
        size: '10.75'
        unit: oz
  - source_text: 2 13 oz cans cooked and shredded chicken
    grocery_item:
      key: canned-chicken
      name: Canned Chicken
      store_section:
        key: meat
        name: Meat
      shopping_mode: counted
    quantity:
      kind: exact
      amount: '2'
      package:
        type: can
        size: '13'
        unit: oz
    preparation: cooked and shredded
  - source_text: 1 8.75 oz can corn
    grocery_item:
      key: canned-corn
      name: Canned Corn
      store_section:
        key: pantry
        name: Pantry
      shopping_mode: counted
    quantity:
      kind: exact
      amount: '1'
      package:
        type: can
        size: '8.75'
        unit: oz
  - source_text: 1 14.5 oz can sliced carrots
    grocery_item:
      key: canned-sliced-carrots
      name: Canned Sliced Carrots
      store_section:
        key: pantry
        name: Pantry
      shopping_mode: counted
    quantity:
      kind: exact
      amount: '1'
      package:
        type: can
        size: '14.5'
        unit: oz
  - source_text: 2 16.3 oz tubes refrigerated biscuits cut into quarters
    grocery_item:
      key: refrigerated-biscuits
      name: Refrigerated Biscuits
      store_section:
        key: dairy
        name: Dairy
      shopping_mode: counted
    quantity:
      kind: exact
      amount: '2'
      package:
        type: tube
        size: '16.3'
        unit: oz
    preparation: cut into quarters
  - source_text: fresh parsley
    grocery_item:
      key: fresh-parsley
      name: Fresh Parsley
      store_section:
        key: produce
        name: Produce
      shopping_mode: measured
    quantity:
      kind: unspecified
    preparation: for topping
    optional: true
instruction_sections:
- name: Method
  steps:
  - Bring the chicken broth, chicken, and cream of chicken soup to a boil in a large pot.
  - Reduce the heat to low, cover, and simmer for 5 minutes.
  - Add the quartered biscuits, corn, and carrots. Cover and simmer for 15 to 20 minutes, stirring occasionally so the biscuits do not stick together.
  - Serve in bowls and top with fresh parsley if desired.
review:
- field: corpus_membership
  kind: conflict-resolved
  note: Included the recipe under the household delegated completion instruction despite the unexplained question mark in the PDF title; logged for revisit in trueup/CONTROVERSIAL_CALLS.md.
  approved: true
- field: yield
  kind: rewritten
  note: Recorded the clearly displayed six servings and treated the adjacent AE characters as capture noise.
  approved: true
- field: instruction_sections
  kind: rewritten
  note: Preserved the captured method and omitted only the non-action ENJOY line.
  approved: true
---

# Chicken and Dumplings

> Approved bootstrap recipe. YAML front matter is ingested into SQLite; the sections below
> are the checked human-readable view.

## Recipe details

- Source: Lil Luna recipe captured in Recipes.pdf, pages 13–15 (`source`)
- Source checked: 2026-08-17
- Yield: 6 servings
- Hands-on: 10 minutes
- Unattended: 25 minutes

## Ingredients

### Soup

- 4 c chicken broth
  - Shopping: 4 cup Chicken Broth — Pantry
- 1 10.75 oz can cream of chicken soup
  - Shopping: 1 × 10.75 oz can Cream of Chicken Soup — Pantry
- 2 13 oz cans cooked and shredded chicken
  - Shopping: 2 × 13 oz cans Canned Chicken — Meat
  - Preparation: cooked and shredded
- 1 8.75 oz can corn
  - Shopping: 1 × 8.75 oz can Canned Corn — Pantry
- 1 14.5 oz can sliced carrots
  - Shopping: 1 × 14.5 oz can Canned Sliced Carrots — Pantry
- 2 16.3 oz tubes refrigerated biscuits cut into quarters
  - Shopping: 2 × 16.3 oz tubes Refrigerated Biscuits — Dairy
  - Preparation: cut into quarters
- fresh parsley
  - Shopping: Fresh Parsley — optional — Produce
  - Preparation: for topping
  - Optional: yes

## Instructions

### Method

1. Bring the chicken broth, chicken, and cream of chicken soup to a boil in a large pot.
2. Reduce the heat to low, cover, and simmer for 5 minutes.
3. Add the quartered biscuits, corn, and carrots. Cover and simmer for 15 to 20 minutes, stirring occasionally so the biscuits do not stick together.
4. Serve in bowls and top with fresh parsley if desired.

## One-batch grocery preview

### Dairy

- 2 × 16.3 oz tubes Refrigerated Biscuits

### Meat

- 2 × 13 oz cans Canned Chicken

### Pantry

- 4 cup Chicken Broth
- 1 × 10.75 oz can Cream of Chicken Soup
- 1 × 8.75 oz can Canned Corn
- 1 × 14.5 oz can Canned Sliced Carrots

### Produce

- Fresh Parsley — optional

## Approved true-up decisions

- `corpus_membership` — **conflict-resolved:** Included the recipe under the household delegated completion instruction despite the unexplained question mark in the PDF title; logged for revisit in trueup/CONTROVERSIAL_CALLS.md.
- `yield` — **rewritten:** Recorded the clearly displayed six servings and treated the adjacent AE characters as capture noise.
- `instruction_sections` — **rewritten:** Preserved the captured method and omitted only the non-action ENJOY line.
