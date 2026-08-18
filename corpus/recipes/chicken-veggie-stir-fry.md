---
format_version: 1
key: chicken-veggie-stir-fry
name: Chicken Veggie Stir Fry
status: verified
approved_on: 2026-08-17
source:
  relationship: source
  url: https://tasty.co/recipe/chicken-veggie-stir-fry
  attribution: Tasty — Chicken & Veggie Stir-Fry
  checked_on: 2026-08-17
yield: 6 servings
hands_on:
  min: 32
  max: 32
unattended:
  min: 0
  max: 0
ingredient_sections:
  - name: Stir Fry
    ingredients:
      - source_text: 1 lb chicken breast, cubed
        grocery_item:
          key: chicken-breast
          name: Chicken Breast
          store_section: {key: meat, name: Meat}
          shopping_mode: measured
        quantity: {kind: exact, amount: "1", unit: lb}
        preparation: cubed
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
      - source_text: 1 lb broccoli florets
        grocery_item:
          key: broccoli-florets
          name: Broccoli Florets
          store_section: {key: produce, name: Produce}
          shopping_mode: measured
        quantity: {kind: exact, amount: "1", unit: lb}
      - source_text: 8 oz mushroom, sliced
        grocery_item:
          key: mushrooms
          name: Mushrooms
          store_section: {key: produce, name: Produce}
          shopping_mode: measured
        quantity: {kind: exact, amount: "8", unit: oz}
        preparation: sliced
      - source_text: 3 tablespoons oil, for frying
        grocery_item:
          key: cooking-oil
          name: Cooking Oil
          store_section: {key: pantry, name: Pantry}
          shopping_mode: measured
        quantity: {kind: exact, amount: "3", unit: tbsp}
        preparation: divided for frying
      - source_text: 3 cloves garlic, minced
        grocery_item:
          key: garlic
          name: Garlic
          store_section: {key: produce, name: Produce}
          shopping_mode: counted
        quantity: {kind: exact, amount: "3", unit: clove}
        preparation: minced
      - source_text: 1 tablespoon ginger, minced
        grocery_item:
          key: fresh-ginger
          name: Fresh Ginger
          store_section: {key: produce, name: Produce}
          shopping_mode: measured
        quantity: {kind: exact, amount: "1", unit: tbsp}
        preparation: minced
      - source_text: 2 teaspoons sesame oil
        grocery_item:
          key: sesame-oil
          name: Sesame Oil
          store_section: {key: pantry, name: Pantry}
          shopping_mode: measured
        quantity: {kind: exact, amount: "2", unit: tsp}
      - source_text: ⅓ cup reduced sodium soy sauce
        grocery_item:
          key: reduced-sodium-soy-sauce
          name: Reduced-Sodium Soy Sauce
          store_section: {key: condiments, name: Condiments}
          shopping_mode: measured
        quantity: {kind: exact, amount: "1/3", unit: cup}
      - source_text: 1 tablespoon brown sugar
        grocery_item:
          key: brown-sugar
          name: Brown Sugar
          store_section: {key: baking, name: Baking}
          shopping_mode: measured
        quantity: {kind: exact, amount: "1", unit: tbsp}
      - source_text: 1 cup chicken broth
        grocery_item:
          key: chicken-broth
          name: Chicken Broth
          store_section: {key: pantry, name: Pantry}
          shopping_mode: measured
        quantity: {kind: exact, amount: "1", unit: cup}
      - source_text: ¼ cup flour
        grocery_item:
          key: all-purpose-flour
          name: All-Purpose Flour
          store_section: {key: baking, name: Baking}
          shopping_mode: measured
        quantity: {kind: exact, amount: "1/4", unit: cup}
      - source_text: Serve with hot rice or noodles.
        grocery_item:
          key: white-rice
          name: White Rice
          store_section: {key: pantry, name: Pantry}
          shopping_mode: presence-only
        quantity: {kind: unspecified}
        preparation: cooked for serving
        optional: true
instruction_sections:
  - name: Method
    steps:
      - Heat 1 tablespoon of the cooking oil in a large pan over medium-high heat. Add the chicken, season with salt and black pepper, and sauté until cooked through and browned. Remove the chicken and set aside.
      - Heat another 1 tablespoon cooking oil in the same pan and add the mushrooms. When they begin to soften, add the broccoli and stir-fry until tender. Remove the vegetables and set aside.
      - Add the final 1 tablespoon cooking oil to the pan and sauté the garlic and ginger until fragrant.
      - Add the sesame oil, reduced-sodium soy sauce, brown sugar, chicken broth, and flour. Stir until the sauce is smooth.
      - Return the chicken and vegetables to the pan and stir until heated through.
      - Serve with hot rice.
