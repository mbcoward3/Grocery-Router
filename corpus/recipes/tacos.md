---
format_version: 1
key: tacos
name: Tacos
status: verified
approved_on: 2026-08-17
source:
  relationship: source
  attribution: Household notes in Recipes.pdf, pages 9–10
  checked_on: 2026-08-17
hands_on:
  min: 20
  max: 20
unattended:
  min: 0
  max: 0
ingredient_sections:
  - name: Taco Meat
    ingredients:
      - source_text: 1 lb ground beef
        grocery_item:
          key: ground-beef
          name: Ground Beef
          store_section: {key: meat, name: Meat}
          shopping_mode: measured
        quantity: {kind: exact, amount: "1", unit: lb}
      - source_text: 1 oz pkt La Preferida Taco Seasoning
        grocery_item:
          key: la-preferida-taco-seasoning
          name: La Preferida Taco Seasoning
          store_section: {key: spices, name: Spices}
          shopping_mode: counted
        quantity:
          kind: exact
          amount: "1"
          package: {type: packet, size: "1", unit: oz}
      - source_text: ⅔ cup water
        grocery_item:
          key: water
          name: Water
          store_section: {key: non-shopping, name: Non-shopping}
          shopping_mode: measured
        quantity: {kind: exact, amount: "2/3", unit: cup}
        non_shopping: true
  - name: Tortillas and Toppings
    ingredients:
      - source_text: Flour tortillas
        grocery_item:
          key: flour-tortillas
          name: Flour Tortillas
          store_section: {key: bakery, name: Bakery}
          shopping_mode: counted
        quantity:
          kind: exact
          amount: "1"
          package: {type: package}
      - source_text: Shredded cheese
        grocery_item:
          key: shredded-cheese
          name: Shredded Cheese
          store_section: {key: dairy, name: Dairy}
          shopping_mode: presence-only
        quantity: {kind: unspecified}
      - source_text: Chopped onions
        grocery_item:
          key: onion
          name: Onion
          store_section: {key: produce, name: Produce}
          shopping_mode: counted
        quantity: {kind: unspecified}
        preparation: chopped
      - source_text: Sour cream
        grocery_item:
          key: sour-cream
          name: Sour Cream
          store_section: {key: dairy, name: Dairy}
          shopping_mode: presence-only
        quantity: {kind: unspecified}
      - source_text: Avocado
        grocery_item:
          key: avocado
          name: Avocado
          store_section: {key: produce, name: Produce}
          shopping_mode: counted
        quantity: {kind: unspecified}
        preparation: sliced or chopped for topping
      - source_text: El Paso mild taco sauce
        grocery_item:
          key: mild-taco-sauce
          name: Mild Taco Sauce
          store_section: {key: condiments, name: Condiments}
          shopping_mode: presence-only
        quantity: {kind: unspecified}
      - source_text: El Paso hot taco sauce
        grocery_item:
          key: hot-taco-sauce
          name: Hot Taco Sauce
          store_section: {key: condiments, name: Condiments}
          shopping_mode: presence-only
        quantity: {kind: unspecified}
instruction_sections:
  - name: Method
    steps:
      - Brown the ground beef in a skillet over medium-high heat, breaking it into crumbles, until cooked through. Drain excess fat.
      - Stir in the taco seasoning and ⅔ cup water. Cook, stirring, until the sauce thickens and coats the beef.
      - Warm the flour tortillas according to their package directions.
      - Serve the taco meat in warm tortillas with shredded cheese, chopped onion, sour cream, avocado, and mild or hot taco sauce.
