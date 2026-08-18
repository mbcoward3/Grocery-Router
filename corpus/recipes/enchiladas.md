---
format_version: 1
key: enchiladas
name: Enchiladas
status: verified
approved_on: 2026-08-17
source:
  relationship: source
  url: https://southernbite.com/5-ingredient-beef-enchiladas/
  attribution: Southern Bite — 5 Ingredient Beef Enchiladas
  checked_on: 2026-08-17
yield: 8 enchiladas
hands_on:
  min: 20
  max: 20
unattended:
  min: 35
  max: 35
ingredient_sections:
  - name: Enchiladas
    ingredients:
      - source_text: 1 pound lean ground beef
        grocery_item:
          key: lean-ground-beef
          name: Lean Ground Beef
          store_section: {key: meat, name: Meat}
          shopping_mode: measured
        quantity: {kind: exact, amount: "1", unit: lb}
      - source_text: 1 cup chunky salsa
        grocery_item:
          key: chunky-salsa
          name: Chunky Salsa
          store_section: {key: condiments, name: Condiments}
          shopping_mode: measured
        quantity: {kind: exact, amount: "1", unit: cup}
      - source_text: 1 (10-ounce) can red enchilada sauce
        grocery_item:
          key: red-enchilada-sauce
          name: Red Enchilada Sauce
          store_section: {key: pantry, name: Pantry}
          shopping_mode: counted
        quantity:
          kind: exact
          amount: "1"
          package: {type: can, size: "10", unit: oz}
      - source_text: 8 (8-inch) tortillas (We much prefer flour tortillas in this recipe, but corn are more traditional.)
        grocery_item:
          key: flour-tortillas
          name: Flour Tortillas
          store_section: {key: bakery, name: Bakery}
          shopping_mode: counted
        quantity: {kind: exact, amount: "8", unit: each}
        preparation: 8-inch
      - source_text: 1 (8-ounce) package Borden® Cheese Thick Cut Shredded Four Cheese Mexican (about 2 cups)
        grocery_item:
          key: shredded-mexican-cheese-blend
          name: Shredded Mexican Cheese Blend
          store_section: {key: dairy, name: Dairy}
          shopping_mode: counted
        quantity:
          kind: exact
          amount: "1"
          package: {type: package, size: "8", unit: oz}
        note: "source product: Borden Thick Cut Four Cheese Mexican"
      - source_text: nonstick cooking spray
        grocery_item:
          key: nonstick-cooking-spray
          name: Nonstick Cooking Spray
          store_section: {key: pantry, name: Pantry}
          shopping_mode: presence-only
        quantity: {kind: unspecified}
instruction_sections:
  - name: Method
    steps:
      - Preheat the oven to 350°F and lightly coat a 9×13-inch baking dish with nonstick cooking spray.
      - Brown the ground beef in a large skillet over medium-high heat. Drain the excess fat, return the skillet to medium-low heat, stir in the salsa, and cook until heated through.
      - Pour half of the enchilada sauce into the prepared baking dish and warm the tortillas according to their package directions.
      - Place about ¼ cup meat mixture and 1 heaping tablespoon cheese down the center of each tortilla. Roll tightly and arrange seam-side down in the dish.
      - Pour the remaining enchilada sauce over the tortillas and sprinkle with the remaining cheese. Cover tightly with aluminum foil and bake for 30 to 35 minutes.
review:
  - field: source
    kind: conflict-resolved
    note: Recovered the exact current source URL and matched the PDF capture against the live structured recipe, confirming that no ingredients were cut off above the screenshot.
    approved: true
  - field: ingredient_sections[0].ingredients[3]
    kind: conflict-resolved
    note: Selected flour tortillas as the concrete v1 requirement because the authoritative source explicitly says they are strongly preferred, while retaining the original alternative in source_text.
    approved: true
  - field: ingredient_sections[0].ingredients[4].grocery_item
    kind: rewritten
    note: Preserved the exact branded source line and package while mapping it to the generic purchased form Shredded Mexican Cheese Blend; retained the source product as a note.
    approved: true
  - field: ingredient_sections[0].ingredients[5]
    kind: backfilled
    note: Added nonstick cooking spray as an unquantified grocery requirement because it appears in the authoritative method but not the five-ingredient list.
    approved: true
  - field: instruction_sections
    kind: rewritten
    note: Preserved all five authoritative steps and exact bake range in a concise ordered method.
    approved: true
