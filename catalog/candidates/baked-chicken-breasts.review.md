# Baked Chicken Breasts

- Candidate: `baked-chicken-breast#$/@graph[7]`
- Catalog key: `baked-chicken-breasts`
- State: `reviewable`
- Source: `baked-chicken-breast` <https://www.gimmesomeoven.com/baked-chicken-breast/>
- Selected JSON-LD: artifact `baked-chicken-breast.json`, recipe 0, script 0, path `$/@graph[7]`

## Blockers

- **BLOCKER:** ingredient_sections[0].ingredients[1]: alternative: Source lists melted butter or olive oil; selected butter as first-listed baseline.
- **BLOCKER:** ingredient_sections[0].ingredients[5].grocery_item: backfill: No exact existing grocery item for smoked paprika; proposed new Spices measured item.
- **BLOCKER:** instruction_sections[0].steps[2]: source-note: Asterisk after baking dish appears to reference a source note absent from JSON-LD.
- **BLOCKER:** instruction_sections[0].steps[3]: source-note: Asterisk after 15-18 minutes appears to reference a source note absent from JSON-LD.
- **BLOCKER:** source note * is unresolved
- **BLOCKER:** unattended: ambiguous: Cook time is baking and proposed as unattended, but optional broiling requires active monitoring.

## Summary

- Yield: 4 servings
- Hands on: 20 min
- Unattended: 15 min

## Ingredients

### Ingredients

- [0] 4  boneless skinless chicken breasts (pounded to even thickness)
  - Proposal: Chicken Breast; quantity: 4 each; grocery: match chicken-breast
- [1] 1 tablespoon melted butter or olive oil
  - Proposal: Butter; quantity: 1 tbsp; grocery: match butter
- [2] 1 teaspoon kosher salt
  - Proposal: Kosher Salt; quantity: 1 tsp; grocery: match kosher-salt
- [3] 1/2 teaspoon freshly-ground black pepper
  - Proposal: Black Pepper; quantity: 1/2 tsp; grocery: match black-pepper
- [4] 1/2 teaspoon garlic powder
  - Proposal: Garlic Powder; quantity: 1/2 tsp; grocery: match garlic-powder
- [5] 1/2 teaspoon smoked paprika
  - Proposal: Smoked Paprika; quantity: 1/2 tsp; grocery: create smoked-paprika

## Instructions

### Method

1. Fill a large mixing bowl with 2 cups of lukewarm water and 1/4 cup kosher salt. Stir to combine until most of the salt is absorbed. Add 2 cups of cold water (or a few ice cubes) to lower the temperature of the water so that it is cool to the touch. Add the chicken breasts and let them sit in the mixture to brine for 15 minutes, or you can also also cover the bowl and refrigerate for up to 6 hours. Remove the chicken breasts from the brine, rinse them with cold water, then pat them dry with paper towels.
2. Preheat oven to 450°F.
3. Place the chicken breasts in a single layer in a large baking dish*. Brush on both sides (turning once) evenly with the melted butter or olive oil. In a separate small bowl, whisk the salt, pepper, garlic powder and paprika until combined. Sprinkle the seasoning mixture evenly over the chicken on both sides.
4. Bake for 15-18* minutes, or until the chicken is cooked through and no longer pink. Cooking time will depend on the thickness of the chicken breasts, so I recommend using a cooking thermometer to know exactly when the chicken is fully cooked. The thickest part of the breast should measure 165°F. (If you want the chicken to be a little bit browned and crispier on top, you can turn the broiler on high for the final 3-5 minutes of the cooking time and broil the chicken until it is cooked through and golden on top. Keep a close eye on the chicken, however, so that it does not overcook and/or burn.)
5. Once the chicken is cooked, remove the pan from the oven, transfer the chicken to a clean plate, and loosely tent the plate with aluminum foil. Let the chicken rest for at least 5-10 minutes.
6. Serve warm and enjoy!

## Issues

- BLOCKER `ingredient_sections[0].ingredients[1]`: alternative: Source lists melted butter or olive oil; selected butter as first-listed baseline.
- BLOCKER `ingredient_sections[0].ingredients[5].grocery_item`: backfill: No exact existing grocery item for smoked paprika; proposed new Spices measured item.
- BLOCKER `instruction_sections[0].steps[2]`: source-note: Asterisk after baking dish appears to reference a source note absent from JSON-LD.
- BLOCKER `instruction_sections[0].steps[3]`: source-note: Asterisk after 15-18 minutes appears to reference a source note absent from JSON-LD.
- BLOCKER `unattended`: ambiguous: Cook time is baking and proposed as unattended, but optional broiling requires active monitoring.
