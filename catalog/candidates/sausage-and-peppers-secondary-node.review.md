# Sausage and Peppers Recipe

- Candidate: `sausage-and-peppers#$`
- Catalog key: `sausage-and-peppers-secondary-node`
- State: `reviewable`
- Source: `sausage-and-peppers` <https://chefjeanpierre.com/recipes/sausage-and-peppers/>
- Selected JSON-LD: artifact `sausage-and-peppers.json`, recipe 1, script 1, path `$`

## Blockers

- **BLOCKER:** hands_on: unresolved: No Schema.org prepTime or cookTime durations provided.
- **BLOCKER:** ingredient_sections[0].ingredients[1].quantity: unresolved: Olive oil quantity is not specified.
- **BLOCKER:** ingredient_sections[0].ingredients[1].source_text: alternative: Source mentions garlic olive oil as chef-used alternative; baseline proposed as olive oil.
- **BLOCKER:** ingredient_sections[0].ingredients[5].grocery_item: ambiguous: Source combines thyme and rosemary in one ingredient; proposed first-listed herb only while preserving combined source text.
- **BLOCKER:** ingredient_sections[0].ingredients[5].source_text: alternative: Source says if dry use half; proposed fresh baseline.
- **BLOCKER:** key: ambiguous: The page exposed a second Recipe node with the same display name; review whether this is duplicate evidence.
- **BLOCKER:** unattended: unresolved: No Schema.org prepTime or cookTime durations provided; cooling time appears in instructions but no total duration is provided.

## Summary

- Yield: 4 Servings
- Hands on: (unresolved)
- Unattended: (unresolved)

## Ingredients

### Ingredients

- [0] 2  large Italian Sausages
  - Proposal: Italian Sausage; quantity: 2 each; grocery: match italian-sausage
- [1] Olive Oil, the Chef uses Garlic Olive Oil
  - Proposal: Olive Oil; quantity: unspecified; grocery: match olive-oil
- [2] 1 cup Onion sliced
  - Proposal: Onion; quantity: 1 cup; grocery: match onion
- [3] 1  Green Bell Pepper sliced
  - Proposal: Green Bell Pepper; quantity: 1 each; grocery: match green-bell-pepper
- [4] 1  Red Bell Pepper sliced
  - Proposal: Red Bell Pepper; quantity: 1 each; grocery: match red-bell-pepper
- [5] 2 teaspoons of Thyme &amp; Rosemary freshly chopped, if dry use half
  - Proposal: Fresh Thyme; quantity: 2 tsp; grocery: match fresh-thyme
- [6] 1 tablespoon Garlic chopped
  - Proposal: Garlic; quantity: 1 tbsp; grocery: match garlic

## Instructions

### Method

1. Steam the sausage in a frying pan with a cover like the chef did in the video. Let cool at least one hour to let the pork fat congeal so that when you cut it on your cutting board you do not lose all the juices.

### When ready to Eat

1. In a frying pan add some oil when it is 365ºF / 185ºC add onion and cook until light golden brown, cut the sausage into bite size and add to the onion. When the sausage starts getting golden brown, add your peppers add the herbs and salt and pepper to taste. Cook until the peppers are tender.Just before removing from the heat add the garlic and sauté for one minute.

## Issues

- BLOCKER `ingredient_sections[0].ingredients[1].quantity`: unresolved: Olive oil quantity is not specified.
- BLOCKER `ingredient_sections[0].ingredients[1].source_text`: alternative: Source mentions garlic olive oil as chef-used alternative; baseline proposed as olive oil.
- BLOCKER `ingredient_sections[0].ingredients[5].grocery_item`: ambiguous: Source combines thyme and rosemary in one ingredient; proposed first-listed herb only while preserving combined source text.
- BLOCKER `ingredient_sections[0].ingredients[5].source_text`: alternative: Source says if dry use half; proposed fresh baseline.
- BLOCKER `hands_on`: unresolved: No Schema.org prepTime or cookTime durations provided.
- BLOCKER `unattended`: unresolved: No Schema.org prepTime or cookTime durations provided; cooling time appears in instructions but no total duration is provided.
- BLOCKER `key`: ambiguous: The page exposed a second Recipe node with the same display name; review whether this is duplicate evidence.