review:
  - field: hands_on
    kind: rewritten
    note: The source reports 20 minutes prep and 12 minutes cook time; recorded all 32 minutes as hands-on for the continuously tended stir-fry method.
    approved: true
  - field: ingredient_sections[0].ingredients[13]
    kind: conflict-resolved
    note: The source offers hot rice or noodles. Household review selected optional white rice so v1 has one concrete serving choice.
    approved: true
  - field: instruction_sections
    kind: rewritten
    note: Preserved the source method, made the three one-tablespoon oil additions explicit, grouped the named sauce ingredients, and omitted the non-action "Enjoy!" line.
    approved: true
---

# Chicken Veggie Stir Fry

> Approved bootstrap recipe. YAML front matter is ingested into SQLite; the sections below
> are the checked human-readable view.

## Recipe details

- Source: [Tasty — Chicken & Veggie Stir-Fry](https://tasty.co/recipe/chicken-veggie-stir-fry) (`source`)
- Source checked: 2026-08-17
- Yield: 6 servings
- Hands-on: 32 minutes
- Unattended: 0 minutes

## Ingredients

### Stir Fry

- 1 lb chicken breast, cubed
  - Shopping: 1 lb Chicken Breast — Meat
  - Preparation: cubed
- salt, to taste
  - Shopping: Salt — Spices
- pepper, to taste
  - Shopping: Black Pepper — Spices
- 1 lb broccoli florets
  - Shopping: 1 lb Broccoli Florets — Produce
- 8 oz mushroom, sliced
  - Shopping: 8 oz Mushrooms — Produce
  - Preparation: sliced
- 3 tablespoons oil, for frying
  - Shopping: 3 tbsp Cooking Oil — Pantry
  - Preparation: divided for frying
- 3 cloves garlic, minced
  - Shopping: 3 clove Garlic — Produce
  - Preparation: minced
- 1 tablespoon ginger, minced
  - Shopping: 1 tbsp Fresh Ginger — Produce
  - Preparation: minced
- 2 teaspoons sesame oil
  - Shopping: 2 tsp Sesame Oil — Pantry
- ⅓ cup reduced sodium soy sauce
  - Shopping: 1/3 cup Reduced-Sodium Soy Sauce — Condiments
- 1 tablespoon brown sugar
  - Shopping: 1 tbsp Brown Sugar — Baking
- 1 cup chicken broth
  - Shopping: 1 cup Chicken Broth — Pantry
- ¼ cup flour
  - Shopping: 1/4 cup All-Purpose Flour — Baking
- Serve with hot rice or noodles.
  - Shopping: White Rice — optional — Pantry
  - Preparation: cooked for serving
  - Optional: yes

## Instructions

### Method

1. Heat 1 tablespoon of the cooking oil in a large pan over medium-high heat. Add the chicken, season with salt and black pepper, and sauté until cooked through and browned. Remove the chicken and set aside.
2. Heat another 1 tablespoon cooking oil in the same pan and add the mushrooms. When they begin to soften, add the broccoli and stir-fry until tender. Remove the vegetables and set aside.
3. Add the final 1 tablespoon cooking oil to the pan and sauté the garlic and ginger until fragrant.
4. Add the sesame oil, reduced-sodium soy sauce, brown sugar, chicken broth, and flour. Stir until the sauce is smooth.
5. Return the chicken and vegetables to the pan and stir until heated through.
6. Serve with hot rice.

## One-batch grocery preview

### Baking

- 1 tbsp Brown Sugar
- 1/4 cup All-Purpose Flour

### Condiments

- 1/3 cup Reduced-Sodium Soy Sauce

### Meat

- 1 lb Chicken Breast

### Pantry

- 3 tbsp Cooking Oil
- 2 tsp Sesame Oil
- 1 cup Chicken Broth
- White Rice — optional

### Produce

- 1 lb Broccoli Florets
- 8 oz Mushrooms
- 3 clove Garlic
- 1 tbsp Fresh Ginger

### Spices

- Salt
- Black Pepper

## Approved true-up decisions

- `hands_on` — **rewritten:** The source reports 20 minutes prep and 12 minutes cook time; recorded all 32 minutes as hands-on for the continuously tended stir-fry method.
- `ingredient_sections[0].ingredients[13]` — **conflict-resolved:** The source offers hot rice or noodles. Household review selected optional white rice so v1 has one concrete serving choice.
- `instruction_sections` — **rewritten:** Preserved the source method, made the three one-tablespoon oil additions explicit, grouped the named sauce ingredients, and omitted the non-action "Enjoy!" line.
