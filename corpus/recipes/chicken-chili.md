---
format_version: 1
key: chicken-chili
name: Chicken Chili
status: verified
approved_on: '2026-08-17'
source:
  relationship: source
  attribution: Help with House Plants Facebook post captured in Recipes.pdf, pages 18–20
  checked_on: '2026-08-17'
hands_on:
  min: 15
  max: 15
unattended:
  min: 240
  max: 240
ingredient_sections:
- name: Slow Cooker Chili
  ingredients:
  - source_text: 2 pounds chicken breast
    grocery_item:
      key: chicken-breast
      name: Chicken Breast
      store_section:
        key: meat
        name: Meat
      shopping_mode: measured
    quantity:
      kind: exact
      amount: '2'
      unit: lb
  - source_text: 2 cans Rotel (undrained)
    grocery_item:
      key: rotel-tomatoes-and-chiles
      name: Rotel Tomatoes and Green Chiles
      store_section:
        key: pantry
        name: Pantry
      shopping_mode: counted
    quantity:
      kind: exact
      amount: '2'
      package:
        type: can
        size: '10'
        unit: oz
    preparation: undrained
  - source_text: 2 cans Corn (undrained)
    grocery_item:
      key: canned-corn
      name: Canned Corn
      store_section:
        key: pantry
        name: Pantry
      shopping_mode: counted
    quantity:
      kind: exact
      amount: '2'
      package:
        type: can
        size: '15.25'
        unit: oz
    preparation: undrained
  - source_text: 2 cans pinto beans (drained & rinsed)
    grocery_item:
      key: pinto-beans
      name: Pinto Beans
      store_section:
        key: pantry
        name: Pantry
      shopping_mode: counted
    quantity:
      kind: exact
      amount: '2'
      package:
        type: can
        size: '15'
        unit: oz
    preparation: drained and rinsed
  - source_text: 2 cans black beans (drained & rinsed)
    grocery_item:
      key: black-beans
      name: Black Beans
      store_section:
        key: pantry
        name: Pantry
      shopping_mode: counted
    quantity:
      kind: exact
      amount: '2'
      package:
        type: can
        size: '15'
        unit: oz
    preparation: drained and rinsed
  - source_text: 1 large onion
    grocery_item:
      key: onion
      name: Onion
      store_section:
        key: produce
        name: Produce
      shopping_mode: counted
    quantity:
      kind: exact
      amount: '1'
      unit: each
    preparation: large; chopped
  - source_text: 2 cups or chicken broth
    grocery_item:
      key: chicken-broth
      name: Chicken Broth
      store_section:
        key: pantry
        name: Pantry
      shopping_mode: measured
    quantity:
      kind: exact
      amount: '2'
      unit: cup
  - source_text: 2 tbsp of chili powder
    grocery_item:
      key: chili-powder
      name: Chili Powder
      store_section:
        key: spices
        name: Spices
      shopping_mode: measured
    quantity:
      kind: exact
      amount: '2'
      unit: tbsp
  - source_text: 1 tbsp of cumin
    grocery_item:
      key: ground-cumin
      name: Ground Cumin
      store_section:
        key: spices
        name: Spices
      shopping_mode: measured
    quantity:
      kind: exact
      amount: '1'
      unit: tbsp
  - source_text: 2 tsp of salt
    grocery_item:
      key: salt
      name: Salt
      store_section:
        key: spices
        name: Spices
      shopping_mode: measured
    quantity:
      kind: exact
      amount: '2'
      unit: tsp
  - source_text: 1 tsp of pepper
    grocery_item:
      key: black-pepper
      name: Black Pepper
      store_section:
        key: spices
        name: Spices
      shopping_mode: measured
    quantity:
      kind: exact
      amount: '1'
      unit: tsp
  - source_text: 2 packets of ranch seasoning
    grocery_item:
      key: ranch-seasoning
      name: Ranch Seasoning
      store_section:
        key: spices
        name: Spices
      shopping_mode: counted
    quantity:
      kind: exact
      amount: '2'
      package:
        type: packet
        size: '1'
        unit: oz
  - source_text: 16 ounces of cream cheese (2 bricks)
    grocery_item:
      key: cream-cheese
      name: Cream Cheese
      store_section:
        key: dairy
        name: Dairy
      shopping_mode: counted
    quantity:
      kind: exact
      amount: '2'
      package:
        type: block
        size: '8'
        unit: oz
