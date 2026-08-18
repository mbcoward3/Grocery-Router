---
format_version: 1
key: cheesy-pasta
name: Cheesy Pasta
status: verified
approved_on: 2026-08-17
source:
  relationship: source
  attribution: Household notes in Recipes.pdf, page 10
  checked_on: 2026-08-17
yield: 6 servings
hands_on:
  min: 25
  max: 25
unattended:
  min: 20
  max: 25
ingredient_sections:
  - name: Pasta Bake
    ingredients:
      - source_text: 1 lb Ground beef
        grocery_item:
          key: ground-beef
          name: Ground Beef
          store_section: {key: meat, name: Meat}
          shopping_mode: measured
        quantity: {kind: exact, amount: "1", unit: lb}
      - source_text: Elbow noodles box
        grocery_item:
          key: elbow-pasta
          name: Elbow Pasta
          store_section: {key: pantry, name: Pantry}
          shopping_mode: counted
        quantity:
          kind: exact
          amount: "1"
          package: {type: box, size: "16", unit: oz}
      - source_text: water for boiling the pasta
        grocery_item:
          key: water
          name: Water
          store_section: {key: non-shopping, name: Non-shopping}
          shopping_mode: measured
        quantity: {kind: unspecified}
        non_shopping: true
      - source_text: Cream cheese block
        grocery_item:
          key: cream-cheese
          name: Cream Cheese
          store_section: {key: dairy, name: Dairy}
          shopping_mode: counted
        quantity:
          kind: exact
          amount: "1"
          package: {type: block, size: "8", unit: oz}
      - source_text: Marinara sauce
        grocery_item:
          key: marinara-sauce
          name: Marinara Sauce
          store_section: {key: pantry, name: Pantry}
          shopping_mode: counted
        quantity:
          kind: exact
          amount: "1"
          package: {type: jar, size: "24", unit: oz}
      - source_text: Italian seasoning
        grocery_item:
          key: italian-seasoning
          name: Italian Seasoning
          store_section: {key: spices, name: Spices}
          shopping_mode: measured
        quantity: {kind: unspecified}
      - source_text: Salt
        grocery_item:
          key: salt
          name: Salt
          store_section: {key: spices, name: Spices}
          shopping_mode: measured
        quantity: {kind: unspecified}
      - source_text: Pepper
        grocery_item:
          key: black-pepper
          name: Black Pepper
          store_section: {key: spices, name: Spices}
          shopping_mode: measured
        quantity: {kind: unspecified}
      - source_text: Shredded cheese on top
        grocery_item:
          key: shredded-cheese
          name: Shredded Cheese
          store_section: {key: dairy, name: Dairy}
          shopping_mode: measured
        quantity: {kind: exact, amount: "2", unit: cup}
        preparation: for topping
instruction_sections:
  - name: Method
    steps:
      - Preheat the oven to 375°F.
      - Boil the elbow pasta in water until just al dente according to the package directions, then drain.
      - While the pasta cooks, brown the ground beef in a large skillet over medium-high heat, breaking it into crumbles. Drain excess fat.
      - Reduce the heat and stir the marinara, cream cheese, Italian seasoning, salt, and black pepper into the beef until the cream cheese melts and the sauce is smooth.
      - Combine the drained pasta with the beef sauce and transfer to a baking dish. Top evenly with the shredded cheese.
      - Bake uncovered for 20 to 25 minutes, until hot and bubbly and the cheese is melted.
review:
  - field: ingredient_sections[0].ingredients[1,3,4,8].quantity
    kind: backfilled
    note: Household review approved a 16-ounce pasta box, 8-ounce cream cheese block, 24-ounce marinara jar, and two cups shredded cheese.
    approved: true
  - field: ingredient_sections[0].ingredients[2]
    kind: backfilled
    note: Added the water required by the pasta package method and explicitly marked it non-shopping without inventing a quantity.
    approved: true
  - field: yield
    kind: backfilled
    note: Estimated six servings from one pound beef, one pound pasta, the approved sauce, and cheese quantities.
    approved: true
  - field: hands_on_and_unattended
    kind: backfilled
    note: Estimated 25 minutes hands-on and a 20-to-25-minute bake from the completed method.
    approved: true
  - field: instruction_sections
    kind: backfilled
    note: The household source has no method. Household review confirmed that the combined pasta is baked; the draft completes the stovetop preparation and uncovered bake without introducing food ingredients.
    approved: true
---

# Cheesy Pasta

> Approved bootstrap recipe. YAML front matter is ingested into SQLite; the sections below
> are the checked human-readable view.

## Recipe details

- Source: Household notes in Recipes.pdf, page 10 (`source`)
- Source checked: 2026-08-17
- Yield: 6 servings
- Hands-on: 25 minutes
- Unattended: 20–25 minutes

## Ingredients

### Pasta Bake

- 1 lb Ground beef
  - Shopping: 1 lb Ground Beef — Meat
- Elbow noodles box
  - Shopping: 1 × 16 oz box Elbow Pasta — Pantry
- water for boiling the pasta
  - Recipe only: Water — not added to the grocery list
- Cream cheese block
  - Shopping: 1 × 8 oz block Cream Cheese — Dairy
- Marinara sauce
  - Shopping: 1 × 24 oz jar Marinara Sauce — Pantry
- Italian seasoning
  - Shopping: Italian Seasoning — Spices
- Salt
  - Shopping: Salt — Spices
- Pepper
  - Shopping: Black Pepper — Spices
- Shredded cheese on top
  - Shopping: 2 cup Shredded Cheese — Dairy
  - Preparation: for topping

## Instructions

### Method

1. Preheat the oven to 375°F.
2. Boil the elbow pasta in water until just al dente according to the package directions, then drain.
3. While the pasta cooks, brown the ground beef in a large skillet over medium-high heat, breaking it into crumbles. Drain excess fat.
4. Reduce the heat and stir the marinara, cream cheese, Italian seasoning, salt, and black pepper into the beef until the cream cheese melts and the sauce is smooth.
5. Combine the drained pasta with the beef sauce and transfer to a baking dish. Top evenly with the shredded cheese.
6. Bake uncovered for 20 to 25 minutes, until hot and bubbly and the cheese is melted.

## One-batch grocery preview

### Dairy

- 1 × 8 oz block Cream Cheese
- 2 cup Shredded Cheese

### Meat

- 1 lb Ground Beef

### Pantry

- 1 × 16 oz box Elbow Pasta
- 1 × 24 oz jar Marinara Sauce

### Spices

- Italian Seasoning
- Salt
- Black Pepper

## Approved true-up decisions

- `ingredient_sections[0].ingredients[1,3,4,8].quantity` — **backfilled:** Household review approved a 16-ounce pasta box, 8-ounce cream cheese block, 24-ounce marinara jar, and two cups shredded cheese.
- `ingredient_sections[0].ingredients[2]` — **backfilled:** Added the water required by the pasta package method and explicitly marked it non-shopping without inventing a quantity.
- `yield` — **backfilled:** Estimated six servings from one pound beef, one pound pasta, the approved sauce, and cheese quantities.
- `hands_on_and_unattended` — **backfilled:** Estimated 25 minutes hands-on and a 20-to-25-minute bake from the completed method.
- `instruction_sections` — **backfilled:** The household source has no method. Household review confirmed that the combined pasta is baked; the draft completes the stovetop preparation and uncovered bake without introducing food ingredients.
