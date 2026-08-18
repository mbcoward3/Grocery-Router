---
format_version: 1
key: three-ingredient-teriyaki-chicken
name: 3-Ingredient Teriyaki Chicken
status: verified
approved_on: 2026-08-17
source:
  relationship: source
  url: https://tasty.co/recipe/3-ingredient-teriyaki-chicken
  attribution: Tasty — 3-Ingredient Teriyaki Chicken
  checked_on: 2026-08-17
yield: 4 servings
hands_on:
  min: 30
  max: 30
unattended:
  min: 0
  max: 0
ingredient_sections:
  - name: Teriyaki Chicken
    ingredients:
      - source_text: 2 lb boneless, skinless chicken thighs, cubed
        grocery_item:
          key: boneless-skinless-chicken-thighs
          name: Boneless Skinless Chicken Thighs
          store_section: {key: meat, name: Meat}
          shopping_mode: measured
        quantity: {kind: exact, amount: "2", unit: lb}
        preparation: cubed
      - source_text: 1 cup soy sauce
        grocery_item:
          key: soy-sauce
          name: Soy Sauce
          store_section: {key: condiments, name: Condiments}
          shopping_mode: measured
        quantity: {kind: exact, amount: "1", unit: cup}
      - source_text: ½ cup brown sugar
        grocery_item:
          key: brown-sugar
          name: Brown Sugar
          store_section: {key: baking, name: Baking}
          shopping_mode: measured
        quantity: {kind: exact, amount: "1/2", unit: cup}
      - source_text: Serve with rice, if desired.
        grocery_item:
          key: white-rice
          name: White Rice
          store_section: {key: pantry, name: Pantry}
          shopping_mode: presence-only
        quantity: {kind: unspecified}
        preparation: cooked for serving
        optional: true
instruction_sections:
  - name: Method
    steps:
      - Heat a large nonstick pan over medium-high heat. Add the chicken and quickly sear until golden brown on both sides.
      - Add the soy sauce and brown sugar and stir to combine, then bring to a boil. Cook until the sauce reduces and coats the chicken.
      - Serve with rice, if desired.
review:
  - field: hands_on
    kind: rewritten
    note: The source reports 15 minutes prep and 15 minutes cook time; recorded all 30 minutes as hands-on because the short stovetop method requires attention.
    approved: true
  - field: ingredient_sections[0].ingredients[3]
    kind: backfilled
    note: Rice appears in the authoritative serving instruction rather than its three-ingredient list; retained it as an optional presence-only requirement.
    approved: true
  - field: instruction_sections
    kind: rewritten
    note: Preserved the three substantive source instructions, added the household-approved "quickly" clarification to the searing step, and omitted the non-action "Enjoy!" line.
    approved: true
---

# 3-Ingredient Teriyaki Chicken

> Approved bootstrap recipe. YAML front matter is ingested into SQLite; the sections below
> are the checked human-readable view.

## Recipe details

- Source: [Tasty — 3-Ingredient Teriyaki Chicken](https://tasty.co/recipe/3-ingredient-teriyaki-chicken) (`source`)
- Source checked: 2026-08-17
- Yield: 4 servings
- Hands-on: 30 minutes
- Unattended: 0 minutes

## Ingredients

### Teriyaki Chicken

- 2 lb boneless, skinless chicken thighs, cubed
  - Shopping: 2 lb Boneless Skinless Chicken Thighs — Meat
  - Preparation: cubed
- 1 cup soy sauce
  - Shopping: 1 cup Soy Sauce — Condiments
- ½ cup brown sugar
  - Shopping: 1/2 cup Brown Sugar — Baking
- Serve with rice, if desired.
  - Shopping: White Rice — optional — Pantry
  - Preparation: cooked for serving
  - Optional: yes

## Instructions

### Method

1. Heat a large nonstick pan over medium-high heat. Add the chicken and quickly sear until golden brown on both sides.
2. Add the soy sauce and brown sugar and stir to combine, then bring to a boil. Cook until the sauce reduces and coats the chicken.
3. Serve with rice, if desired.

## One-batch grocery preview

### Baking

- 1/2 cup Brown Sugar

### Condiments

- 1 cup Soy Sauce

### Meat

- 2 lb Boneless Skinless Chicken Thighs

### Pantry

- White Rice — optional

## Approved true-up decisions

- `hands_on` — **rewritten:** The source reports 15 minutes prep and 15 minutes cook time; recorded all 30 minutes as hands-on because the short stovetop method requires attention.
- `ingredient_sections[0].ingredients[3]` — **backfilled:** Rice appears in the authoritative serving instruction rather than its three-ingredient list; retained it as an optional presence-only requirement.
- `instruction_sections` — **rewritten:** Preserved the three substantive source instructions, added the household-approved "quickly" clarification to the searing step, and omitted the non-action "Enjoy!" line.
