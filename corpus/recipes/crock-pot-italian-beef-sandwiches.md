---
format_version: 1
key: crock-pot-italian-beef-sandwiches
name: Crock Pot Italian Beef Sandwiches
status: verified
approved_on: 2026-08-17
source:
  relationship: source
  url: https://iowagirleats.com/crock-pot-italian-beef-sandwiches/
  attribution: Iowa Girl Eats — Crock Pot Italian Beef Sandwiches
  checked_on: 2026-08-17
yield: 8 servings
hands_on:
  min: 20
  max: 20
unattended:
  min: 600
  max: 600
ingredient_sections:
  - name: Italian Beef
    ingredients:
      - source_text: 3 lb chuck roast (trimmed of large pieces of fat then cut into large pieces)
        grocery_item:
          key: chuck-roast
          name: Chuck Roast
          store_section: {key: meat, name: Meat}
          shopping_mode: measured
        quantity: {kind: exact, amount: "3", unit: lb}
        preparation: trimmed of large pieces of fat and cut into large pieces
      - source_text: 1 envelope Italian salad dressing mix (see notes)
        grocery_item:
          key: italian-salad-dressing-mix
          name: Italian Salad Dressing Mix
          store_section: {key: spices, name: Spices}
          shopping_mode: counted
        quantity:
          kind: exact
          amount: "1"
          package: {type: envelope}
      - source_text: 8 oz pepperoncini pepper slices + splash of juice (plus extra peppers for serving)
        grocery_item:
          key: sliced-pepperoncini
          name: Sliced Pepperoncini
          store_section: {key: condiments, name: Condiments}
          shopping_mode: measured
        quantity: {kind: exact, amount: "8", unit: oz}
        preparation: with a splash of juice
      - source_text: 8 oz Chicago-Style Giardiniera (drained, plus extra for serving)
        grocery_item:
          key: chicago-style-giardiniera
          name: Chicago-Style Giardiniera
          store_section: {key: condiments, name: Condiments}
          shopping_mode: measured
        quantity: {kind: exact, amount: "8", unit: oz}
        preparation: drained
      - source_text: 14.5 oz can beef broth
        grocery_item:
          key: beef-broth
          name: Beef Broth
          store_section: {key: pantry, name: Pantry}
          shopping_mode: measured
        quantity:
          kind: exact
          amount: "1"
          package: {type: can, size: "14.5", unit: oz}
  - name: Sandwiches
    ingredients:
      - source_text: provolone cheese slices
        grocery_item:
          key: sliced-provolone-cheese
          name: Sliced Provolone Cheese
          store_section: {key: dairy, name: Dairy}
          shopping_mode: counted
        quantity: {kind: exact, amount: "8", unit: slice}
      - source_text: buns
        grocery_item:
          key: sandwich-buns
          name: Sandwich Buns
          store_section: {key: bakery, name: Bakery}
          shopping_mode: counted
        quantity:
          kind: exact
          amount: "1"
          package: {type: package, size: "8", unit: each}
      - source_text: extra peppers for serving
        grocery_item:
          key: sliced-pepperoncini
          name: Sliced Pepperoncini
          store_section: {key: condiments, name: Condiments}
          shopping_mode: measured
        quantity: {kind: unspecified}
        optional: true
      - source_text: extra Giardiniera for serving
        grocery_item:
          key: chicago-style-giardiniera
          name: Chicago-Style Giardiniera
          store_section: {key: condiments, name: Condiments}
          shopping_mode: measured
        quantity: {kind: unspecified}
        preparation: drained
        optional: true
instruction_sections:
  - name: Method
    steps:
      - Place the chuck roast in the bottom of a 6-quart slow cooker and sprinkle with the Italian salad dressing mix.
      - Add 8 ounces sliced pepperoncini with a splash of juice, 8 ounces drained giardiniera, and the beef broth. Lift the roast pieces so broth gets underneath them.
      - Cover and cook on low for 9 hours, or until the meat shreds easily with a fork.
      - Shred the meat, return it to the juices in the slow cooker, and cook on low for 1 more hour.
      - Split the buns and fill them with shredded beef. Add provolone and, if desired, extra pepperoncini and giardiniera, then serve.
