---
format_version: 1
key: biscuits-and-gravy
name: Biscuits and Gravy
status: verified
approved_on: '2026-08-17'
source:
  relationship: source
  attribution: Household notes in Recipes.pdf, page 10
  checked_on: '2026-08-17'
yield: 6 servings
hands_on:
  min: 25
  max: 25
unattended:
  min: 15
  max: 15
ingredient_sections:
- name: Biscuits and Gravy
  ingredients:
  - source_text: 1 sausage tube spicy or regular
    grocery_item:
      key: breakfast-sausage
      name: Breakfast Sausage
      store_section:
        key: meat
        name: Meat
      shopping_mode: counted
    quantity:
      kind: exact
      amount: '1'
      package:
        type: tube
        size: '1'
        unit: lb
    preparation: regular
  - source_text: Tube of biscuits
    grocery_item:
      key: refrigerated-biscuits
      name: Refrigerated Biscuits
      store_section:
        key: dairy
        name: Dairy
      shopping_mode: counted
    quantity:
      kind: exact
      amount: '1'
      package:
        type: tube
  - source_text: Onion
    grocery_item:
      key: onion
      name: Onion
      store_section:
        key: produce
        name: Produce
      shopping_mode: counted
    quantity:
      kind: exact
      amount: 1/2
      unit: each
    preparation: finely chopped
  - source_text: Flour
    grocery_item:
      key: all-purpose-flour
      name: All-Purpose Flour
      store_section:
        key: baking
        name: Baking
      shopping_mode: measured
    quantity:
      kind: exact
      amount: 1/4
      unit: cup
  - source_text: Butter
    grocery_item:
      key: butter
      name: Butter
      store_section:
        key: dairy
        name: Dairy
      shopping_mode: measured
    quantity:
      kind: exact
      amount: '2'
      unit: tbsp
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
  - source_text: Pepper
    grocery_item:
      key: black-pepper
      name: Black Pepper
      store_section:
        key: spices
        name: Spices
      shopping_mode: measured
    quantity:
      kind: unspecified
  - source_text: 2 1/2 cups milk
    grocery_item:
      key: milk
      name: Milk
      store_section:
        key: dairy
        name: Dairy
      shopping_mode: measured
    quantity:
      kind: exact
      amount: 2 1/2
      unit: cup
instruction_sections:
- name: Method
  steps:
  - Bake the biscuits according to their package directions.
  - While the biscuits bake, cook the breakfast sausage and onion in a large skillet over medium heat, breaking up the sausage, until browned and cooked through.
  - Stir in the butter, then sprinkle in the flour and cook for 1 minute.
  - Gradually stir in the milk. Simmer, stirring often, until the gravy thickens, then season with salt and black pepper.
  - Split the warm biscuits and spoon the sausage gravy over them.
review:
- field: ingredient_sections
  kind: conflict-resolved
  note: Household re-review approved regular 1-pound breakfast sausage, one standard biscuit tube, half an onion, 1/4 cup flour, 2 tablespoons butter, and 2 1/2 cups milk.
  approved: true
- field: instruction_sections
  kind: backfilled
  note: The household source has no method; completed a standard biscuit-package and skillet-gravy method using every approved ingredient.
  approved: true
---

# Biscuits and Gravy

> Approved bootstrap recipe. YAML front matter is ingested into SQLite; the sections below
> are the checked human-readable view.

## Recipe details

- Source: Household notes in Recipes.pdf, page 10 (`source`)
- Source checked: 2026-08-17
- Yield: 6 servings
- Hands-on: 25 minutes
- Unattended: 15 minutes

## Ingredients

### Biscuits and Gravy

- 1 sausage tube spicy or regular
  - Shopping: 1 × 1 lb tube Breakfast Sausage — Meat
  - Preparation: regular
- Tube of biscuits
  - Shopping: 1 tube Refrigerated Biscuits — Dairy
- Onion
  - Shopping: 1/2 Onion — Produce
  - Preparation: finely chopped
- Flour
  - Shopping: 1/4 cup All-Purpose Flour — Baking
- Butter
  - Shopping: 2 tbsp Butter — Dairy
- Salt
  - Shopping: Salt — Spices
- Pepper
  - Shopping: Black Pepper — Spices
- 2 1/2 cups milk
  - Shopping: 2 1/2 cup Milk — Dairy

## Instructions

### Method

1. Bake the biscuits according to their package directions.
2. While the biscuits bake, cook the breakfast sausage and onion in a large skillet over medium heat, breaking up the sausage, until browned and cooked through.
3. Stir in the butter, then sprinkle in the flour and cook for 1 minute.
4. Gradually stir in the milk. Simmer, stirring often, until the gravy thickens, then season with salt and black pepper.
5. Split the warm biscuits and spoon the sausage gravy over them.

## One-batch grocery preview

### Baking

- 1/4 cup All-Purpose Flour

### Dairy

- 1 tube Refrigerated Biscuits
- 2 tbsp Butter
- 2 1/2 cup Milk

### Meat

- 1 × 1 lb tube Breakfast Sausage

### Produce

- 1/2 Onion

### Spices

- Salt
- Black Pepper

## Approved true-up decisions

- `ingredient_sections` — **conflict-resolved:** Household re-review approved regular 1-pound breakfast sausage, one standard biscuit tube, half an onion, 1/4 cup flour, 2 tablespoons butter, and 2 1/2 cups milk.
- `instruction_sections` — **backfilled:** The household source has no method; completed a standard biscuit-package and skillet-gravy method using every approved ingredient.
