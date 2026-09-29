# Banana Bread

- Candidate: `love-and-lemons-banana-bread#$/@graph[7]`
- Catalog key: `banana-bread`
- State: `reviewable`
- Source: `love-and-lemons-banana-bread` <https://www.loveandlemons.com/banana-bread/>
- Selected JSON-LD: artifact `love-and-lemons-banana-bread.json`, recipe 0, script 0, path `$/@graph[7]`

## Blockers

- **BLOCKER:** $.ingredient_sections[0].ingredients[0].grocery_item: backfill: Proposed new grocery item for banana.
- **BLOCKER:** $.ingredient_sections[0].ingredients[10].grocery_item: backfill: Proposed new grocery item for walnuts.
- **BLOCKER:** $.ingredient_sections[0].ingredients[1].grocery_item: backfill: Proposed new grocery item for cane sugar.
- **BLOCKER:** $.ingredient_sections[0].ingredients[1]: alternative: Source lists cane sugar or brown sugar; selected cane sugar as first-listed baseline.
- **BLOCKER:** $.ingredient_sections[0].ingredients[2]: alternative: Source lists melted butter or vegetable oil; selected butter as first-listed baseline.
- **BLOCKER:** $.ingredient_sections[0].ingredients[4].grocery_item: backfill: Proposed new grocery item for vanilla extract.
- **BLOCKER:** $.ingredient_sections[0].ingredients[6].grocery_item: backfill: Proposed new grocery item for baking soda.
- **BLOCKER:** $.ingredient_sections[0].ingredients[8].grocery_item: backfill: Proposed new grocery item for cinnamon.
- **BLOCKER:** $.ingredient_sections[0].ingredients[9].grocery_item: backfill: Proposed new grocery item for nutmeg.

## Summary

- Yield: 8
- Hands on: 15 min
- Unattended: 55 min

## Ingredients

### Ingredients

- [0] 2 cups mashed very ripe banana (about 4 large)
  - Proposal: Banana; quantity: 2 cup; grocery: create banana
- [1] ½ cup cane sugar or brown sugar
  - Proposal: Cane Sugar; quantity: 1/2 cup; grocery: create cane-sugar
- [2] ½ cup melted butter or vegetable oil (plus more for the pan)
  - Proposal: Butter; quantity: 1/2 cup; grocery: match butter
- [3] 2  large eggs
  - Proposal: Eggs; quantity: 2 each; grocery: match eggs
- [4] 1 teaspoon vanilla extract
  - Proposal: Vanilla Extract; quantity: 1 tsp; grocery: create vanilla-extract
- [5] 1½ cups all-purpose flour (spooned and leveled)
  - Proposal: All-Purpose Flour; quantity: 1 1/2 cup; grocery: match all-purpose-flour
- [6] 1 teaspoon baking soda
  - Proposal: Baking Soda; quantity: 1 tsp; grocery: create baking-soda
- [7] ½ teaspoon sea salt
  - Proposal: Salt; quantity: 1/2 tsp; grocery: match salt
- [8] ½ teaspoon cinnamon
  - Proposal: Cinnamon; quantity: 1/2 tsp; grocery: create cinnamon
- [9] ¼ teaspoon nutmeg
  - Proposal: Nutmeg; quantity: 1/4 tsp; grocery: create nutmeg
- [10] ½ cup chopped walnuts (plus 2 tablespoons for topping)
  - Proposal: Walnuts; quantity: 1/2 cup; grocery: create walnuts

## Instructions

### Method

1. Preheat the oven to 350°F and grease an 8x4 or 9x5-inch loaf pan.
2. In a large bowl, whisk together the mashed banana, sugar, butter, eggs, and vanilla.
3. In a medium bowl, whisk together the flour, baking soda, salt, cinnamon, and nutmeg.
4. Add the dry ingredients to the wet ingredients and stir until just combined. Don’t overmix. Fold in the ½ cup walnuts.
5. Pour the batter into the prepared pan and top with the remaining 2 tablespoons walnuts.
6. Bake for 50 to 60 minutes, or until a toothpick inserted comes out clean and the top springs back to the touch. I like to check the loaf after 40 minutes. If the top is golden brown, I cover it with foil for the remaining bake time to prevent further browning.

## Issues

- BLOCKER `$.ingredient_sections[0].ingredients[0].grocery_item`: backfill: Proposed new grocery item for banana.
- BLOCKER `$.ingredient_sections[0].ingredients[1]`: alternative: Source lists cane sugar or brown sugar; selected cane sugar as first-listed baseline.
- BLOCKER `$.ingredient_sections[0].ingredients[1].grocery_item`: backfill: Proposed new grocery item for cane sugar.
- BLOCKER `$.ingredient_sections[0].ingredients[2]`: alternative: Source lists melted butter or vegetable oil; selected butter as first-listed baseline.
- BLOCKER `$.ingredient_sections[0].ingredients[4].grocery_item`: backfill: Proposed new grocery item for vanilla extract.
- BLOCKER `$.ingredient_sections[0].ingredients[6].grocery_item`: backfill: Proposed new grocery item for baking soda.
- BLOCKER `$.ingredient_sections[0].ingredients[8].grocery_item`: backfill: Proposed new grocery item for cinnamon.
- BLOCKER `$.ingredient_sections[0].ingredients[9].grocery_item`: backfill: Proposed new grocery item for nutmeg.
- BLOCKER `$.ingredient_sections[0].ingredients[10].grocery_item`: backfill: Proposed new grocery item for walnuts.
