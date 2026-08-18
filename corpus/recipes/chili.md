---
format_version: 1
key: chili
name: Chili
status: verified
approved_on: 2026-08-17
source:
  relationship: source
  url: https://www.julieseatsandtreats.com/easy-chili-recipe/
  attribution: Julie's Eats & Treats — Easy Homemade Chili Recipe
  checked_on: 2026-08-17
yield: 4 servings
hands_on:
  min: 10
  max: 10
unattended:
  min: 25
  max: 30
ingredient_sections:
  - name: Chili
    ingredients:
      - source_text: 1 lb ground hamburger
        grocery_item:
          key: ground-beef
          name: Ground Beef
          store_section: {key: meat, name: Meat}
          shopping_mode: measured
        quantity: {kind: exact, amount: "1", unit: lb}
      - source_text: 1 white onion
        grocery_item:
          key: white-onion
          name: White Onion
          store_section: {key: produce, name: Produce}
          shopping_mode: counted
        quantity: {kind: exact, amount: "1", unit: each}
      - source_text: 3 cloves garlic (minced)
        grocery_item:
          key: garlic
          name: Garlic
          store_section: {key: produce, name: Produce}
          shopping_mode: counted
        quantity: {kind: exact, amount: "3", unit: clove}
        preparation: minced
      - source_text: 15 oz beef broth
        grocery_item:
          key: beef-broth
          name: Beef Broth
          store_section: {key: pantry, name: Pantry}
          shopping_mode: measured
        quantity: {kind: exact, amount: "15", unit: oz}
      - source_text: 1 can chili beans (15.5 oz)
        grocery_item:
          key: chili-beans
          name: Chili Beans
          store_section: {key: pantry, name: Pantry}
          shopping_mode: counted
        quantity:
          kind: exact
          amount: "1"
          package: {type: can, size: "15.5", unit: oz}
      - source_text: 8 oz tomato sauce
        grocery_item:
          key: tomato-sauce
          name: Tomato Sauce
          store_section: {key: pantry, name: Pantry}
          shopping_mode: measured
        quantity: {kind: exact, amount: "8", unit: oz}
      - source_text: 2 Tbsp chili powder
        grocery_item:
          key: chili-powder
          name: Chili Powder
          store_section: {key: spices, name: Spices}
          shopping_mode: measured
        quantity: {kind: exact, amount: "2", unit: tbsp}
      - source_text: 1 Tbsp ground cumin
        grocery_item:
          key: ground-cumin
          name: Ground Cumin
          store_section: {key: spices, name: Spices}
          shopping_mode: measured
        quantity: {kind: exact, amount: "1", unit: tbsp}
      - source_text: 1 tsp garlic salt
        grocery_item:
          key: garlic-salt
          name: Garlic Salt
          store_section: {key: spices, name: Spices}
          shopping_mode: measured
        quantity: {kind: exact, amount: "1", unit: tsp}
      - source_text: 1/2 tsp oregano
        grocery_item:
          key: dried-oregano
          name: Dried Oregano
          store_section: {key: spices, name: Spices}
          shopping_mode: measured
        quantity: {kind: exact, amount: "1/2", unit: tsp}
      - source_text: 2 Tbsp tomato paste
        grocery_item:
          key: tomato-paste
          name: Tomato Paste
          store_section: {key: pantry, name: Pantry}
          shopping_mode: measured
        quantity: {kind: exact, amount: "2", unit: tbsp}
      - source_text: sour cream and shredded cheese to top with (optional)
        grocery_item:
          key: sour-cream
          name: Sour Cream
          store_section: {key: dairy, name: Dairy}
          shopping_mode: presence-only
        quantity: {kind: unspecified}
        preparation: for topping
        optional: true
      - source_text: sour cream and shredded cheese to top with (optional)
        grocery_item:
          key: shredded-cheese
          name: Shredded Cheese
          store_section: {key: dairy, name: Dairy}
          shopping_mode: presence-only
        quantity: {kind: unspecified}
        preparation: for topping
        optional: true
      - source_text: 1 can petite diced tomatoes (14.5 oz can)
        grocery_item:
          key: petite-diced-tomatoes
          name: Petite Diced Tomatoes
          store_section: {key: pantry, name: Pantry}
          shopping_mode: counted
        quantity:
          kind: exact
          amount: "1"
          package: {type: can, size: "14.5", unit: oz}
instruction_sections:
  - name: Method
    steps:
      - In a Dutch oven or soup pot over medium heat, cook the ground beef and onion for about 5 minutes, stirring occasionally, until the onion is translucent and the beef is cooked through. Add the garlic during the final minute, then drain the grease.
      - Add the beef broth, chili beans, tomato sauce, chili powder, cumin, garlic salt, oregano, tomato paste, and diced tomatoes. Stir, bring to a low boil, reduce the heat to low, and simmer uncovered for 20 minutes, stirring occasionally.
      - Remove from the heat and let stand for 5 to 10 minutes. Serve with sour cream and shredded cheese if desired.