review:
  - field: ingredient_sections[1].ingredients[0]
    kind: backfilled
    note: Household review selected one standard package of flour tortillas without imposing a package count or claiming a taco yield.
    approved: true
  - field: ingredient_sections[1].ingredients[1:7].quantity
    kind: backfilled
    note: Preserved the source's unquantified toppings and sauces as presence-only or unquantified counted requirements rather than inventing amounts.
    approved: true
  - field: ingredient_sections[1].ingredients[5:7].grocery_item
    kind: rewritten
    note: Preserved the household's Old El Paso source lines but mapped both purchased requirements to generic mild and hot taco sauce without brand restrictions.
    approved: true
  - field: ingredient_sections[0].ingredients[2]
    kind: conflict-resolved
    note: Retained the exact water contribution in the recipe and explicitly excluded it from grocery generation.
    approved: true
  - field: hands_on_and_unattended
    kind: backfilled
    note: Estimated 20 minutes hands-on and no unattended time from the completed skillet method.
    approved: true
  - field: instruction_sections
    kind: backfilled
    note: The household source contains only an ingredient list. Added a minimal taco-seasoning-packet method that uses every source contribution without adding ingredients.
    approved: true
---

# Tacos

> Approved bootstrap recipe. YAML front matter is ingested into SQLite; the sections below
> are the checked human-readable view.

## Recipe details

- Source: Household notes in Recipes.pdf, pages 9–10 (`source`)
- Source checked: 2026-08-17
- Yield: unknown
- Hands-on: 20 minutes
- Unattended: 0 minutes

## Ingredients

### Taco Meat

- 1 lb ground beef
  - Shopping: 1 lb Ground Beef — Meat
- 1 oz pkt La Preferida Taco Seasoning
  - Shopping: 1 × 1 oz packet La Preferida Taco Seasoning — Spices
- ⅔ cup water
  - Recipe only: 2/3 cup Water — not added to the grocery list

### Tortillas and Toppings

- Flour tortillas
  - Shopping: 1 package Flour Tortillas — Bakery
- Shredded cheese
  - Shopping: Shredded Cheese — Dairy
- Chopped onions
  - Shopping: Onion — Produce
  - Preparation: chopped
- Sour cream
  - Shopping: Sour Cream — Dairy
- Avocado
  - Shopping: Avocado — Produce
  - Preparation: sliced or chopped for topping
- El Paso mild taco sauce
  - Shopping: Mild Taco Sauce — Condiments
- El Paso hot taco sauce
  - Shopping: Hot Taco Sauce — Condiments

## Instructions

### Method

1. Brown the ground beef in a skillet over medium-high heat, breaking it into crumbles, until cooked through. Drain excess fat.
2. Stir in the taco seasoning and ⅔ cup water. Cook, stirring, until the sauce thickens and coats the beef.
3. Warm the flour tortillas according to their package directions.
4. Serve the taco meat in warm tortillas with shredded cheese, chopped onion, sour cream, avocado, and mild or hot taco sauce.

## One-batch grocery preview

### Bakery

- 1 package Flour Tortillas

### Condiments

- Mild Taco Sauce
- Hot Taco Sauce

### Dairy

- Shredded Cheese
- Sour Cream

### Meat

- 1 lb Ground Beef

### Produce

- Onion
- Avocado

### Spices

- 1 × 1 oz packet La Preferida Taco Seasoning

## Approved true-up decisions

- `ingredient_sections[1].ingredients[0]` — **backfilled:** Household review selected one standard package of flour tortillas without imposing a package count or claiming a taco yield.
- `ingredient_sections[1].ingredients[1:7].quantity` — **backfilled:** Preserved the source's unquantified toppings and sauces as presence-only or unquantified counted requirements rather than inventing amounts.
- `ingredient_sections[1].ingredients[5:7].grocery_item` — **rewritten:** Preserved the household's Old El Paso source lines but mapped both purchased requirements to generic mild and hot taco sauce without brand restrictions.
- `ingredient_sections[0].ingredients[2]` — **conflict-resolved:** Retained the exact water contribution in the recipe and explicitly excluded it from grocery generation.
- `hands_on_and_unattended` — **backfilled:** Estimated 20 minutes hands-on and no unattended time from the completed skillet method.
- `instruction_sections` — **backfilled:** The household source contains only an ingredient list. Added a minimal taco-seasoning-packet method that uses every source contribution without adding ingredients.
