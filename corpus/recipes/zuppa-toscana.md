---
format_version: 1
key: zuppa-toscana
name: Zuppa Toscana
status: verified
approved_on: '2026-08-17'
source:
  relationship: source
  attribution: Temporary Manus page captured in Recipes.pdf, pages 20–22
  checked_on: '2026-08-17'
yield: 6 servings
hands_on:
  min: 20
  max: 20
unattended:
  min: 40
  max: 40
ingredient_sections:
- name: Soup
  ingredients:
  - source_text: 1/2 pound ground mild Italian sausage
    grocery_item:
      key: ground-mild-italian-sausage
      name: Ground Mild Italian Sausage
      store_section:
        key: meat
        name: Meat
      shopping_mode: measured
    quantity:
      kind: exact
      amount: 1/2
      unit: lb
  - source_text: 1 medium onion, chopped
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
    preparation: medium; chopped
  - source_text: 6 cloves garlic, minced
    grocery_item:
      key: garlic
      name: Garlic
      store_section:
        key: produce
        name: Produce
      shopping_mode: counted
    quantity:
      kind: exact
      amount: '6'
      unit: clove
    preparation: minced
  - source_text: 1 teaspoon dried basil
    grocery_item:
      key: dried-basil
      name: Dried Basil
      store_section:
        key: spices
        name: Spices
      shopping_mode: measured
    quantity:
      kind: exact
      amount: '1'
      unit: tsp
  - source_text: 1 teaspoon dried oregano
    grocery_item:
      key: dried-oregano
      name: Dried Oregano
      store_section:
        key: spices
        name: Spices
      shopping_mode: measured
    quantity:
      kind: exact
      amount: '1'
      unit: tsp
  - source_text: 1 teaspoon dried thyme
    grocery_item:
      key: dried-thyme
      name: Dried Thyme
      store_section:
        key: spices
        name: Spices
      shopping_mode: measured
    quantity:
      kind: exact
      amount: '1'
      unit: tsp
  - source_text: 4-5 medium Russet potatoes, peeled, thinly sliced and quartered
    grocery_item:
      key: russet-potatoes
      name: Russet Potatoes
      store_section:
        key: produce
        name: Produce
      shopping_mode: counted
    quantity:
      kind: range
      amount: '4'
      maximum: '5'
      unit: each
    preparation: medium; peeled, thinly sliced, and quartered
  - source_text: 4 cups chicken broth
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
  - source_text: 1 can coconut milk (about 2 cups)
    grocery_item:
      key: coconut-milk
      name: Coconut Milk
      store_section:
        key: pantry
        name: Pantry
      shopping_mode: counted
    quantity:
      kind: exact
      amount: '1'
      package:
        type: can
    note: source says about 2 cups
  - source_text: 1/2 teaspoon salt
    grocery_item:
      key: salt
      name: Salt
      store_section:
        key: spices
        name: Spices
      shopping_mode: measured
    quantity:
      kind: exact
      amount: 1/2
      unit: tsp
  - source_text: 3 cups chopped kale
    grocery_item:
      key: kale
      name: Kale
      store_section:
        key: produce
        name: Produce
      shopping_mode: measured
    quantity:
      kind: exact
      amount: '3'
      unit: cup
    preparation: chopped
  - source_text: 1/2 teaspoon crushed red pepper (optional)
    grocery_item:
      key: crushed-red-pepper
      name: Crushed Red Pepper
      store_section:
        key: spices
        name: Spices
      shopping_mode: measured
    quantity:
      kind: exact
      amount: 1/2
      unit: tsp
    optional: true
  - source_text: 2 tablespoons cornstarch
    grocery_item:
      key: cornstarch
      name: Cornstarch
      store_section:
        key: baking
        name: Baking
      shopping_mode: measured
    quantity:
      kind: exact
      amount: '2'
      unit: tbsp
  - source_text: 2 tablespoons water
    grocery_item:
      key: water
      name: Water
      store_section:
        key: non-shopping
        name: Non-shopping
      shopping_mode: measured
    quantity:
      kind: exact
      amount: '2'
      unit: tbsp
    non_shopping: true
instruction_sections:
- name: Method
  steps:
  - Brown the sausage in a large soup pot over medium-high heat, breaking it up, then drain excess fat.
  - Add the onion, garlic, basil, oregano, thyme, and optional crushed red pepper. Cook until the onion is soft and translucent.
  - Add the potatoes, chicken broth, coconut milk, and salt. Simmer for 20 to 25 minutes, until the potatoes are soft.
  - Stir the cornstarch and water into a smooth slurry, add it to the soup, and stir.
  - Add the kale and cook for 15 minutes more, until tender and the soup is slightly thickened.