review:
  - field: source
    kind: conflict-resolved
    note: Recovered the exact current source URL from the captured site and matched every captured ingredient and instruction against its live structured recipe data, resolving the screenshot gap.
    approved: true
  - field: hands_on_and_unattended
    kind: rewritten
    note: Preserved the source's 10-minute prep and 20-minute simmer, and included its explicit 5-to-10-minute standing time in the unattended range.
    approved: true
  - field: ingredient_sections[0].ingredients[11:13]
    kind: rewritten
    note: Split the source's combined optional sour cream and shredded cheese topping line so neither contribution is dropped.
    approved: true
  - field: instruction_sections
    kind: rewritten
    note: Preserved all three authoritative method stages while naming each ingredient represented by "the rest of the ingredients."
    approved: true
---

# Chili

> Approved bootstrap recipe. YAML front matter is ingested into SQLite; the sections below
> are the checked human-readable view.

## Recipe details

- Source: [Julie's Eats & Treats — Easy Homemade Chili Recipe](https://www.julieseatsandtreats.com/easy-chili-recipe/) (`source`)
- Source checked: 2026-08-17
- Yield: 4 servings
- Hands-on: 10 minutes
- Unattended: 25–30 minutes

## Ingredients

### Chili

- 1 lb ground hamburger
  - Shopping: 1 lb Ground Beef — Meat
- 1 white onion
  - Shopping: 1 White Onion — Produce
- 3 cloves garlic (minced)
  - Shopping: 3 clove Garlic — Produce
  - Preparation: minced
- 15 oz beef broth
  - Shopping: 15 oz Beef Broth — Pantry
- 1 can chili beans (15.5 oz)
  - Shopping: 1 × 15.5 oz can Chili Beans — Pantry
- 8 oz tomato sauce
  - Shopping: 8 oz Tomato Sauce — Pantry
- 2 Tbsp chili powder
  - Shopping: 2 tbsp Chili Powder — Spices
- 1 Tbsp ground cumin
  - Shopping: 1 tbsp Ground Cumin — Spices
- 1 tsp garlic salt
  - Shopping: 1 tsp Garlic Salt — Spices
- 1/2 tsp oregano
  - Shopping: 1/2 tsp Dried Oregano — Spices
- 2 Tbsp tomato paste
  - Shopping: 2 tbsp Tomato Paste — Pantry
- sour cream and shredded cheese to top with (optional)
  - Shopping: Sour Cream — optional — Dairy
  - Preparation: for topping
  - Optional: yes
- sour cream and shredded cheese to top with (optional)
  - Shopping: Shredded Cheese — optional — Dairy
  - Preparation: for topping
  - Optional: yes
- 1 can petite diced tomatoes (14.5 oz can)
  - Shopping: 1 × 14.5 oz can Petite Diced Tomatoes — Pantry

## Instructions

### Method

1. In a Dutch oven or soup pot over medium heat, cook the ground beef and onion for about 5 minutes, stirring occasionally, until the onion is translucent and the beef is cooked through. Add the garlic during the final minute, then drain the grease.
2. Add the beef broth, chili beans, tomato sauce, chili powder, cumin, garlic salt, oregano, tomato paste, and diced tomatoes. Stir, bring to a low boil, reduce the heat to low, and simmer uncovered for 20 minutes, stirring occasionally.
3. Remove from the heat and let stand for 5 to 10 minutes. Serve with sour cream and shredded cheese if desired.

## One-batch grocery preview

### Dairy

- Sour Cream — optional
- Shredded Cheese — optional

### Meat

- 1 lb Ground Beef

### Pantry

- 15 oz Beef Broth
- 1 × 15.5 oz can Chili Beans
- 8 oz Tomato Sauce
- 2 tbsp Tomato Paste
- 1 × 14.5 oz can Petite Diced Tomatoes

### Produce

- 1 White Onion
- 3 clove Garlic

### Spices

- 2 tbsp Chili Powder
- 1 tbsp Ground Cumin
- 1 tsp Garlic Salt
- 1/2 tsp Dried Oregano

## Approved true-up decisions

- `source` — **conflict-resolved:** Recovered the exact current source URL from the captured site and matched every captured ingredient and instruction against its live structured recipe data, resolving the screenshot gap.
- `hands_on_and_unattended` — **rewritten:** Preserved the source's 10-minute prep and 20-minute simmer, and included its explicit 5-to-10-minute standing time in the unattended range.
- `ingredient_sections[0].ingredients[11:13]` — **rewritten:** Split the source's combined optional sour cream and shredded cheese topping line so neither contribution is dropped.
- `instruction_sections` — **rewritten:** Preserved all three authoritative method stages while naming each ingredient represented by "the rest of the ingredients."
