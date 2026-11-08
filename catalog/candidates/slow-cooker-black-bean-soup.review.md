# Slow Cooker Black Bean Soup

- Candidate: `slow-cooker-black-bean-soup#$/@graph[7]`
- Catalog key: `slow-cooker-black-bean-soup`
- State: `reviewable`
- Source: `slow-cooker-black-bean-soup` <https://www.budgetbytes.com/slow-cooker-black-bean-soup/>
- Selected JSON-LD: artifact `slow-cooker-black-bean-soup.json`, recipe 0, script 0, path `$/@graph[7]`

## Blockers

- **BLOCKER:** ingredient_sections[0].ingredients[6]: source-note: Ingredient references * note, but note body is absent from JSON-LD.
- **BLOCKER:** ingredient_sections[0].ingredients[9].grocery_item: backfill: No exact existing grocery item for vegetable broth; proposed new pantry measured item.
- **BLOCKER:** instruction_sections[0].steps[2]: source-note: Step references ** note, but note body is absent from JSON-LD.
- **BLOCKER:** source note * is unresolved
- **BLOCKER:** source note ** is unresolved

## Summary

- Yield: 6; 6 1.5 cups each
- Hands on: 15 min
- Unattended: 360 min

## Ingredients

### Ingredients

- [0] 2 cloves garlic ($0.16)
  - Proposal: Garlic; quantity: 2 clove; grocery: match garlic
- [1] 1  yellow onion ($0.41)
  - Proposal: Yellow Onion; quantity: 1 each; grocery: match yellow-onion
- [2] 2 ribs celery ($0.33)
  - Proposal: Celery; quantity: 2 each; grocery: match celery
- [3] 2  carrots ($0.28)
  - Proposal: Carrots; quantity: 2 each; grocery: match carrots
- [4] 1 lb. black beans (uncooked) ($1.75)
  - Proposal: Black Beans; quantity: 1 lb; grocery: match black-beans
- [5] 1 cup salsa ($0.85)
  - Proposal: Chunky Salsa; quantity: 1 cup; grocery: match chunky-salsa
- [6] 1 Tbsp chili powder* ($0.30)
  - Proposal: Chili Powder; quantity: 1 tbsp; grocery: match chili-powder
- [7] 1/2 Tbsp ground cumin ($0.15)
  - Proposal: Ground Cumin; quantity: 1/2 tbsp; grocery: match ground-cumin
- [8] 1 tsp dried oregano ($0.05)
  - Proposal: Dried Oregano; quantity: 1 tsp; grocery: match dried-oregano
- [9] 4 cups vegetable broth ($0.53)
  - Proposal: Vegetable Broth; quantity: 4 cup; grocery: create vegetable-broth
- [10] 2 cups water ($0.00) _( non-shopping )_
  - Proposal: Water; quantity: 2 cup; grocery: match water

## Instructions

### Method

1. Mince the garlic, dice the onion and celery, and grate the carrots on a large holed cheese grater. Rinse the black beans in a colander under cool running water and pick out any stones or debris.
2. Combine the garlic, onion, celery, carrots, black beans, salsa, chili powder, cumin, oregano, vegetable broth, and water in a 5-7 quart slow cooker. Stir well.
3. Place the lid on the slow cooker and cook on high for 6-8 hours (you want the beans to get VERY soft). Once the beans are very soft, use an immersion blender** to blend the soup until it is thick and creamy (leave some beans whole if desired). Taste the soup and add salt if needed (this will depend on the brand of vegetable broth used).

## Issues

- BLOCKER `ingredient_sections[0].ingredients[6]`: source-note: Ingredient references * note, but note body is absent from JSON-LD.
- BLOCKER `instruction_sections[0].steps[2]`: source-note: Step references ** note, but note body is absent from JSON-LD.
- BLOCKER `ingredient_sections[0].ingredients[9].grocery_item`: backfill: No exact existing grocery item for vegetable broth; proposed new pantry measured item.