instruction_sections:
- name: Method
  steps:
  - Place the chicken, undrained Rotel, undrained corn, drained pinto and black beans, onion, chicken broth, chili powder, cumin, salt, black pepper, ranch seasoning, and cream cheese in a slow cooker.
  - Cover and cook on high for 4 hours.
  - Remove the chicken, shred it, return it to the slow cooker, and stir until the cream cheese is incorporated before serving.
review:
- field: ingredient_sections[0].ingredients[6]
  kind: conflict-resolved
  note: Corrected the source typo “2 cups or chicken broth” to 2 cups of chicken broth.
  approved: true
- field: package_sizes
  kind: backfilled
  note: Household re-review approved common 10-ounce Rotel, 15.25-ounce corn, 15-ounce bean, and 1-ounce ranch packet sizes.
  approved: true
- field: instruction_sections
  kind: conflict-resolved
  note: Selected the source-listed high-for-four-hours path as the concrete v1 method; the low-for-eight-hours alternative remains in source evidence.
  approved: true
---

# Chicken Chili

> Approved bootstrap recipe. YAML front matter is ingested into SQLite; the sections below
> are the checked human-readable view.

## Recipe details

- Source: Help with House Plants Facebook post captured in Recipes.pdf, pages 18–20 (`source`)
- Source checked: 2026-08-17
- Yield: unknown
- Hands-on: 15 minutes
- Unattended: 240 minutes

## Ingredients

### Slow Cooker Chili

- 2 pounds chicken breast
  - Shopping: 2 lb Chicken Breast — Meat
- 2 cans Rotel (undrained)
  - Shopping: 2 × 10 oz cans Rotel Tomatoes and Green Chiles — Pantry
  - Preparation: undrained
- 2 cans Corn (undrained)
  - Shopping: 2 × 15.25 oz cans Canned Corn — Pantry
  - Preparation: undrained
- 2 cans pinto beans (drained & rinsed)
  - Shopping: 2 × 15 oz cans Pinto Beans — Pantry
  - Preparation: drained and rinsed
- 2 cans black beans (drained & rinsed)
  - Shopping: 2 × 15 oz cans Black Beans — Pantry
  - Preparation: drained and rinsed
- 1 large onion
  - Shopping: 1 Onion — Produce
  - Preparation: large; chopped
- 2 cups or chicken broth
  - Shopping: 2 cup Chicken Broth — Pantry
- 2 tbsp of chili powder
  - Shopping: 2 tbsp Chili Powder — Spices
- 1 tbsp of cumin
  - Shopping: 1 tbsp Ground Cumin — Spices
- 2 tsp of salt
  - Shopping: 2 tsp Salt — Spices
- 1 tsp of pepper
  - Shopping: 1 tsp Black Pepper — Spices
- 2 packets of ranch seasoning
  - Shopping: 2 × 1 oz packets Ranch Seasoning — Spices
- 16 ounces of cream cheese (2 bricks)
  - Shopping: 2 × 8 oz blocks Cream Cheese — Dairy

## Instructions

### Method

1. Place the chicken, undrained Rotel, undrained corn, drained pinto and black beans, onion, chicken broth, chili powder, cumin, salt, black pepper, ranch seasoning, and cream cheese in a slow cooker.
2. Cover and cook on high for 4 hours.
3. Remove the chicken, shred it, return it to the slow cooker, and stir until the cream cheese is incorporated before serving.

## One-batch grocery preview

### Dairy

- 2 × 8 oz blocks Cream Cheese

### Meat

- 2 lb Chicken Breast

### Pantry

- 2 × 10 oz cans Rotel Tomatoes and Green Chiles
- 2 × 15.25 oz cans Canned Corn
- 2 × 15 oz cans Pinto Beans
- 2 × 15 oz cans Black Beans
- 2 cup Chicken Broth

### Produce

- 1 Onion

### Spices

- 2 tbsp Chili Powder
- 1 tbsp Ground Cumin
- 2 tsp Salt
- 1 tsp Black Pepper
- 2 × 1 oz packets Ranch Seasoning

## Approved true-up decisions

- `ingredient_sections[0].ingredients[6]` — **conflict-resolved:** Corrected the source typo “2 cups or chicken broth” to 2 cups of chicken broth.
- `package_sizes` — **backfilled:** Household re-review approved common 10-ounce Rotel, 15.25-ounce corn, 15-ounce bean, and 1-ounce ranch packet sizes.
- `instruction_sections` — **conflict-resolved:** Selected the source-listed high-for-four-hours path as the concrete v1 method; the low-for-eight-hours alternative remains in source evidence.
