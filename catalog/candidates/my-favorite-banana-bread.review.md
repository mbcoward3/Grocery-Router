# My Favorite Banana Bread

- Candidate: `banana-bread#$/@graph[7]`
- Catalog key: `my-favorite-banana-bread`
- State: `reviewable`
- Source: `banana-bread` <https://sallysbakingaddiction.com/best-banana-bread-recipe/>
- Selected JSON-LD: artifact `banana-bread.json`, recipe 0, script 0, path `$/@graph[7]`

## Blockers

- **BLOCKER:** ingredient_sections[0].ingredients[10].grocery_item: backfill: Proposed new grocery item for chopped pecans.
- **BLOCKER:** ingredient_sections[0].ingredients[10]: alternative: Source allows pecans, walnuts, or semi-sweet chocolate chips; selected first-listed baseline.
- **BLOCKER:** ingredient_sections[0].ingredients[1].grocery_item: backfill: Proposed new grocery item for baking soda.
- **BLOCKER:** ingredient_sections[0].ingredients[3].grocery_item: backfill: Proposed new grocery item for ground cinnamon.
- **BLOCKER:** ingredient_sections[0].ingredients[5]: alternative: Source allows light or dark brown sugar; retained existing brown sugar item.
- **BLOCKER:** ingredient_sections[0].ingredients[7].grocery_item: backfill: Proposed new grocery item for bananas.
- **BLOCKER:** ingredient_sections[0].ingredients[8].grocery_item: backfill: Proposed new grocery item for plain Greek yogurt.
- **BLOCKER:** ingredient_sections[0].ingredients[8]: alternative: Source allows plain Greek yogurt or full-fat sour cream; selected first-listed baseline.
- **BLOCKER:** ingredient_sections[0].ingredients[9].grocery_item: backfill: Proposed new grocery item for pure vanilla extract.
- **BLOCKER:** unattended: source-note: Cook time is PT1H5M and baking is clearly passive; cooling/rest time appears in total time but is not represented as cookTime.

## Summary

- Yield: 1 loaf
- Hands on: 10 min
- Unattended: 65 min

## Ingredients

### Ingredients

- [0] 2 cups (250g) all-purpose flour (spooned &amp; leveled)
  - Proposal: All-Purpose Flour; quantity: 2 cup; grocery: match all-purpose-flour
- [1] 1 teaspoon baking soda
  - Proposal: Baking Soda; quantity: 1 tsp; grocery: create baking-soda
- [2] 1/4 teaspoon salt
  - Proposal: Salt; quantity: 1/4 tsp; grocery: match salt
- [3] 1/2 teaspoon ground cinnamon
  - Proposal: Ground Cinnamon; quantity: 1/2 tsp; grocery: create ground-cinnamon
- [4] 1/2 cup (8 Tbsp; 113g) unsalted butter, softened to room temperature
  - Proposal: Unsalted Butter; quantity: 1/2 cup; grocery: match unsalted-butter
- [5] 3/4 cup (150g) packed light or dark brown sugar
  - Proposal: Brown Sugar; quantity: 3/4 cup; grocery: match brown-sugar
- [6] 2 large eggs, at room temperature
  - Proposal: Eggs; quantity: 2 each; grocery: match eggs
- [7] 1 and 1/2 cups (345g) mashed bananas (about 3–4 ripe bananas)
  - Proposal: Bananas; quantity: 1 1/2 cup; grocery: create bananas
- [8] 1/3 cup (80g) plain Greek yogurt or full-fat sour cream, at room temperature
  - Proposal: Plain Greek Yogurt; quantity: 1/3 cup; grocery: create plain-greek-yogurt
- [9] 1 teaspoon pure vanilla extract
  - Proposal: Pure Vanilla Extract; quantity: 1 tsp; grocery: create pure-vanilla-extract
