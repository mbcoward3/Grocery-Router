---
format_version: 1
key: sausage-and-peppers
name: Sausage and Peppers
status: verified
approved_on: 2026-08-17
source:
  relationship: source
  url: https://chefjeanpierre.com/recipes/sausage-and-peppers/
  attribution: Chef Jean-Pierre — Sausage and Peppers Recipe
  checked_on: 2026-08-17
yield: 4 servings
hands_on:
  min: 25
  max: 30
unattended:
  min: 60
  max: 60
ingredient_sections:
  - name: Sausage and Peppers
    ingredients:
      - source_text: 2 large Italian Sausages
        grocery_item:
          key: italian-sausage
          name: Italian Sausage
          store_section: {key: meat, name: Meat}
          shopping_mode: counted
        quantity: {kind: exact, amount: "2", unit: each}
        preparation: large
      - source_text: Olive Oil, the Chef uses Garlic Olive Oil
        grocery_item:
          key: olive-oil
          name: Olive Oil
          store_section: {key: pantry, name: Pantry}
          shopping_mode: measured
        quantity: {kind: unspecified}
        note: "suggestion: garlic olive oil"
      - source_text: 1 cup Onion sliced
        grocery_item:
          key: onion
          name: Onion
          store_section: {key: produce, name: Produce}
          shopping_mode: counted
        quantity: {kind: exact, amount: "1", unit: cup}
        preparation: sliced
      - source_text: 1 Green Bell Pepper sliced
        grocery_item:
          key: green-bell-pepper
          name: Green Bell Pepper
          store_section: {key: produce, name: Produce}
          shopping_mode: counted
        quantity: {kind: exact, amount: "1", unit: each}
        preparation: sliced
      - source_text: 1 Red Bell Pepper sliced
        grocery_item:
          key: red-bell-pepper
          name: Red Bell Pepper
          store_section: {key: produce, name: Produce}
          shopping_mode: counted
        quantity: {kind: exact, amount: "1", unit: each}
        preparation: sliced
      - source_text: 2 teaspoons of Thyme & Rosemary freshly chopped, if dry use half
        grocery_item:
          key: fresh-thyme
          name: Fresh Thyme
          store_section: {key: produce, name: Produce}
          shopping_mode: measured
        quantity: {kind: exact, amount: "2", unit: tsp}
        preparation: finely chopped
      - source_text: 2 teaspoons of Thyme & Rosemary freshly chopped, if dry use half
        grocery_item:
          key: fresh-rosemary
          name: Fresh Rosemary
          store_section: {key: produce, name: Produce}
          shopping_mode: measured
        quantity: {kind: exact, amount: "2", unit: tsp}
        preparation: finely chopped
      - source_text: 1 tablespoon Garlic chopped
        grocery_item:
          key: garlic
          name: Garlic
          store_section: {key: produce, name: Produce}
          shopping_mode: counted
        quantity: {kind: exact, amount: "1", unit: tbsp}
        preparation: chopped
      - source_text: salt to taste
        grocery_item:
          key: salt
          name: Salt
          store_section: {key: spices, name: Spices}
          shopping_mode: measured
        quantity: {kind: unspecified}
      - source_text: pepper to taste
        grocery_item:
          key: black-pepper
          name: Black Pepper
          store_section: {key: spices, name: Spices}
          shopping_mode: measured
        quantity: {kind: unspecified}
instruction_sections:
  - name: Method
    steps:
      - Steam the sausages in a covered frying pan. Let them cool for at least 1 hour so the pork fat congeals and the juices are not lost when they are cut.
      - Cut the cooled sausages into bite-size pieces.
      - Heat olive oil in a frying pan to 365°F (185°C). Add the sliced onion and cook until light golden brown.
      - Add the sausage pieces. When they begin to brown, add the green and red peppers, fresh thyme, fresh rosemary, salt, and black pepper. Cook until the peppers are tender.
      - Add the chopped garlic and sauté for 1 minute, then remove from the heat.
