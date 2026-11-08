# French Onion Frittata

- Candidate: `frittata#$`
- Catalog key: `french-onion-frittata`
- State: `reviewable`
- Source: `frittata` <https://www.thekitchn.com/how-to-make-frittata-246166>
- Selected JSON-LD: artifact `frittata.json`, recipe 0, script 0, path `$`

## Blockers

- **BLOCKER:** hands_on: unresolved: JSON-LD totalTime is PT0S and no prepTime/cookTime values are provided; instruction timing mixes active and intermittent work.
- **BLOCKER:** ingredient_sections[0].ingredients[10].grocery_item: backfill: Proposed new grocery item for Gruyère cheese.
- **BLOCKER:** ingredient_sections[0].ingredients[11].grocery_item: backfill: Proposed new grocery item for fresh chives.
- **BLOCKER:** ingredient_sections[0].ingredients[1].grocery_item: backfill: Proposed new grocery item for day-old bread.
- **BLOCKER:** ingredient_sections[0].ingredients[7]: alternative: Source lists whole or 2% milk; selected whole milk as baseline proposal.
- **BLOCKER:** ingredient_sections[0].ingredients[8].grocery_item: backfill: Proposed new grocery item for Dijon mustard rather than using generic mustard.
- **BLOCKER:** unattended: unresolved: Unattended time not explicitly provided; baking and cooling times appear passive but source duration metadata is missing.

## Summary

- Yield: 6
- Hands on: (unresolved)
- Unattended: (unresolved)

## Ingredients

### Ingredients

- [0] 2 tablespoons unsalted butter, divided
  - Proposal: Unsalted Butter; quantity: 2 tbsp; grocery: match unsalted-butter
- [1] 2 cups diced day-old bread
  - Proposal: Day-Old Bread; quantity: 2 cup; grocery: create day-old-bread
- [2] 1 tablespoon olive oil
  - Proposal: Olive Oil; quantity: 1 tbsp; grocery: match olive-oil
- [3] 2  large yellow onions, thinly sliced
  - Proposal: Yellow Onion; quantity: 2 each; grocery: match yellow-onion
- [4] 1 1/2 teaspoons kosher salt, divided
  - Proposal: Kosher Salt; quantity: 1 1/2 tsp; grocery: match kosher-salt
- [5] 1 tablespoon balsamic vinegar
  - Proposal: Balsamic Vinegar; quantity: 1 tbsp; grocery: match balsamic-vinegar
- [6] 8  large eggs
  - Proposal: Eggs; quantity: 8 each; grocery: match eggs
- [7] 1/4 cup whole or 2% milk
  - Proposal: Milk; quantity: 1/4 cup; grocery: match milk
- [8] 1 tablespoon Dijon mustard
  - Proposal: Dijon Mustard; quantity: 1 tbsp; grocery: create dijon-mustard
- [9] 1/4 teaspoon freshly ground black pepper
  - Proposal: Black Pepper; quantity: 1/4 tsp; grocery: match black-pepper
- [10] 1/2 cup diced Gruyère cheese (2 ounces)
  - Proposal: Gruyère Cheese; quantity: 1/2 cup; grocery: create gruyere-cheese
- [11] 1 tablespoon finely chopped fresh chives
  - Proposal: Fresh Chives; quantity: 1 tbsp; grocery: create fresh-chives

## Instructions

### Method

1. Arrange a rack in the middle of the oven and heat to 400°F.
2. Heat 1 tablespoon of the butter in a 10-inch cast iron skillet over medium heat until bubbling. Add the bread cubes to the pan, toss to coat with the butter, and arrange in a single layer. Toast the bread, tossing every minute or so, until the bread cubes are golden-brown on all sides, about 5 minutes total. Transfer the croutons to a bowl; set aside.
3. Reduce the heat to low. Add the oil, onions, and 1/2 teaspoon of the salt to the same skillet. Cook, stirring every 5 to 10 minutes and scraping any browned build-up from the bottom of the pan, until the onions are soft and deeply browned, about 40 minutes total. Add the vinegar and scrape up the browned bits at the bottom of the pan.
4. Whisk together the eggs, milk, mustard, remaining 1 teaspoon salt, and pepper in a large bowl. Stir the remaining butter and croutons into the skillet, then spread in an even layer. Pour the egg mixture over the top. Tilt the pan to make sure the eggs settle evenly. Top with the cheese. Cook until the eggs at the edges of the skillet begin to set, 2 to 4 minutes.
5. Bake until the eggs are set, 8 to 10 minutes. To check, cut a small slit in the center of the frittata. If raw eggs run into the cut, bake for another few minutes; if the eggs are set, pull the frittata from the oven.
6. Cool in the pan for 5 minutes, top with chives, then slice into wedges and serve warm.

### Recipe Notes

1. Make ahead: The croutons can be made up to 1 week in advance and stored in an airtight container at room temperature. The caramelized onions can be made up to 3 days in advance and stored in an airtight container in the refrigerator, or up to 3 months in advance and stored in the freezer.Storage: Leftovers can be stored in a covered container in the refrigerator for up to 5 days.

## Issues

- BLOCKER `hands_on`: unresolved: JSON-LD totalTime is PT0S and no prepTime/cookTime values are provided; instruction timing mixes active and intermittent work.
- BLOCKER `unattended`: unresolved: Unattended time not explicitly provided; baking and cooling times appear passive but source duration metadata is missing.
- BLOCKER `ingredient_sections[0].ingredients[1].grocery_item`: backfill: Proposed new grocery item for day-old bread.
- BLOCKER `ingredient_sections[0].ingredients[7]`: alternative: Source lists whole or 2% milk; selected whole milk as baseline proposal.
- BLOCKER `ingredient_sections[0].ingredients[8].grocery_item`: backfill: Proposed new grocery item for Dijon mustard rather than using generic mustard.
- WARNING `ingredient_sections[0].ingredients[10]`: dual-unit: Source provides 1/2 cup and 2 ounces; quantity uses first source measure and preserves ounces in source_text.
- BLOCKER `ingredient_sections[0].ingredients[10].grocery_item`: backfill: Proposed new grocery item for Gruyère cheese.
- BLOCKER `ingredient_sections[0].ingredients[11].grocery_item`: backfill: Proposed new grocery item for fresh chives.
