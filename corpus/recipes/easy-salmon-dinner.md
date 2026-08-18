---
format_version: 1
key: easy-salmon-dinner
name: Easy Salmon Dinner
status: verified
approved_on: 2026-08-17
source:
  relationship: source
  url: https://tasty.co/recipe/easy-salmon-dinner
  attribution: Tasty — Easy Salmon Dinner
  checked_on: 2026-08-17
yield: 2 servings
hands_on:
  min: 10
  max: 10
unattended:
  min: 34
  max: 34
ingredient_sections:
  - name: Salmon Dinner
    ingredients:
      - source_text: 1 lb potato
        grocery_item:
          key: potatoes
          name: Potatoes
          store_section: {key: produce, name: Produce}
          shopping_mode: measured
        quantity: {kind: exact, amount: "1", unit: lb}
      - source_text: olive oil, to taste
        grocery_item:
          key: olive-oil
          name: Olive Oil
          store_section: {key: pantry, name: Pantry}
          shopping_mode: measured
        quantity: {kind: unspecified}
      - source_text: salt, to taste
        grocery_item:
          key: salt
          name: Salt
          store_section: {key: spices, name: Spices}
          shopping_mode: measured
        quantity: {kind: unspecified}
      - source_text: pepper, to taste
        grocery_item:
          key: black-pepper
          name: Black Pepper
          store_section: {key: spices, name: Spices}
          shopping_mode: measured
        quantity: {kind: unspecified}
      - source_text: 3 tablespoons lemon juice
        grocery_item:
          key: lemon-juice
          name: Lemon Juice
          store_section: {key: condiments, name: Condiments}
          shopping_mode: measured
        quantity: {kind: exact, amount: "3", unit: tbsp}
      - source_text: 2 cloves garlic, minced
        grocery_item:
          key: garlic
          name: Garlic
          store_section: {key: produce, name: Produce}
          shopping_mode: counted
        quantity: {kind: exact, amount: "2", unit: clove}
        preparation: minced
      - source_text: ½ teaspoon onion powder
        grocery_item:
          key: onion-powder
          name: Onion Powder
          store_section: {key: spices, name: Spices}
          shopping_mode: measured
        quantity: {kind: exact, amount: "1/2", unit: tsp}
      - source_text: ½ teaspoon paprika
        grocery_item:
          key: paprika
          name: Paprika
          store_section: {key: spices, name: Spices}
          shopping_mode: measured
        quantity: {kind: exact, amount: "1/2", unit: tsp}
      - source_text: ½ teaspoon dried thyme
        grocery_item:
          key: dried-thyme
          name: Dried Thyme
          store_section: {key: spices, name: Spices}
          shopping_mode: measured
        quantity: {kind: exact, amount: "1/2", unit: tsp}
      - source_text: ½ teaspoon dried parsley
        grocery_item:
          key: dried-parsley
          name: Dried Parsley
          store_section: {key: spices, name: Spices}
          shopping_mode: measured
        quantity: {kind: exact, amount: "1/2", unit: tsp}
      - source_text: 2 tablespoons honey
        grocery_item:
          key: honey
          name: Honey
          store_section: {key: condiments, name: Condiments}
          shopping_mode: measured
        quantity: {kind: exact, amount: "2", unit: tbsp}
      - source_text: 2 salmon fillets
        grocery_item:
          key: salmon-fillets
          name: Salmon Fillets
          store_section: {key: seafood, name: Seafood}
          shopping_mode: counted
        quantity: {kind: exact, amount: "2", unit: each}
      - source_text: 1 bunch asparagus
        grocery_item:
          key: asparagus
          name: Asparagus
          store_section: {key: produce, name: Produce}
          shopping_mode: counted
        quantity: {kind: exact, amount: "1", unit: bunch}
      - source_text: 6 slices lemon
        grocery_item:
          key: lemon
          name: Lemon
          store_section: {key: produce, name: Produce}
          shopping_mode: counted
        quantity: {kind: exact, amount: "6", unit: slice}
        preparation: sliced for topping
      - source_text: 4 sprigs fresh thyme
        grocery_item:
          key: fresh-thyme
          name: Fresh Thyme
          store_section: {key: produce, name: Produce}
          shopping_mode: measured
        quantity: {kind: exact, amount: "4", unit: sprig}
        preparation: for topping
instruction_sections:
  - name: Method
    steps:
      - Preheat the oven to 400°F and line a baking sheet with parchment paper.
      - Add the potatoes to the baking sheet, season with olive oil, dried thyme, salt, and black pepper, and bake for 20 minutes.
      - Stir together the lemon juice, garlic, onion powder, paprika, dried thyme, dried parsley, and honey to make the salmon marinade.
      - Push the potatoes to one side of the baking sheet and add the salmon and asparagus.
      - Season the salmon and asparagus with olive oil, salt, and black pepper, then brush the marinade over the salmon.
      - Top the salmon with the lemon slices and fresh thyme sprigs.
      - Bake for 12 to 14 minutes, or until the salmon is cooked.
