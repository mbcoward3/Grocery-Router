---
format_version: 1
key: meatloaf
name: Meatloaf
status: verified
approved_on: 2026-08-17
source:
  relationship: source
  url: https://natashaskitchen.com/meatloaf-recipe/
  attribution: Natasha's Kitchen — Meatloaf Recipe
  checked_on: 2026-08-17
yield: 8 servings
hands_on:
  min: 15
  max: 15
unattended:
  min: 70
  max: 75
ingredient_sections:
  - name: Meatloaf
    ingredients:
      - source_text: 2 lbs ground beef (85% or 80% lean*)
        grocery_item:
          key: 85-percent-lean-ground-beef
          name: 85% Lean Ground Beef
          store_section: {key: meat, name: Meat}
          shopping_mode: measured
        quantity: {kind: exact, amount: "2", unit: lb}
        note: "suggestion: 80–90% lean works"
      - source_text: 1 med onion ((1 cup), finely chopped)
        grocery_item:
          key: onion
          name: Onion
          store_section: {key: produce, name: Produce}
          shopping_mode: counted
        quantity: {kind: exact, amount: "1", unit: each}
        preparation: medium; finely chopped, about 1 cup
      - source_text: 1 tsp olive oil
        grocery_item:
          key: olive-oil
          name: Olive Oil
          store_section: {key: pantry, name: Pantry}
          shopping_mode: measured
        quantity: {kind: exact, amount: "1", unit: tsp}
      - source_text: 2 large eggs
        grocery_item:
          key: eggs
          name: Eggs
          store_section: {key: dairy, name: Dairy}
          shopping_mode: counted
        quantity: {kind: exact, amount: "2", unit: each}
        preparation: large
      - source_text: 3 garlic cloves (minced)
        grocery_item:
          key: garlic
          name: Garlic
          store_section: {key: produce, name: Produce}
          shopping_mode: counted
        quantity: {kind: exact, amount: "3", unit: clove}
        preparation: minced
      - source_text: 2 Tbsp ketchup
        grocery_item:
          key: ketchup
          name: Ketchup
          store_section: {key: condiments, name: Condiments}
          shopping_mode: measured
        quantity: {kind: exact, amount: "2", unit: tbsp}
      - source_text: 3 Tbsp fresh parsley (finely chopped)
        grocery_item:
          key: fresh-parsley
          name: Fresh Parsley
          store_section: {key: produce, name: Produce}
          shopping_mode: measured
        quantity: {kind: exact, amount: "3", unit: tbsp}
        preparation: finely chopped
      - source_text: 3/4 cup Panko breadcrumbs (or gluten-free bread crumbs)
        grocery_item:
          key: panko-breadcrumbs
          name: Panko Breadcrumbs
          store_section: {key: baking, name: Baking}
          shopping_mode: measured
        quantity: {kind: exact, amount: "3/4", unit: cup}
      - source_text: 1/3 cup milk
        grocery_item:
          key: milk
          name: Milk
          store_section: {key: dairy, name: Dairy}
          shopping_mode: measured
        quantity: {kind: exact, amount: "1/3", unit: cup}
      - source_text: 1 tsp salt (or to taste)
        grocery_item:
          key: salt
          name: Salt
          store_section: {key: spices, name: Spices}
          shopping_mode: measured
        quantity: {kind: exact, amount: "1", unit: tsp}
      - source_text: 1 tsp Italian seasoning
        grocery_item:
          key: italian-seasoning
          name: Italian Seasoning
          store_section: {key: spices, name: Spices}
          shopping_mode: measured
        quantity: {kind: exact, amount: "1", unit: tsp}
      - source_text: 1/2 tsp ground black pepper
        grocery_item:
          key: black-pepper
          name: Black Pepper
          store_section: {key: spices, name: Spices}
          shopping_mode: measured
        quantity: {kind: exact, amount: "1/2", unit: tsp}
  - name: Sauce
    ingredients:
      - source_text: 3/4 cup ketchup
        grocery_item:
          key: ketchup
          name: Ketchup
          store_section: {key: condiments, name: Condiments}
          shopping_mode: measured
        quantity: {kind: exact, amount: "3/4", unit: cup}
      - source_text: 1 ½ tsp white vinegar
        grocery_item:
          key: white-vinegar
          name: White Vinegar
          store_section: {key: pantry, name: Pantry}
          shopping_mode: measured
        quantity: {kind: exact, amount: "1 1/2", unit: tsp}
      - source_text: 2 Tbsp brown sugar
        grocery_item:
          key: brown-sugar
          name: Brown Sugar
          store_section: {key: baking, name: Baking}
          shopping_mode: measured
        quantity: {kind: exact, amount: "2", unit: tbsp}
      - source_text: 1/2 tsp garlic powder
        grocery_item:
          key: garlic-powder
          name: Garlic Powder
          store_section: {key: spices, name: Spices}
          shopping_mode: measured
        quantity: {kind: exact, amount: "1/2", unit: tsp}
      - source_text: 1/2 tsp onion powder
        grocery_item:
          key: onion-powder
          name: Onion Powder
          store_section: {key: spices, name: Spices}
          shopping_mode: measured
        quantity: {kind: exact, amount: "1/2", unit: tsp}