---

# Enchiladas

> Approved bootstrap recipe. YAML front matter is ingested into SQLite; the sections below
> are the checked human-readable view.

## Recipe details

- Source: [Southern Bite — 5 Ingredient Beef Enchiladas](https://southernbite.com/5-ingredient-beef-enchiladas/) (`source`)
- Source checked: 2026-08-17
- Yield: 8 enchiladas
- Hands-on: 20 minutes
- Unattended: 35 minutes

## Ingredients

### Enchiladas

- 1 pound lean ground beef
  - Shopping: 1 lb Lean Ground Beef — Meat
- 1 cup chunky salsa
  - Shopping: 1 cup Chunky Salsa — Condiments
- 1 (10-ounce) can red enchilada sauce
  - Shopping: 1 × 10 oz can Red Enchilada Sauce — Pantry
- 8 (8-inch) tortillas (We much prefer flour tortillas in this recipe, but corn are more traditional.)
  - Shopping: 8 Flour Tortillas — Bakery
  - Preparation: 8-inch
- 1 (8-ounce) package Borden® Cheese Thick Cut Shredded Four Cheese Mexican (about 2 cups)
  - Shopping: 1 × 8 oz package Shredded Mexican Cheese Blend — `source product: Borden Thick Cut Four Cheese Mexican` — Dairy
  - Note: source product: Borden Thick Cut Four Cheese Mexican
- nonstick cooking spray
  - Shopping: Nonstick Cooking Spray — Pantry

## Instructions

### Method

1. Preheat the oven to 350°F and lightly coat a 9×13-inch baking dish with nonstick cooking spray.
2. Brown the ground beef in a large skillet over medium-high heat. Drain the excess fat, return the skillet to medium-low heat, stir in the salsa, and cook until heated through.
3. Pour half of the enchilada sauce into the prepared baking dish and warm the tortillas according to their package directions.
4. Place about ¼ cup meat mixture and 1 heaping tablespoon cheese down the center of each tortilla. Roll tightly and arrange seam-side down in the dish.
5. Pour the remaining enchilada sauce over the tortillas and sprinkle with the remaining cheese. Cover tightly with aluminum foil and bake for 30 to 35 minutes.

## One-batch grocery preview

### Bakery

- 8 Flour Tortillas

### Condiments

- 1 cup Chunky Salsa

### Dairy

- 1 × 8 oz package Shredded Mexican Cheese Blend — `source product: Borden Thick Cut Four Cheese Mexican`

### Meat

- 1 lb Lean Ground Beef

### Pantry

- 1 × 10 oz can Red Enchilada Sauce
- Nonstick Cooking Spray

## Approved true-up decisions

- `source` — **conflict-resolved:** Recovered the exact current source URL and matched the PDF capture against the live structured recipe, confirming that no ingredients were cut off above the screenshot.
- `ingredient_sections[0].ingredients[3]` — **conflict-resolved:** Selected flour tortillas as the concrete v1 requirement because the authoritative source explicitly says they are strongly preferred, while retaining the original alternative in source_text.
- `ingredient_sections[0].ingredients[4].grocery_item` — **rewritten:** Preserved the exact branded source line and package while mapping it to the generic purchased form Shredded Mexican Cheese Blend; retained the source product as a note.
- `ingredient_sections[0].ingredients[5]` — **backfilled:** Added nonstick cooking spray as an unquantified grocery requirement because it appears in the authoritative method but not the five-ingredient list.
- `instruction_sections` — **rewritten:** Preserved all five authoritative steps and exact bake range in a concise ordered method.