review:
  - field: hands_on
    kind: backfilled
    note: The source gives no duration summary; estimated 25 to 30 minutes of hands-on work while preserving the separate cooling wait.
    approved: true
  - field: unattended
    kind: rewritten
    note: Recorded the source instruction to cool the sausage for at least one hour as 60 minutes; the instruction retains the lower-bound wording.
    approved: true
  - field: ingredient_sections[0].ingredients[5:7]
    kind: conflict-resolved
    note: The source says "2 teaspoons of Thyme & Rosemary" but does not establish whether that is two teaspoons combined or two teaspoons of each. Household review resolved it as two teaspoons of each.
    approved: true
  - field: ingredient_sections[0].ingredients[8:10]
    kind: backfilled
    note: Salt and pepper appear in the authoritative instructions but not the ingredient list; retained both as presence-only grocery requirements.
    approved: true
  - field: instruction_sections
    kind: rewritten
    note: Split the source's two long instruction paragraphs into five ordered steps without changing the method.
    approved: true
---

# Sausage and Peppers

> Approved bootstrap recipe. YAML front matter is ingested into SQLite; the sections below
> are the checked human-readable view.

## Recipe details

- Source: [Chef Jean-Pierre — Sausage and Peppers Recipe](https://chefjeanpierre.com/recipes/sausage-and-peppers/) (`source`)
- Source checked: 2026-08-17
- Yield: 4 servings
- Hands-on: 25–30 minutes
- Unattended: 60 minutes

## Ingredients

### Sausage and Peppers

- 2 large Italian Sausages
  - Shopping: 2 Italian Sausage — Meat
  - Preparation: large
- Olive Oil, the Chef uses Garlic Olive Oil
  - Shopping: Olive Oil — `suggestion: garlic olive oil` — Pantry
  - Note: suggestion: garlic olive oil
- 1 cup Onion sliced
  - Shopping: 1 cup Onion — Produce
  - Preparation: sliced
- 1 Green Bell Pepper sliced
  - Shopping: 1 Green Bell Pepper — Produce
  - Preparation: sliced
- 1 Red Bell Pepper sliced
  - Shopping: 1 Red Bell Pepper — Produce
  - Preparation: sliced
- 2 teaspoons of Thyme & Rosemary freshly chopped, if dry use half
  - Shopping: 2 tsp Fresh Thyme — Produce
  - Preparation: finely chopped
- 2 teaspoons of Thyme & Rosemary freshly chopped, if dry use half
  - Shopping: 2 tsp Fresh Rosemary — Produce
  - Preparation: finely chopped
- 1 tablespoon Garlic chopped
  - Shopping: 1 tbsp Garlic — Produce
  - Preparation: chopped
- salt to taste
  - Shopping: Salt — Spices
- pepper to taste
  - Shopping: Black Pepper — Spices

## Instructions

### Method

1. Steam the sausages in a covered frying pan. Let them cool for at least 1 hour so the pork fat congeals and the juices are not lost when they are cut.
2. Cut the cooled sausages into bite-size pieces.
3. Heat olive oil in a frying pan to 365°F (185°C). Add the sliced onion and cook until light golden brown.
4. Add the sausage pieces. When they begin to brown, add the green and red peppers, fresh thyme, fresh rosemary, salt, and black pepper. Cook until the peppers are tender.
5. Add the chopped garlic and sauté for 1 minute, then remove from the heat.

## One-batch grocery preview

### Meat

- 2 Italian Sausage

### Pantry

- Olive Oil — `suggestion: garlic olive oil`

### Produce

- 1 cup Onion
- 1 Green Bell Pepper
- 1 Red Bell Pepper
- 2 tsp Fresh Thyme
- 2 tsp Fresh Rosemary
- 1 tbsp Garlic

### Spices

- Salt
- Black Pepper

## Approved true-up decisions

- `hands_on` — **backfilled:** The source gives no duration summary; estimated 25 to 30 minutes of hands-on work while preserving the separate cooling wait.
- `unattended` — **rewritten:** Recorded the source instruction to cool the sausage for at least one hour as 60 minutes; the instruction retains the lower-bound wording.
- `ingredient_sections[0].ingredients[5:7]` — **conflict-resolved:** The source says "2 teaspoons of Thyme & Rosemary" but does not establish whether that is two teaspoons combined or two teaspoons of each. Household review resolved it as two teaspoons of each.
- `ingredient_sections[0].ingredients[8:10]` — **backfilled:** Salt and pepper appear in the authoritative instructions but not the ingredient list; retained both as presence-only grocery requirements.
- `instruction_sections` — **rewritten:** Split the source's two long instruction paragraphs into five ordered steps without changing the method.