instruction_sections:
  - name: Method
    steps:
      - Line a rimmed baking sheet with parchment paper or foil and preheat the oven to 350°F.
      - Heat the olive oil in a medium skillet over medium heat. Add the onion and sauté, stirring occasionally, until softened and golden, about 5 to 7 minutes. Transfer to a plate to cool.
      - In a large bowl, combine the ground beef, eggs, garlic, 2 tablespoons ketchup, parsley, panko, milk, salt, Italian seasoning, black pepper, and cooled onion. Mix just until combined.
      - Shape the mixture on the prepared pan into a loaf about 8 inches long, 4 inches wide, and 3 inches tall. Bake uncovered for 40 minutes.
      - Stir together the sauce ingredients in a small bowl.
      - Spread the sauce over the meatloaf. Bake for another 20 minutes, or until the center reaches 160°F. Rest for 10 to 15 minutes before slicing.
review:
  - field: ingredient_sections[0].ingredients[0]
    kind: conflict-resolved
    note: Household review selected 85% lean ground beef as the concrete default and approved a suggestion that 80–90% lean works.
    approved: true
  - field: ingredient_sections[0].ingredients[7]
    kind: conflict-resolved
    note: Selected the source's primary panko breadcrumb option rather than its gluten-free alternative.
    approved: true
  - field: hands_on_and_unattended
    kind: rewritten
    note: Preserved the source's 15-minute prep time; represented the explicit 60-minute bake plus 10-to-15-minute rest as 70 to 75 unattended minutes rather than silently omitting the rest.
    approved: true
  - field: instruction_sections
    kind: rewritten
    note: Retained all six authoritative steps while removing commentary that is not needed to execute the recipe.
    approved: true
---

# Meatloaf

> Approved bootstrap recipe. YAML front matter is ingested into SQLite; the sections below
> are the checked human-readable view.

## Recipe details

