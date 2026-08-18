---
format_version: 1
key: chicken-and-biscuits-casserole
name: Chicken and Biscuits Casserole
status: verified
approved_on: 2026-08-17
source:
  relationship: source
  attribution: The Country Cook
  url: https://www.thecountrycook.net/chicken-and-biscuits-casserole/
  checked_on: 2026-08-17
yield: 6 servings
hands_on:
  min: 15
  max: 15
unattended:
  min: 35
  max: 45
ingredient_sections:
  - name: Ingredients
    ingredients:
      - source_text: 2 (10.5 ounce) cans cream of chicken soup
        grocery_item:
          key: cream-of-chicken-soup
          name: Cream of Chicken Soup
          store_section: {key: pantry, name: Pantry}
          shopping_mode: counted
        quantity:
          kind: exact
          amount: "2"
          package: {type: can, size: "10.5", unit: oz}
      - source_text: 1 cup milk
        grocery_item:
          key: milk
          name: Milk
          store_section: {key: dairy, name: Dairy}
          shopping_mode: measured
        quantity: {kind: exact, amount: "1", unit: cup}
      - source_text: 1 teaspoon garlic powder
        grocery_item:
          key: garlic-powder
          name: Garlic Powder
          store_section: {key: spices, name: Spices}
          shopping_mode: presence-only
        quantity: {kind: exact, amount: "1", unit: tsp}
      - source_text: 1/2 teaspoon rotisserie seasoning
        grocery_item:
          key: rotisserie-seasoning
          name: Rotisserie Seasoning
          store_section: {key: spices, name: Spices}
          shopping_mode: presence-only
        quantity: {kind: exact, amount: "1/2", unit: tsp}
      - source_text: 1/2 teaspoon black pepper
        grocery_item:
          key: black-pepper
          name: Black Pepper
          store_section: {key: spices, name: Spices}
          shopping_mode: presence-only
        quantity: {kind: exact, amount: "1/2", unit: tsp}
      - source_text: 12 ounce can refrigerated biscuits (or two smaller 6 ounce cans)
        grocery_item:
          key: refrigerated-biscuits
          name: Refrigerated Biscuits
          store_section: {key: dairy, name: Dairy}
          shopping_mode: counted
        quantity:
          kind: exact
          amount: "1"
          package: {type: can, size: "12", unit: oz}
        note: "v1 default: one 12 ounce can"
      - source_text: 1 cup frozen peas and carrots (allow to thaw slightly)
        grocery_item:
          key: frozen-peas-and-carrots
          name: Frozen Peas and Carrots
          store_section: {key: frozen, name: Frozen}
          shopping_mode: measured
        quantity: {kind: exact, amount: "1", unit: cup}
        preparation: allow to thaw slightly
      - source_text: 1 cup shredded cheddar cheese
        grocery_item:
          key: shredded-cheddar-cheese
          name: Shredded Cheddar Cheese
          store_section: {key: dairy, name: Dairy}
          shopping_mode: measured
        quantity: {kind: exact, amount: "1", unit: cup}
        preparation: shredded
      - source_text: 2 cups cooked chicken (shredded or diced)
        grocery_item:
          key: cooked-chicken
          name: Cooked Chicken
          store_section: {key: meat, name: Meat}
          shopping_mode: measured
        quantity: {kind: exact, amount: "2", unit: cup}
        preparation: shredded or diced
        note: "suggestion: rotisserie chicken"
      - source_text: 1/4 cup sliced green onion (optional)
        grocery_item:
          key: green-onion
          name: Green Onion
          store_section: {key: produce, name: Produce}
          shopping_mode: measured
        quantity: {kind: exact, amount: "1/4", unit: cup}
        preparation: sliced
        optional: true
instruction_sections:
  - name: Method
    steps:
      - Heat the oven to 375°F and coat a 9×13-inch baking dish with nonstick cooking spray.
      - Whisk the cream of chicken soup, milk, garlic powder, rotisserie seasoning, and black pepper until mostly smooth.
      - Separate the biscuits, cut each into quarters, and stir them into the soup mixture.
      - Fold in the peas and carrots, cheddar, and chicken, then spread the mixture in the prepared dish.
      - Bake uncovered on the middle rack for 35–45 minutes, until the biscuits are golden and the filling bubbles at the edges.
      - If the center needs more time, cover the top with foil and continue baking. Cool briefly, top with optional green onion, and serve.
