# Chicken Marsala

- Candidate: `chicken-marsala#$/@graph[6]`
- Catalog key: `chicken-marsala`
- State: `reviewable`
- Source: `chicken-marsala` <https://www.recipetineats.com/chicken-marsala/>
- Selected JSON-LD: artifact `chicken-marsala.json`, recipe 0, script 0, path `$/@graph[6]`

## Blockers

- **BLOCKER:** ingredient_sections[0].ingredients[0]: alternative: Source lists chicken breasts or 4 thighs; selected chicken breasts as baseline.
- **BLOCKER:** ingredient_sections[0].ingredients[11].grocery_item: backfill: Proposed new grocery item for heavy cream.
- **BLOCKER:** ingredient_sections[0].ingredients[12]: alternative: Source lists cooking salt / kosher salt; selected generic salt as proposal.
- **BLOCKER:** ingredient_sections[0].ingredients[1]: alternative: Source lists cooking salt / kosher salt; selected generic salt as proposal.
- **BLOCKER:** ingredient_sections[0].ingredients[6].grocery_item: backfill: Proposed new grocery item for eschalots.
- **BLOCKER:** ingredient_sections[0].ingredients[7].quantity: unresolved: Source says '1 garlic' without supported unit or clear piece noun; quantity left unspecified.
- **BLOCKER:** ingredient_sections[0].ingredients[9].grocery_item: backfill: Proposed new grocery item for dry marsala wine.
- **BLOCKER:** instruction_sections[0].steps[0]: source-note: Note 5 is referenced but absent from JSON-LD.
- **BLOCKER:** source note Note 1 is unresolved
- **BLOCKER:** source note Note 2 is unresolved
- **BLOCKER:** source note Note 3 is unresolved
- **BLOCKER:** source note Note 4 is unresolved
- **BLOCKER:** source note Note 5 is unresolved
- **BLOCKER:** source_notes: source-note: Note 1 is referenced but absent from JSON-LD.
- **BLOCKER:** source_notes: source-note: Note 2 is referenced but absent from JSON-LD.
- **BLOCKER:** source_notes: source-note: Note 3 is referenced but absent from JSON-LD.
- **BLOCKER:** source_notes: source-note: Note 4 is referenced but absent from JSON-LD.

## Summary

- Yield: 4
- Hands on: 25 min
- Unattended: (unresolved)

## Ingredients

### Ingredients

- [0] 2 large chicken breasts ((300g/10oz each), cut in half horizontally (or 4 thighs, Note 1))
  - Proposal: Chicken Breast; quantity: 2 each; grocery: match chicken-breast
- [1] 1/2 tsp cooking salt / kosher salt
  - Proposal: Salt; quantity: 1/2 tsp; grocery: match salt
- [2] 1/2 tsp black pepper
  - Proposal: Black Pepper; quantity: 1/2 tsp; grocery: match black-pepper
- [3] 1/4 cup flour (, plain/all-purpose)
  - Proposal: All-Purpose Flour; quantity: 1/4 cup; grocery: match all-purpose-flour
- [4] 2 tbsp extra virgin olive oil
  - Proposal: Olive Oil; quantity: 2 tbsp; grocery: match olive-oil
- [5] 2 tbsp / 30g  unsalted butter
  - Proposal: Unsalted Butter; quantity: 2 tbsp; grocery: match unsalted-butter
