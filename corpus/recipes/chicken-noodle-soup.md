---
format_version: 1
key: chicken-noodle-soup
name: Chicken Noodle Soup
status: verified
approved_on: 2026-08-17
source:
  relationship: source
  attribution: Household PDF capture of a temporary Manus page, Recipes.pdf page 8
  checked_on: 2026-08-17
yield: 6 servings
hands_on:
  min: 15
  max: 15
unattended:
  min: 15
  max: 20
ingredient_sections:
  - name: Soup
    ingredients:
      - source_text: 6 cups chicken broth
        grocery_item:
          key: chicken-broth
          name: Chicken Broth
          store_section: {key: pantry, name: Pantry}
          shopping_mode: measured
        quantity: {kind: exact, amount: "6", unit: cup}
      - source_text: 4 cups water
        grocery_item:
          key: water
          name: Water
          store_section: {key: non-shopping, name: Non-shopping}
          shopping_mode: measured
        quantity: {kind: exact, amount: "4", unit: cup}
        non_shopping: true
      - source_text: 1-2 carrots, peeled and thinly sliced
        grocery_item:
          key: carrots
          name: Carrots
          store_section: {key: produce, name: Produce}
          shopping_mode: counted
        quantity: {kind: range, amount: "1", maximum: "2", unit: each}
        preparation: peeled and thinly sliced
      - source_text: 1 rib of celery, thinly sliced
        grocery_item:
          key: celery
          name: Celery
          store_section: {key: produce, name: Produce}
          shopping_mode: counted
        quantity: {kind: exact, amount: "1", unit: each}
        preparation: rib; thinly sliced
      - source_text: 1/2 small onion, diced
        grocery_item:
          key: onion
          name: Onion
          store_section: {key: produce, name: Produce}
          shopping_mode: counted
        quantity: {kind: exact, amount: "1/2", unit: each}
        preparation: small; diced
      - source_text: 1-2 cloves garlic, minced
        grocery_item:
          key: garlic
          name: Garlic
          store_section: {key: produce, name: Produce}
          shopping_mode: counted
        quantity: {kind: range, amount: "1", maximum: "2", unit: clove}
        preparation: minced
      - source_text: 3 packages Maruchan Ramen noodles (do not use seasoning packet)
        grocery_item:
          key: maruchan-ramen-noodles
          name: Maruchan Ramen Noodles
          store_section: {key: pantry, name: Pantry}
          shopping_mode: counted
        quantity:
          kind: exact
          amount: "3"
          package: {type: package}
        preparation: discard seasoning packets
      - source_text: 2 cups cooked shredded chicken
        grocery_item:
          key: cooked-chicken
          name: Cooked Chicken
          store_section: {key: meat, name: Meat}
          shopping_mode: measured
        quantity: {kind: exact, amount: "2", unit: cup}
        preparation: shredded
        note: "suggestion: rotisserie chicken"
instruction_sections:
  - name: Method
    steps:
      - Add the chicken broth, water, carrots, celery, onion, and garlic to a large soup pot and bring to a boil over medium-high heat.
      - Reduce to a steady simmer and cook for 12 to 15 minutes, until the vegetables are tender.
      - Add the cooked shredded chicken and return the soup to a simmer.
      - Discard the ramen seasoning packets. Add the noodles and cook for about 3 minutes, or according to the package directions, until tender.
      - Remove from the heat, separate the noodles with a spoon, and serve.