review:
- field: ingredient_sections[0].ingredients[12:14]
  kind: backfilled
  note: Household re-review approved 2 tablespoons cornstarch and 2 tablespoons non-shopping water for the slurry required by the captured method.
  approved: true
- field: yield_and_time
  kind: backfilled
  note: Estimated six servings, 20 minutes hands-on, and 40 minutes unattended from the captured quantities and timed method.
  approved: true
- field: instruction_sections
  kind: rewritten
  note: Preserved the captured method and made the backfilled slurry quantities explicit.
  approved: true
---

# Zuppa Toscana

> Approved bootstrap recipe. YAML front matter is ingested into SQLite; the sections below
> are the checked human-readable view.

## Recipe details

- Source: Temporary Manus page captured in Recipes.pdf, pages 20–22 (`source`)
- Source checked: 2026-08-17
- Yield: 6 servings
- Hands-on: 20 minutes
- Unattended: 40 minutes

## Ingredients

### Soup

- 1/2 pound ground mild Italian sausage
  - Shopping: 1/2 lb Ground Mild Italian Sausage — Meat
- 1 medium onion, chopped
  - Shopping: 1 Onion — Produce
  - Preparation: medium; chopped
- 6 cloves garlic, minced
  - Shopping: 6 clove Garlic — Produce
  - Preparation: minced
- 1 teaspoon dried basil
  - Shopping: 1 tsp Dried Basil — Spices
- 1 teaspoon dried oregano
  - Shopping: 1 tsp Dried Oregano — Spices
- 1 teaspoon dried thyme
  - Shopping: 1 tsp Dried Thyme — Spices
- 4-5 medium Russet potatoes, peeled, thinly sliced and quartered
  - Shopping: 4–5 Russet Potatoes — Produce
  - Preparation: medium; peeled, thinly sliced, and quartered
- 4 cups chicken broth
  - Shopping: 4 cup Chicken Broth — Pantry
- 1 can coconut milk (about 2 cups)
  - Shopping: 1 can Coconut Milk — `source says about 2 cups` — Pantry
  - Note: source says about 2 cups
- 1/2 teaspoon salt
  - Shopping: 1/2 tsp Salt — Spices
- 3 cups chopped kale
  - Shopping: 3 cup Kale — Produce
  - Preparation: chopped
- 1/2 teaspoon crushed red pepper (optional)
  - Shopping: 1/2 tsp Crushed Red Pepper — optional — Spices
  - Optional: yes
- 2 tablespoons cornstarch
  - Shopping: 2 tbsp Cornstarch — Baking
- 2 tablespoons water
  - Recipe only: 2 tbsp Water — not added to the grocery list

## Instructions

### Method

1. Brown the sausage in a large soup pot over medium-high heat, breaking it up, then drain excess fat.
2. Add the onion, garlic, basil, oregano, thyme, and optional crushed red pepper. Cook until the onion is soft and translucent.
3. Add the potatoes, chicken broth, coconut milk, and salt. Simmer for 20 to 25 minutes, until the potatoes are soft.
4. Stir the cornstarch and water into a smooth slurry, add it to the soup, and stir.
5. Add the kale and cook for 15 minutes more, until tender and the soup is slightly thickened.

## One-batch grocery preview

### Baking

- 2 tbsp Cornstarch

### Meat

- 1/2 lb Ground Mild Italian Sausage

### Pantry

- 4 cup Chicken Broth
- 1 can Coconut Milk — `source says about 2 cups`

### Produce

- 1 Onion
- 6 clove Garlic
- 4–5 Russet Potatoes
- 3 cup Kale

### Spices

- 1 tsp Dried Basil
- 1 tsp Dried Oregano
- 1 tsp Dried Thyme
- 1/2 tsp Salt
- 1/2 tsp Crushed Red Pepper — optional

## Approved true-up decisions

- `ingredient_sections[0].ingredients[12:14]` — **backfilled:** Household re-review approved 2 tablespoons cornstarch and 2 tablespoons non-shopping water for the slurry required by the captured method.
- `yield_and_time` — **backfilled:** Estimated six servings, 20 minutes hands-on, and 40 minutes unattended from the captured quantities and timed method.
- `instruction_sections` — **rewritten:** Preserved the captured method and made the backfilled slurry quantities explicit.