- [6] 2  eschalots ((US: shallots), peeled and cut into 1cm / 1/3" squares (Note 2))
  - Proposal: Echalots; quantity: 2 each; grocery: create eschalots
- [7] 1  garlic (, finely minced)
  - Proposal: Garlic; quantity: unspecified; grocery: match garlic
- [8] 2 cups white mushrooms (, sliced 0.5cm / 1/5&quot; thick)
  - Proposal: Mushrooms; quantity: 2 cup; grocery: match mushrooms
- [9] 1 cup dry marsala wine ((Note 3))
  - Proposal: Dry Marsala Wine; quantity: 1 cup; grocery: create dry-marsala-wine
- [10] 1/2 cup chicken stock/broth (, low sodium)
  - Proposal: Chicken Broth; quantity: 1/2 cup; grocery: match chicken-broth
- [11] 1/2 cup thickened / heavy cream ( (Note 4))
  - Proposal: Heavy Cream; quantity: 1/2 cup; grocery: create heavy-cream
- [12] 1/4 tsp cooking salt / kosher salt
  - Proposal: Salt; quantity: 1/4 tsp; grocery: match salt
- [13] 1/8 tsp black pepper
  - Proposal: Black Pepper; quantity: 1/8 tsp; grocery: match black-pepper
- [14] 1 tbsp finely chopped parsley (, for garnish (optional)) _( optional )_
  - Proposal: Fresh Parsley; quantity: 1 tbsp; grocery: match fresh-parsley

## Instructions

### Chicken escalopes:

1. Pound - Cut each breast in half to form 2 thin steaks. Cover with a freezer bag or Go-Between (Note 5) and pound to 1 cm / 0.4″ thickness using a meat mallet or rolling pin. This tenderises and ensures even cooking of the chicken.
2. Dust - Sprinkle the surface with half the salt, pepper then flour. Lightly rub flour across surface, turn and repeat with remaining salt, pepper and flour. Shake excess flour off each piece just before cooking.
3. Cook - Put half the butter and oil in a large non-stick pan over medium high heat. Once the butter is melted and foamy, place chicken in, then cook for 3 to 4 minutes until it's gorgeously golden and crispy. Turn and cook the other side for 2 minutes. Remove onto a plate.

### Creamy marsala sauce:

1. Sauté aromatics - In the same pan, add remaining butter and oil. Once butter is melted, add eschalots and garlic. Cook for 1 minute.
2. Cook mushrooms - Add mushrooms. Cook for 3 minutes, stirring regularly.
3. Reduce marsala - Add marsala, turn up heat to high and boil for 3 minutes or until reduced by half.
4. Thicken sauce - Add chicken stock, cream, salt and pepper. Stir, then lower heat so it's simmering (not boiling rapidly) and simmer for 3 to 5 minutes until the sauce thickens to a cream consistency (not too thick, will thicken more).
5. Rewarm chicken - Nestle chicken into sauce and leave for 30 seconds to reheat.
6. Serve! Take off the stove. Sprinkle with parsley, serve over starchy vehicle of choice (I chose mash. Rice, small pasta, polenta or bread for plate mopping also work well!).

## Issues

- WARNING `ingredient_sections[0].ingredients[0].quantity`: dual-unit: Source displays 300 g / 10 oz per breast; count is the single arithmetic requirement and both masses remain provenance.
- BLOCKER `ingredient_sections[0].ingredients[0]`: alternative: Source lists chicken breasts or 4 thighs; selected chicken breasts as baseline.
- BLOCKER `source_notes`: source-note: Note 1 is referenced but absent from JSON-LD.
- BLOCKER `ingredient_sections[0].ingredients[1]`: alternative: Source lists cooking salt / kosher salt; selected generic salt as proposal.
- WARNING `ingredient_sections[0].ingredients[5].quantity`: dual-unit: Source lists 2 tbsp / 30g; used first measure and preserved gram amount in source_text/note.
- BLOCKER `ingredient_sections[0].ingredients[6].grocery_item`: backfill: Proposed new grocery item for eschalots.
- BLOCKER `source_notes`: source-note: Note 2 is referenced but absent from JSON-LD.
- BLOCKER `ingredient_sections[0].ingredients[7].quantity`: unresolved: Source says '1 garlic' without supported unit or clear piece noun; quantity left unspecified.
- BLOCKER `ingredient_sections[0].ingredients[9].grocery_item`: backfill: Proposed new grocery item for dry marsala wine.
- BLOCKER `source_notes`: source-note: Note 3 is referenced but absent from JSON-LD.
- BLOCKER `ingredient_sections[0].ingredients[11].grocery_item`: backfill: Proposed new grocery item for heavy cream.
- BLOCKER `source_notes`: source-note: Note 4 is referenced but absent from JSON-LD.
- BLOCKER `ingredient_sections[0].ingredients[12]`: alternative: Source lists cooking salt / kosher salt; selected generic salt as proposal.
- BLOCKER `instruction_sections[0].steps[0]`: source-note: Note 5 is referenced but absent from JSON-LD.