- Source: [Natasha's Kitchen — Meatloaf Recipe](https://natashaskitchen.com/meatloaf-recipe/) (`source`)
- Source checked: 2026-08-17
- Yield: 8 servings
- Hands-on: 15 minutes
- Unattended: 70–75 minutes

## Ingredients

### Meatloaf

- 2 lbs ground beef (85% or 80% lean*)
  - Shopping: 2 lb 85% Lean Ground Beef — `suggestion: 80–90% lean works` — Meat
  - Note: suggestion: 80–90% lean works
- 1 med onion ((1 cup), finely chopped)
  - Shopping: 1 Onion — Produce
  - Preparation: medium; finely chopped, about 1 cup
- 1 tsp olive oil
  - Shopping: 1 tsp Olive Oil — Pantry
- 2 large eggs
  - Shopping: 2 Eggs — Dairy
  - Preparation: large
- 3 garlic cloves (minced)
  - Shopping: 3 clove Garlic — Produce
  - Preparation: minced
- 2 Tbsp ketchup
  - Shopping: 2 tbsp Ketchup — Condiments
- 3 Tbsp fresh parsley (finely chopped)
  - Shopping: 3 tbsp Fresh Parsley — Produce
  - Preparation: finely chopped
- 3/4 cup Panko breadcrumbs (or gluten-free bread crumbs)
  - Shopping: 3/4 cup Panko Breadcrumbs — Baking
- 1/3 cup milk
  - Shopping: 1/3 cup Milk — Dairy
- 1 tsp salt (or to taste)
  - Shopping: 1 tsp Salt — Spices
- 1 tsp Italian seasoning
  - Shopping: 1 tsp Italian Seasoning — Spices
- 1/2 tsp ground black pepper
  - Shopping: 1/2 tsp Black Pepper — Spices

### Sauce

- 3/4 cup ketchup
  - Shopping: 3/4 cup Ketchup — Condiments
- 1 ½ tsp white vinegar
  - Shopping: 1 1/2 tsp White Vinegar — Pantry
- 2 Tbsp brown sugar
  - Shopping: 2 tbsp Brown Sugar — Baking
- 1/2 tsp garlic powder
  - Shopping: 1/2 tsp Garlic Powder — Spices
- 1/2 tsp onion powder
  - Shopping: 1/2 tsp Onion Powder — Spices

## Instructions

### Method

1. Line a rimmed baking sheet with parchment paper or foil and preheat the oven to 350°F.
2. Heat the olive oil in a medium skillet over medium heat. Add the onion and sauté, stirring occasionally, until softened and golden, about 5 to 7 minutes. Transfer to a plate to cool.
3. In a large bowl, combine the ground beef, eggs, garlic, 2 tablespoons ketchup, parsley, panko, milk, salt, Italian seasoning, black pepper, and cooled onion. Mix just until combined.
4. Shape the mixture on the prepared pan into a loaf about 8 inches long, 4 inches wide, and 3 inches tall. Bake uncovered for 40 minutes.
5. Stir together the sauce ingredients in a small bowl.
6. Spread the sauce over the meatloaf. Bake for another 20 minutes, or until the center reaches 160°F. Rest for 10 to 15 minutes before slicing.

## One-batch grocery preview

### Baking

- 3/4 cup Panko Breadcrumbs
- 2 tbsp Brown Sugar

### Condiments

- 2 tbsp Ketchup
- 3/4 cup Ketchup

### Dairy

- 2 Eggs
- 1/3 cup Milk

### Meat

- 2 lb 85% Lean Ground Beef — `suggestion: 80–90% lean works`

### Pantry

- 1 tsp Olive Oil
- 1 1/2 tsp White Vinegar

### Produce

- 1 Onion
- 3 clove Garlic
- 3 tbsp Fresh Parsley

### Spices

- 1 tsp Salt
- 1 tsp Italian Seasoning
- 1/2 tsp Black Pepper
- 1/2 tsp Garlic Powder
- 1/2 tsp Onion Powder

## Approved true-up decisions

- `ingredient_sections[0].ingredients[0]` — **conflict-resolved:** Household review selected 85% lean ground beef as the concrete default and approved a suggestion that 80–90% lean works.
- `ingredient_sections[0].ingredients[7]` — **conflict-resolved:** Selected the source's primary panko breadcrumb option rather than its gluten-free alternative.
- `hands_on_and_unattended` — **rewritten:** Preserved the source's 15-minute prep time; represented the explicit 60-minute bake plus 10-to-15-minute rest as 70 to 75 unattended minutes rather than silently omitting the rest.
- `instruction_sections` — **rewritten:** Retained all six authoritative steps while removing commentary that is not needed to execute the recipe.