review:
  - field: ingredient_sections[1].ingredients[0:2]
    kind: backfilled
    note: The source gives no sandwich component counts. Household review approved eight provolone slices and one 8-count bun package for its eight-serving yield.
    approved: true
  - field: ingredient_sections[1].ingredients[2:4]
    kind: rewritten
    note: Retained the source's unquantified extra pepperoncini and giardiniera as optional contributions instead of silently dropping them.
    approved: true
  - field: instruction_sections
    kind: rewritten
    note: Split the source's two long paragraphs into five ordered steps and selected standard buns rather than the source's gluten-free conditional.
    approved: true
---

# Crock Pot Italian Beef Sandwiches

> Approved bootstrap recipe. YAML front matter is ingested into SQLite; the sections below
> are the checked human-readable view.

## Recipe details

- Source: [Iowa Girl Eats — Crock Pot Italian Beef Sandwiches](https://iowagirleats.com/crock-pot-italian-beef-sandwiches/) (`source`)
- Source checked: 2026-08-17
- Yield: 8 servings
- Hands-on: 20 minutes
- Unattended: 600 minutes

## Ingredients

### Italian Beef

- 3 lb chuck roast (trimmed of large pieces of fat then cut into large pieces)
  - Shopping: 3 lb Chuck Roast — Meat
  - Preparation: trimmed of large pieces of fat and cut into large pieces
- 1 envelope Italian salad dressing mix (see notes)
  - Shopping: 1 envelope Italian Salad Dressing Mix — Spices
- 8 oz pepperoncini pepper slices + splash of juice (plus extra peppers for serving)
  - Shopping: 8 oz Sliced Pepperoncini — Condiments
  - Preparation: with a splash of juice
- 8 oz Chicago-Style Giardiniera (drained, plus extra for serving)
  - Shopping: 8 oz Chicago-Style Giardiniera — Condiments
  - Preparation: drained
- 14.5 oz can beef broth
  - Shopping: 1 × 14.5 oz can Beef Broth — Pantry

### Sandwiches

- provolone cheese slices
  - Shopping: 8 slices Sliced Provolone Cheese — Dairy
- buns
  - Shopping: 1 × 8-count package Sandwich Buns — Bakery
- extra peppers for serving
  - Shopping: Sliced Pepperoncini — optional — Condiments
  - Optional: yes
- extra Giardiniera for serving
  - Shopping: Chicago-Style Giardiniera — optional — Condiments
  - Preparation: drained
  - Optional: yes

## Instructions

### Method

1. Place the chuck roast in the bottom of a 6-quart slow cooker and sprinkle with the Italian salad dressing mix.
2. Add 8 ounces sliced pepperoncini with a splash of juice, 8 ounces drained giardiniera, and the beef broth. Lift the roast pieces so broth gets underneath them.
3. Cover and cook on low for 9 hours, or until the meat shreds easily with a fork.
4. Shred the meat, return it to the juices in the slow cooker, and cook on low for 1 more hour.
5. Split the buns and fill them with shredded beef. Add provolone and, if desired, extra pepperoncini and giardiniera, then serve.

## One-batch grocery preview

### Bakery

- 1 × 8-count package Sandwich Buns

### Condiments

- 8 oz Sliced Pepperoncini
- 8 oz Chicago-Style Giardiniera
- Sliced Pepperoncini — optional
- Chicago-Style Giardiniera — optional

### Dairy

- 8 slices Sliced Provolone Cheese

### Meat

- 3 lb Chuck Roast

### Pantry

- 1 × 14.5 oz can Beef Broth

### Spices

- 1 envelope Italian Salad Dressing Mix

## Approved true-up decisions

- `ingredient_sections[1].ingredients[0:2]` — **backfilled:** The source gives no sandwich component counts. Household review approved eight provolone slices and one 8-count bun package for its eight-serving yield.
- `ingredient_sections[1].ingredients[2:4]` — **rewritten:** Retained the source's unquantified extra pepperoncini and giardiniera as optional contributions instead of silently dropping them.
- `instruction_sections` — **rewritten:** Split the source's two long paragraphs into five ordered steps and selected standard buns rather than the source's gluten-free conditional.
