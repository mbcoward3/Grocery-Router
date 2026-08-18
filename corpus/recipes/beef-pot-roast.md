---
format_version: 1
key: beef-pot-roast
name: Beef Pot Roast
status: verified
approved_on: '2026-08-17'
source:
  relationship: source
  attribution: Dinner Then Dessert slow-cooker recipe captured in Recipes.pdf, pages 22–26
  checked_on: '2026-08-17'
hands_on:
  min: 25
  max: 25
unattended:
  min: 480
  max: 480
ingredient_sections:
- name: Pot Roast
  ingredients:
  - source_text: 4-5 pound chuck roast
    grocery_item:
      key: chuck-roast
      name: Chuck Roast
      store_section:
        key: meat
        name: Meat
      shopping_mode: measured
    quantity:
      kind: range
      amount: '4'
      maximum: '5'
      unit: lb
  - source_text: 2 tablespoons vegetable oil
    grocery_item:
      key: vegetable-oil
      name: Vegetable Oil
      store_section:
        key: pantry
        name: Pantry
      shopping_mode: measured
    quantity:
      kind: exact
      amount: '2'
      unit: tbsp
  - source_text: 2 teaspoons kosher salt
    grocery_item:
      key: kosher-salt
      name: Kosher Salt
      store_section:
        key: spices
        name: Spices
      shopping_mode: measured
    quantity:
      kind: exact
      amount: '2'
      unit: tsp
  - source_text: 1 teaspoon coarse ground black pepper
    grocery_item:
      key: black-pepper
      name: Black Pepper
      store_section:
        key: spices
        name: Spices
      shopping_mode: measured
    quantity:
      kind: exact
      amount: '1'
      unit: tsp
    preparation: coarsely ground
  - source_text: 1 teaspoon dried thyme
    grocery_item:
      key: dried-thyme
      name: Dried Thyme
      store_section:
        key: spices
        name: Spices
      shopping_mode: measured
    quantity:
      kind: exact
      amount: '1'
      unit: tsp
  - source_text: 1 pound carrots, peeled and cut into 2 inch chunks
    grocery_item:
      key: carrots
      name: Carrots
      store_section:
        key: produce
        name: Produce
      shopping_mode: counted
    quantity:
      kind: exact
      amount: '1'
      unit: lb
    preparation: peeled and cut into 2-inch chunks
  - source_text: 2 pounds Yukon Gold potatoes, peeled and cut into 2 inch chunks
    grocery_item:
      key: yukon-gold-potatoes
      name: Yukon Gold Potatoes
      store_section:
        key: produce
        name: Produce
      shopping_mode: measured
    quantity:
      kind: exact
      amount: '2'
      unit: lb
    preparation: peeled and cut into 2-inch chunks
  - source_text: 2 cloves garlic, minced
    grocery_item:
      key: garlic
      name: Garlic
      store_section:
        key: produce
        name: Produce
      shopping_mode: counted
    quantity:
      kind: exact
      amount: '2'
      unit: clove
    preparation: minced
  - source_text: 2 cups beef broth
    grocery_item:
      key: beef-broth
      name: Beef Broth
      store_section:
        key: pantry
        name: Pantry
      shopping_mode: measured
    quantity:
      kind: exact
      amount: '2'
      unit: cup
  - source_text: 2 tablespoons cornstarch
    grocery_item:
      key: cornstarch
      name: Cornstarch
      store_section:
        key: baking
        name: Baking
      shopping_mode: measured
    quantity:
      kind: exact
      amount: '2'
      unit: tbsp
  - source_text: 2 tablespoons cold water
    grocery_item:
      key: water
      name: Water
      store_section:
        key: non-shopping
        name: Non-shopping
      shopping_mode: measured
    quantity:
      kind: exact
      amount: '2'
      unit: tbsp
    preparation: cold
    non_shopping: true
  - source_text: minced parsley, optional garnish
    grocery_item:
      key: fresh-parsley
      name: Fresh Parsley
      store_section:
        key: produce
        name: Produce
      shopping_mode: measured
    quantity:
      kind: unspecified
    preparation: minced garnish
    optional: true
instruction_sections:
- name: Method
  steps:
  - Season the roast with kosher salt, black pepper, and thyme. Heat the vegetable oil over medium-high heat and brown the roast deeply for 4 to 5 minutes per side.
  - Place the carrots, potatoes, and garlic in the slow cooker. Set the browned roast on top, add the beef broth, and cover.
  - Cook on low for 8 hours.
  - During the final hour, stir the cornstarch and cold water into a smooth slurry and add it to the slow cooker to thicken the cooking liquid.
  - Serve the tender roast and vegetables with the thickened liquid and parsley if desired.