review:
  - field: yield
    kind: backfilled
    note: Estimated six servings from ten cups of liquid, three ramen packages, vegetables, and the approved chicken quantity.
    approved: true
  - field: ingredient_sections[0].ingredients[7]
    kind: backfilled
    note: The captured ingredient list is cut off before its chicken line. Household review approved two cups cooked shredded chicken with a rotisserie chicken suggestion.
    approved: true
  - field: ingredient_sections[0].ingredients[1]
    kind: conflict-resolved
    note: Retained four cups water in the recipe while using the approved explicit non-shopping treatment for tap water.
    approved: true
  - field: hands_on_and_unattended
    kind: backfilled
    note: Estimated 15 minutes hands-on and 15 to 20 minutes unattended from the completed stovetop method because the temporary source supplied no timing.
    approved: true
  - field: instruction_sections
    kind: backfilled
    note: The temporary source screenshot contains no method. Added a minimal stovetop method that uses every visible ingredient, cooks the vegetables before the quick ramen noodles, and does not introduce additional ingredients.
    approved: true
---

# Chicken Noodle Soup

> Approved bootstrap recipe. YAML front matter is ingested into SQLite; the sections below
> are the checked human-readable view.

## Recipe details

- Source: Household PDF capture of a temporary Manus page, Recipes.pdf page 8 (`source`)
- Source checked: 2026-08-17
- Yield: 6 servings
- Hands-on: 15 minutes
- Unattended: 15–20 minutes

## Ingredients

### Soup

- 6 cups chicken broth
  - Shopping: 6 cup Chicken Broth — Pantry
- 4 cups water
  - Recipe only: 4 cup Water — not added to the grocery list
- 1-2 carrots, peeled and thinly sliced
  - Shopping: 1–2 Carrots — Produce
  - Preparation: peeled and thinly sliced
- 1 rib of celery, thinly sliced
  - Shopping: 1 Celery — Produce
  - Preparation: rib; thinly sliced
- 1/2 small onion, diced
  - Shopping: 1/2 Onion — Produce
  - Preparation: small; diced
- 1-2 cloves garlic, minced
  - Shopping: 1–2 clove Garlic — Produce
  - Preparation: minced
- 3 packages Maruchan Ramen noodles (do not use seasoning packet)
  - Shopping: 3 packages Maruchan Ramen Noodles — Pantry
  - Preparation: discard seasoning packets
- 2 cups cooked shredded chicken
  - Shopping: 2 cup Cooked Chicken — `suggestion: rotisserie chicken` — Meat
  - Preparation: shredded
  - Note: suggestion: rotisserie chicken

## Instructions

### Method

1. Add the chicken broth, water, carrots, celery, onion, and garlic to a large soup pot and bring to a boil over medium-high heat.
2. Reduce to a steady simmer and cook for 12 to 15 minutes, until the vegetables are tender.
3. Add the cooked shredded chicken and return the soup to a simmer.
4. Discard the ramen seasoning packets. Add the noodles and cook for about 3 minutes, or according to the package directions, until tender.
5. Remove from the heat, separate the noodles with a spoon, and serve.

## One-batch grocery preview

### Meat

- 2 cup Cooked Chicken — `suggestion: rotisserie chicken`

### Pantry

- 6 cup Chicken Broth
- 3 packages Maruchan Ramen Noodles

### Produce

- 1–2 Carrots
- 1 Celery
- 1/2 Onion
- 1–2 clove Garlic

## Approved true-up decisions

- `yield` — **backfilled:** Estimated six servings from ten cups of liquid, three ramen packages, vegetables, and the approved chicken quantity.
- `ingredient_sections[0].ingredients[7]` — **backfilled:** The captured ingredient list is cut off before its chicken line. Household review approved two cups cooked shredded chicken with a rotisserie chicken suggestion.
- `ingredient_sections[0].ingredients[1]` — **conflict-resolved:** Retained four cups water in the recipe while using the approved explicit non-shopping treatment for tap water.
- `hands_on_and_unattended` — **backfilled:** Estimated 15 minutes hands-on and 15 to 20 minutes unattended from the completed stovetop method because the temporary source supplied no timing.
- `instruction_sections` — **backfilled:** The temporary source screenshot contains no method. Added a minimal stovetop method that uses every visible ingredient, cooks the vegetables before the quick ramen noodles, and does not introduce additional ingredients.