- [10] optional: 3/4 cup (90g) chopped pecans or walnuts, or 1 cup (180g) semi-sweet chocolate chips _( optional )_
  - Proposal: Chopped Pecans; quantity: 3/4 cup; grocery: create chopped-pecans

## Instructions

### Method

1. Adjust the oven rack to the lower third position and preheat the oven to 350°F (177°C). Lowering the oven rack prevents the top of your bread from browning too much, too soon. Grease a 9×5-inch loaf pan with nonstick spray. Set aside.
2. In a medium bowl, whisk the flour, baking soda, salt, and cinnamon together. Set aside.
3. In a large bowl using a handheld or stand mixer fitted with a paddle attachment, beat the butter and brown sugar together on medium-high speed until light and creamy, about 3 minutes. (Here’s a helpful tutorial if you need guidance on how to cream butter and sugar.) With the mixer running on medium speed, add the eggs one at a time, beating well after each addition. Scrape down the sides of the bowl as needed. Beat in the mashed bananas, yogurt/sour cream, and vanilla until combined.
4. Add the dry ingredients into the wet ingredients and beat on low speed just until combined. Do not over-mix. Fold in the nuts/chocolate chips, if using. The batter should be thick.
5. Pour and spread the batter into the prepared baking pan. Bake for 60–65 minutes, making sure to loosely cover the pan with aluminum foil halfway through, to prevent the top from getting too brown. The bread is done when a toothpick inserted in the center comes out clean with only a few small moist crumbs. Cool the bread in the pan set on a cooling rack for 1 hour. Remove the bread from the pan and place it directly on the rack to cool completely before slicing and serving.
6. Store wrapped tightly at room temperature for up to 3 days or in the refrigerator for up to 1 week.

## Issues

- WARNING `ingredient_sections[0].ingredients[0].quantity`: dual-unit: Source includes cups and grams; used first source measure.
- BLOCKER `ingredient_sections[0].ingredients[1].grocery_item`: backfill: Proposed new grocery item for baking soda.
- BLOCKER `ingredient_sections[0].ingredients[3].grocery_item`: backfill: Proposed new grocery item for ground cinnamon.
- WARNING `ingredient_sections[0].ingredients[4].quantity`: dual-unit: Source includes cups, tablespoons, and grams; used first source measure.
- BLOCKER `ingredient_sections[0].ingredients[5]`: alternative: Source allows light or dark brown sugar; retained existing brown sugar item.
- WARNING `ingredient_sections[0].ingredients[5].quantity`: dual-unit: Source includes cups and grams; used first source measure.
- BLOCKER `ingredient_sections[0].ingredients[7].grocery_item`: backfill: Proposed new grocery item for bananas.
- WARNING `ingredient_sections[0].ingredients[7].quantity`: dual-unit: Source includes cups, grams, and approximate count range; used first source measure.
- BLOCKER `ingredient_sections[0].ingredients[8]`: alternative: Source allows plain Greek yogurt or full-fat sour cream; selected first-listed baseline.
- BLOCKER `ingredient_sections[0].ingredients[8].grocery_item`: backfill: Proposed new grocery item for plain Greek yogurt.
- WARNING `ingredient_sections[0].ingredients[8].quantity`: dual-unit: Source includes cups and grams; used first source measure.
- BLOCKER `ingredient_sections[0].ingredients[9].grocery_item`: backfill: Proposed new grocery item for pure vanilla extract.
- BLOCKER `ingredient_sections[0].ingredients[10]`: alternative: Source allows pecans, walnuts, or semi-sweet chocolate chips; selected first-listed baseline.
- BLOCKER `ingredient_sections[0].ingredients[10].grocery_item`: backfill: Proposed new grocery item for chopped pecans.
- WARNING `ingredient_sections[0].ingredients[10].quantity`: dual-unit: Source includes cups and grams for alternatives; used first source measure.
- BLOCKER `unattended`: source-note: Cook time is PT1H5M and baking is clearly passive; cooling/rest time appears in total time but is not represented as cookTime.