review:
  - field: hands_on_and_unattended
    kind: rewritten
    note: Preserved the source's published 10-minute prep and 34-minute cook timing as hands-on and unattended summary fields.
    approved: true
  - field: ingredient_sections[0].ingredients[11].grocery_item.store_section
    kind: backfilled
    note: Household review approved Seafood as the store section for salmon.
    approved: true
  - field: ingredient_sections[0].ingredients[13]
    kind: rewritten
    note: Preserved six lemon slices as a separate exact contribution from three tablespoons bottled lemon juice; no approximate lemon conversion was invented.
    approved: true
  - field: instruction_sections
    kind: rewritten
    note: Preserved the substantive source method in seven ordered steps and omitted the non-action "Enjoy!" line.
    approved: true
---

# Easy Salmon Dinner

> Approved bootstrap recipe. YAML front matter is ingested into SQLite; the sections below
> are the checked human-readable view.

## Recipe details

- Source: [Tasty — Easy Salmon Dinner](https://tasty.co/recipe/easy-salmon-dinner) (`source`)
- Source checked: 2026-08-17
- Yield: 2 servings
- Hands-on: 10 minutes
- Unattended: 34 minutes

## Ingredients

### Salmon Dinner

- 1 lb potato
  - Shopping: 1 lb Potatoes — Produce
- olive oil, to taste
  - Shopping: Olive Oil — Pantry
- salt, to taste
  - Shopping: Salt — Spices
- pepper, to taste
  - Shopping: Black Pepper — Spices
- 3 tablespoons lemon juice
  - Shopping: 3 tbsp Lemon Juice — Condiments
- 2 cloves garlic, minced
  - Shopping: 2 clove Garlic — Produce
  - Preparation: minced
- ½ teaspoon onion powder
  - Shopping: 1/2 tsp Onion Powder — Spices
- ½ teaspoon paprika
  - Shopping: 1/2 tsp Paprika — Spices
- ½ teaspoon dried thyme
  - Shopping: 1/2 tsp Dried Thyme — Spices
- ½ teaspoon dried parsley
  - Shopping: 1/2 tsp Dried Parsley — Spices
- 2 tablespoons honey
  - Shopping: 2 tbsp Honey — Condiments
- 2 salmon fillets
  - Shopping: 2 Salmon Fillets — Seafood
- 1 bunch asparagus
  - Shopping: 1 bunch Asparagus — Produce
- 6 slices lemon
  - Shopping: 6 slices Lemon — Produce
  - Preparation: sliced for topping
- 4 sprigs fresh thyme
  - Shopping: 4 sprig Fresh Thyme — Produce
  - Preparation: for topping

## Instructions

### Method

1. Preheat the oven to 400°F and line a baking sheet with parchment paper.
2. Add the potatoes to the baking sheet, season with olive oil, dried thyme, salt, and black pepper, and bake for 20 minutes.
3. Stir together the lemon juice, garlic, onion powder, paprika, dried thyme, dried parsley, and honey to make the salmon marinade.
4. Push the potatoes to one side of the baking sheet and add the salmon and asparagus.
5. Season the salmon and asparagus with olive oil, salt, and black pepper, then brush the marinade over the salmon.
6. Top the salmon with the lemon slices and fresh thyme sprigs.
7. Bake for 12 to 14 minutes, or until the salmon is cooked.

## One-batch grocery preview

### Condiments

- 3 tbsp Lemon Juice
- 2 tbsp Honey

### Pantry

- Olive Oil

### Produce

- 1 lb Potatoes
- 2 clove Garlic
- 1 bunch Asparagus
- 6 slices Lemon
- 4 sprig Fresh Thyme

### Seafood

- 2 Salmon Fillets

### Spices

- Salt
- Black Pepper
- 1/2 tsp Onion Powder
- 1/2 tsp Paprika
- 1/2 tsp Dried Thyme
- 1/2 tsp Dried Parsley

## Approved true-up decisions

- `hands_on_and_unattended` — **rewritten:** Preserved the source's published 10-minute prep and 34-minute cook timing as hands-on and unattended summary fields.
- `ingredient_sections[0].ingredients[11].grocery_item.store_section` — **backfilled:** Household review approved Seafood as the store section for salmon.
- `ingredient_sections[0].ingredients[13]` — **rewritten:** Preserved six lemon slices as a separate exact contribution from three tablespoons bottled lemon juice; no approximate lemon conversion was invented.
- `instruction_sections` — **rewritten:** Preserved the substantive source method in seven ordered steps and omitted the non-action "Enjoy!" line.
