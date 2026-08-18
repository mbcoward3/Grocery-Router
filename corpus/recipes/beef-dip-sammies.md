---
format_version: 1
key: beef-dip-sammies
name: Beef Dip Sammies
status: verified
approved_on: '2026-08-17'
source:
  relationship: source
  attribution: Handwritten household recipe card photographed in Recipes.pdf, page 18
  checked_on: '2026-08-17'
hands_on:
  min: 25
  max: 25
unattended:
  min: 180
  max: 180
ingredient_sections:
- name: Beef Dip
  ingredients:
  - source_text: Chunk beef Roast
    grocery_item:
      key: chuck-roast
      name: Chuck Roast
      store_section:
        key: meat
        name: Meat
      shopping_mode: measured
    quantity:
      kind: exact
      amount: '3'
      unit: lb
  - source_text: Onion Soup Mix (Lipton's)
    grocery_item:
      key: onion-soup-mix
      name: Onion Soup Mix
      store_section:
        key: spices
        name: Spices
      shopping_mode: counted
    quantity:
      kind: exact
      amount: '1'
      package:
        type: packet
    note: 'source brand: Lipton''s'
  - source_text: Jar of Pepperocinis
    grocery_item:
      key: pepperoncini
      name: Pepperoncini
      store_section:
        key: condiments
        name: Condiments
      shopping_mode: counted
    quantity:
      kind: exact
      amount: '1'
      package:
        type: jar
  - source_text: water to reach two-thirds of the roast
    grocery_item:
      key: water
      name: Water
      store_section:
        key: non-shopping
        name: Non-shopping
      shopping_mode: measured
    quantity:
      kind: unspecified
    non_shopping: true
  - source_text: Buns
    grocery_item:
      key: sandwich-buns
      name: Sandwich Buns
      store_section:
        key: bakery
        name: Bakery
      shopping_mode: counted
    quantity:
      kind: exact
      amount: '1'
      package:
        type: package
  - source_text: oleo or butter
    grocery_item:
      key: butter
      name: Butter
      store_section:
        key: dairy
        name: Dairy
      shopping_mode: measured
    quantity:
      kind: unspecified
instruction_sections:
- name: Method
  steps:
  - Heat a large covered pan over high heat. Add the roast and turn it until deeply browned on all sides.
  - Add enough water to reach about two-thirds of the way up the roast. Sprinkle with the onion soup mix and pour in the liquid from the pepperoncini jar.
  - Bring to a boil, reduce the heat to low, cover, and simmer for about 3 hours, turning occasionally, until the beef is tender enough to pull apart. Add water if needed to keep the braising liquid from drying out.
  - Shred or thinly slice the beef and return it to the cooking liquid. Add pepperoncini as desired.
  - Spread butter on the cut sides of the buns and broil on a baking sheet until toasted, watching carefully. Fill with beef and serve with the cooking liquid for dipping.
review:
- field: method_and_quantities
  kind: conflict-resolved
  note: Under delegated household review, trusted the photographed stovetop braise over the old slow-cooker note and backfilled a 3-pound roast, one pepperoncini jar, one bun package, and a covered three-hour low simmer. Logged for revisit in trueup/CONTROVERSIAL_CALLS.md.
  approved: true
- field: ingredient_sections[0].ingredients[3]
  kind: backfilled
  note: Retained the card-directed water level as an unquantified non-shopping input.
  approved: true
- field: instruction_sections
  kind: backfilled
  note: Completed the cut-off card with a conservative covered braise to a shreddable endpoint and retained the captured broiled-bun method.
  approved: true
---

# Beef Dip Sammies

> Approved bootstrap recipe. YAML front matter is ingested into SQLite; the sections below
> are the checked human-readable view.

## Recipe details

- Source: Handwritten household recipe card photographed in Recipes.pdf, page 18 (`source`)
- Source checked: 2026-08-17
- Yield: unknown
- Hands-on: 25 minutes
- Unattended: 180 minutes

## Ingredients

### Beef Dip

- Chunk beef Roast
  - Shopping: 3 lb Chuck Roast — Meat
- Onion Soup Mix (Lipton's)
  - Shopping: 1 packet Onion Soup Mix — `source brand: Lipton's` — Spices
  - Note: source brand: Lipton's
- Jar of Pepperocinis
  - Shopping: 1 jar Pepperoncini — Condiments
- water to reach two-thirds of the roast
  - Recipe only: Water — not added to the grocery list
- Buns
  - Shopping: 1 package Sandwich Buns — Bakery
- oleo or butter
  - Shopping: Butter — Dairy

## Instructions

### Method

1. Heat a large covered pan over high heat. Add the roast and turn it until deeply browned on all sides.
2. Add enough water to reach about two-thirds of the way up the roast. Sprinkle with the onion soup mix and pour in the liquid from the pepperoncini jar.
3. Bring to a boil, reduce the heat to low, cover, and simmer for about 3 hours, turning occasionally, until the beef is tender enough to pull apart. Add water if needed to keep the braising liquid from drying out.
4. Shred or thinly slice the beef and return it to the cooking liquid. Add pepperoncini as desired.
5. Spread butter on the cut sides of the buns and broil on a baking sheet until toasted, watching carefully. Fill with beef and serve with the cooking liquid for dipping.

## One-batch grocery preview

### Bakery

- 1 package Sandwich Buns

### Condiments

- 1 jar Pepperoncini

### Dairy

- Butter

### Meat

- 3 lb Chuck Roast

### Spices

- 1 packet Onion Soup Mix — `source brand: Lipton's`

## Approved true-up decisions

- `method_and_quantities` — **conflict-resolved:** Under delegated household review, trusted the photographed stovetop braise over the old slow-cooker note and backfilled a 3-pound roast, one pepperoncini jar, one bun package, and a covered three-hour low simmer. Logged for revisit in trueup/CONTROVERSIAL_CALLS.md.
- `ingredient_sections[0].ingredients[3]` — **backfilled:** Retained the card-directed water level as an unquantified non-shopping input.
- `instruction_sections` — **backfilled:** Completed the cut-off card with a conservative covered braise to a shreddable endpoint and retained the captured broiled-bun method.