review:
  - field: ingredient_sections[0].ingredients[5]
    kind: conflict-resolved
    note: Selected the source-listed 12 ounce biscuit can instead of two 6 ounce cans.
    approved: true
  - field: ingredient_sections[0].ingredients[8].note
    kind: rewritten
    note: Kept 2 cups cooked chicken and condensed the recommendation to suggestion colon rotisserie chicken.
    approved: true
  - field: instruction_sections
    kind: rewritten
    note: Condensed the source instructions without changing the method.
    approved: true
---

# Chicken and Biscuits Casserole

> Approved bootstrap recipe. YAML front matter is ingested into SQLite; the sections below
> are the checked human-readable view.

## Recipe details

- Source: [The Country Cook](https://www.thecountrycook.net/chicken-and-biscuits-casserole/) (`source`)
- Source checked: 2026-08-17
- Yield: 6 servings
- Hands-on: 15 minutes
- Unattended: 35–45 minutes

## Ingredients

### Ingredients

- 2 (10.5 ounce) cans cream of chicken soup
  - Shopping: 2 × 10.5 oz cans Cream of Chicken Soup — Pantry
- 1 cup milk
  - Shopping: 1 cup Milk — Dairy
- 1 teaspoon garlic powder
  - Shopping: Garlic Powder — Spices
- 1/2 teaspoon rotisserie seasoning
  - Shopping: Rotisserie Seasoning — Spices
- 1/2 teaspoon black pepper
  - Shopping: Black Pepper — Spices
- 12 ounce can refrigerated biscuits (or two smaller 6 ounce cans)
  - Shopping: 1 × 12 oz can Refrigerated Biscuits — `v1 default: one 12 ounce can` — Dairy
  - Note: v1 default: one 12 ounce can
- 1 cup frozen peas and carrots (allow to thaw slightly)
  - Shopping: 1 cup Frozen Peas and Carrots — Frozen
  - Preparation: allow to thaw slightly
- 1 cup shredded cheddar cheese
  - Shopping: 1 cup Shredded Cheddar Cheese — Dairy
  - Preparation: shredded
- 2 cups cooked chicken (shredded or diced)
  - Shopping: 2 cup Cooked Chicken — `suggestion: rotisserie chicken` — Meat
  - Preparation: shredded or diced
  - Note: suggestion: rotisserie chicken
- 1/4 cup sliced green onion (optional)
  - Shopping: 1/4 cup Green Onion — optional — Produce
  - Preparation: sliced
  - Optional: yes

## Instructions

### Method

1. Heat the oven to 375°F and coat a 9×13-inch baking dish with nonstick cooking spray.
2. Whisk the cream of chicken soup, milk, garlic powder, rotisserie seasoning, and black pepper until mostly smooth.
3. Separate the biscuits, cut each into quarters, and stir them into the soup mixture.
4. Fold in the peas and carrots, cheddar, and chicken, then spread the mixture in the prepared dish.
5. Bake uncovered on the middle rack for 35–45 minutes, until the biscuits are golden and the filling bubbles at the edges.
6. If the center needs more time, cover the top with foil and continue baking. Cool briefly, top with optional green onion, and serve.

## One-batch grocery preview

### Dairy

- 1 cup Milk
- 1 × 12 oz can Refrigerated Biscuits — `v1 default: one 12 ounce can`
- 1 cup Shredded Cheddar Cheese

### Frozen

- 1 cup Frozen Peas and Carrots

### Meat

- 2 cup Cooked Chicken — `suggestion: rotisserie chicken`

### Pantry

- 2 × 10.5 oz cans Cream of Chicken Soup

### Produce

- 1/4 cup Green Onion — optional

### Spices

- Garlic Powder
- Rotisserie Seasoning
- Black Pepper

## Approved true-up decisions

- `ingredient_sections[0].ingredients[5]` — **conflict-resolved:** Selected the source-listed 12 ounce biscuit can instead of two 6 ounce cans.
- `ingredient_sections[0].ingredients[8].note` — **rewritten:** Kept 2 cups cooked chicken and condensed the recommendation to suggestion colon rotisserie chicken.
- `instruction_sections` — **rewritten:** Condensed the source instructions without changing the method.