review:
- field: source
  kind: conflict-resolved
  note: Used the captured Dinner Then Dessert recipe and corpus title because the exact live URL and on-page title could not be recovered.
  approved: true
- field: instruction_sections
  kind: conflict-resolved
  note: Under delegated household review, selected low for 8 hours as the concrete source method and completed the ad-obscured final slurry step. Logged for revisit in trueup/CONTROVERSIAL_CALLS.md.
  approved: true
- field: ingredient_sections[0].ingredients[10]
  kind: conflict-resolved
  note: Retained the exact cold-water slurry contribution as non-shopping.
  approved: true
---

# Beef Pot Roast

> Approved bootstrap recipe. YAML front matter is ingested into SQLite; the sections below
> are the checked human-readable view.

## Recipe details

- Source: Dinner Then Dessert slow-cooker recipe captured in Recipes.pdf, pages 22–26 (`source`)
- Source checked: 2026-08-17
- Yield: unknown
- Hands-on: 25 minutes
- Unattended: 480 minutes

## Ingredients

### Pot Roast

- 4-5 pound chuck roast
  - Shopping: 4–5 lb Chuck Roast — Meat
- 2 tablespoons vegetable oil
  - Shopping: 2 tbsp Vegetable Oil — Pantry
- 2 teaspoons kosher salt
  - Shopping: 2 tsp Kosher Salt — Spices
- 1 teaspoon coarse ground black pepper
  - Shopping: 1 tsp Black Pepper — Spices
  - Preparation: coarsely ground
- 1 teaspoon dried thyme
  - Shopping: 1 tsp Dried Thyme — Spices
- 1 pound carrots, peeled and cut into 2 inch chunks
  - Shopping: 1 lb Carrots — Produce
  - Preparation: peeled and cut into 2-inch chunks
- 2 pounds Yukon Gold potatoes, peeled and cut into 2 inch chunks
  - Shopping: 2 lb Yukon Gold Potatoes — Produce
  - Preparation: peeled and cut into 2-inch chunks
- 2 cloves garlic, minced
  - Shopping: 2 clove Garlic — Produce
  - Preparation: minced
- 2 cups beef broth
  - Shopping: 2 cup Beef Broth — Pantry
- 2 tablespoons cornstarch
  - Shopping: 2 tbsp Cornstarch — Baking
- 2 tablespoons cold water
  - Recipe only: 2 tbsp Water — not added to the grocery list
  - Preparation: cold
- minced parsley, optional garnish
  - Shopping: Fresh Parsley — optional — Produce
  - Preparation: minced garnish
  - Optional: yes

## Instructions

### Method

1. Season the roast with kosher salt, black pepper, and thyme. Heat the vegetable oil over medium-high heat and brown the roast deeply for 4 to 5 minutes per side.
2. Place the carrots, potatoes, and garlic in the slow cooker. Set the browned roast on top, add the beef broth, and cover.
3. Cook on low for 8 hours.
4. During the final hour, stir the cornstarch and cold water into a smooth slurry and add it to the slow cooker to thicken the cooking liquid.
5. Serve the tender roast and vegetables with the thickened liquid and parsley if desired.

## One-batch grocery preview

### Baking

- 2 tbsp Cornstarch

### Meat

- 4–5 lb Chuck Roast

### Pantry

- 2 tbsp Vegetable Oil
- 2 cup Beef Broth

### Produce

- 1 lb Carrots
- 2 lb Yukon Gold Potatoes
- 2 clove Garlic
- Fresh Parsley — optional

### Spices

- 2 tsp Kosher Salt
- 1 tsp Black Pepper
- 1 tsp Dried Thyme

## Approved true-up decisions

- `source` — **conflict-resolved:** Used the captured Dinner Then Dessert recipe and corpus title because the exact live URL and on-page title could not be recovered.
- `instruction_sections` — **conflict-resolved:** Under delegated household review, selected low for 8 hours as the concrete source method and completed the ad-obscured final slurry step. Logged for revisit in trueup/CONTROVERSIAL_CALLS.md.
- `ingredient_sections[0].ingredients[10]` — **conflict-resolved:** Retained the exact cold-water slurry contribution as non-shopping.
