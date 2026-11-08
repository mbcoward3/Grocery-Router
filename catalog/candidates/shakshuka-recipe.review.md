# Shakshuka Recipe

- Candidate: `shakshuka#$/@graph[7]`
- Catalog key: `shakshuka-recipe`
- State: `reviewable`
- Source: `shakshuka` <https://www.loveandlemons.com/shakshuka-recipe/>
- Selected JSON-LD: artifact `shakshuka.json`, recipe 0, script 0, path `$/@graph[7]`

## Blockers

- **BLOCKER:** ingredient_sections[0].ingredients[11].source_text: alternative: Source lists parsley or cilantro leaves; proposed parsley as first-listed baseline.
- **BLOCKER:** ingredient_sections[0].ingredients[12].grocery_item: backfill: Proposed new grocery item for feta cheese.
- **BLOCKER:** ingredient_sections[0].ingredients[13].grocery_item: backfill: Proposed new grocery item for pita.
- **BLOCKER:** ingredient_sections[0].ingredients[13].quantity: unresolved: No quantity specified for pita.
- **BLOCKER:** ingredient_sections[0].ingredients[6].grocery_item: backfill: Proposed new grocery item for cayenne pepper.
- **BLOCKER:** ingredient_sections[0].ingredients[6].quantity: unresolved: Pinch has no supported unit; quantity left unspecified.
- **BLOCKER:** ingredient_sections[0].ingredients[7].grocery_item: backfill: Proposed new grocery item for fire-roasted crushed tomatoes.
- **BLOCKER:** ingredient_sections[0].ingredients[9].quantity: unresolved: No quantity specified for black pepper.
- **BLOCKER:** unattended: ambiguous: Cook time includes active stovetop work and covered egg cooking; unattended time not clearly separable.

## Summary

- Yield: 4
- Hands on: 10 min
- Unattended: (unresolved)

## Ingredients

### Ingredients

- [0] 2 tablespoons extra-virgin olive oil
  - Proposal: Olive Oil; quantity: 2 tbsp; grocery: match olive-oil
- [1] 1  small white onion (diced)
  - Proposal: White Onion; quantity: 1 each; grocery: match white-onion
- [2] 1  red bell pepper (stemmed, seeded, and diced)
  - Proposal: Red Bell Pepper; quantity: 1 each; grocery: match red-bell-pepper
- [3] 3  garlic cloves (minced)
  - Proposal: Garlic; quantity: 3 clove; grocery: match garlic
- [4] 1 teaspoon ground cumin
  - Proposal: Ground Cumin; quantity: 1 tsp; grocery: match ground-cumin
- [5] ½ teaspoon paprika
  - Proposal: Paprika; quantity: 1/2 tsp; grocery: match paprika
- [6] Pinch cayenne pepper (optional) _( optional )_
  - Proposal: Cayenne Pepper; quantity: unspecified; grocery: create cayenne-pepper
- [7] 1 (28-ounce) can fire-roasted crushed tomatoes
  - Proposal: Fire-Roasted Crushed Tomatoes; quantity: 1 can; grocery: create fire-roasted-crushed-tomatoes
- [8] ½ teaspoon sea salt (plus more to taste)
  - Proposal: Salt; quantity: 1/2 tsp; grocery: match salt
- [9] Freshly ground black pepper
  - Proposal: Black Pepper; quantity: unspecified; grocery: match black-pepper
- [10] 6  large eggs
  - Proposal: Eggs; quantity: 6 each; grocery: match eggs
- [11] ¼ cup fresh parsley or cilantro leaves
  - Proposal: Fresh Parsley; quantity: 1/4 cup; grocery: match fresh-parsley
- [12] ¼ cup crumbled feta cheese
  - Proposal: Feta Cheese; quantity: 1/4 cup; grocery: create feta-cheese
- [13] Pita (for serving)
  - Proposal: Pita; quantity: unspecified; grocery: create pita

## Instructions

### Method

1. Heat the olive oil in a large lidded skillet over medium heat. Add the onion and red peppers and cook for 5 to 8 minutes, or until softened. Add the garlic, cumin, paprika, and cayenne, if using, and cook, stirring, for 30 seconds, or until fragrant.
2. Add the tomatoes, salt, and several grinds of pepper. Simmer over low heat, stirring often, for 15 minutes, or until the sauce has thickened.
3. Make 6 wells in the sauce and crack one egg into each well. Cover and cook until the eggs are set, 4 to 8 minutes. The timing will depend on how runny or firm you like your eggs. Season to taste with salt and pepper and top with the parsley and feta. Serve with pita.

## Issues

- BLOCKER `ingredient_sections[0].ingredients[6].grocery_item`: backfill: Proposed new grocery item for cayenne pepper.
- BLOCKER `ingredient_sections[0].ingredients[6].quantity`: unresolved: Pinch has no supported unit; quantity left unspecified.
- BLOCKER `ingredient_sections[0].ingredients[7].grocery_item`: backfill: Proposed new grocery item for fire-roasted crushed tomatoes.
- BLOCKER `ingredient_sections[0].ingredients[9].quantity`: unresolved: No quantity specified for black pepper.
- BLOCKER `ingredient_sections[0].ingredients[11].source_text`: alternative: Source lists parsley or cilantro leaves; proposed parsley as first-listed baseline.
- BLOCKER `ingredient_sections[0].ingredients[12].grocery_item`: backfill: Proposed new grocery item for feta cheese.
- BLOCKER `ingredient_sections[0].ingredients[13].grocery_item`: backfill: Proposed new grocery item for pita.
- BLOCKER `ingredient_sections[0].ingredients[13].quantity`: unresolved: No quantity specified for pita.
- BLOCKER `unattended`: ambiguous: Cook time includes active stovetop work and covered egg cooking; unattended time not clearly separable.
