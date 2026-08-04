# Canonical items

The normalization target for Step 2. See `docs/step2-design.md` §3.

One row per canonical item. Four jobs: **merge** synonyms so the same thing from two
recipes becomes one line, **convert** between units so cups and counts can be added,
**group** the list by aisle, and **flag** staples instead of dropping them.

It grows on parse failure — `./shop.py --audit` parses every recipe and prints every item
name with no row here, so a miss is visible rather than silent. Adding a row is the fix,
and it is meant to be routine.

`each_equiv` is how unit reconciliation actually works: per-item, not a general conversion
table. Only fill it in where a recipe somewhere expresses the item in more than one unit.
Multiple clauses are separated by `;` and chain — `1 head = 10 cloves; 1 clove = 1 tsp`
is what lets a tablespoon of chopped garlic and four whole cloves add up to one head.

**Aisles are assigned here, not read from a store.** They order the list for walking and
nothing more. A wrong aisle costs a few steps; it never puts the wrong thing in the cart,
which is why this is the one field in the project allowed to be a considered guess. When
the Kroger adapter lands (§7), real aisle data can replace this column wholesale.

**Near-identical items are kept apart on purpose.** `potato`, `yukon_gold_potato` and
`russet_potato` are three rows, and a week wanting two of them gets two lines. Merging
them would be the system deciding a substitution nobody declared — the exact thing §2.3
forbids. If they *are* interchangeable for you, say so with `accepts:` on the recipe line
and the merge becomes visible and reversible.

| canonical | aisle | staple | each_equiv | synonyms |
|---|---|---|---|---|
| bell_pepper | produce | no | 1 ea = 1 cup sliced | green bell pepper, red bell pepper, bell peppers |
| onion | produce | no | 1 ea = 1 cup sliced | white onion, yellow onion, large white onion |
| green_onion | produce | no | | sliced green onion, green onions, scallion |
| red_onion | produce | no | | red onions |
| garlic | produce | no | 1 head = 10 cloves; 1 clove = 1 tsp | cloves garlic, garlic chopped, garlic minced |
| lime | produce | no | 1 lime = 2 tbsp | lime juice, juice of lime |
| lemon | produce | no | 1 lemon = 3 tbsp; 1 lemon = 8 slices | lemon juice, juice of lemon |
| potato | produce | no | | small white potato, white potato, white boiling potatoes |
| yukon_gold_potato | produce | no | | yukon gold potatoes |
| russet_potato | produce | no | | russet potatoes |
| carrot | produce | no | 1 ea = 2.5 oz | carrots, sliced carrots |
| celery | produce | no | 1 head = 10 ribs | rib celery, celery ribs |
| lettuce | produce | no | | |
| tomato | produce | no | | tomatoes |
| avocado | produce | no | | avocados |
| broccoli | produce | no | | broccoli florets |
| mushroom | produce | no | | mushrooms |
| kale | produce | no | | chopped kale |
| asparagus | produce | no | | |
| ginger | produce | no | | fresh ginger |
| parsley | produce | no | 1 bunch = 8 tbsp chopped | fresh parsley |
| thyme | produce | no | | fresh thyme |
| rosemary | produce | no | | fresh rosemary |
| chuck_roast | meat | no | | beef chuck roast, chuck beef roast, boneless beef chuck |
| ground_beef | meat | no | | lean ground beef, ground hamburger, hamburger |
| chicken_breast | meat | no | | boneless skinless chicken breasts |
| chicken_thigh | meat | no | | boneless skinless chicken thighs, chicken thighs |
| whole_chicken | meat | no | | young chicken |
| rotisserie_chicken | meat | no | | |
| cooked_chicken | meat | no | | cooked and shredded chicken, canned chicken |
| italian_sausage | meat | no | | italian sausages, large italian sausages |
| sausage_tube | meat | no | | sausage tube spicy or regular, breakfast sausage |
| bacon | meat | no | | |
| pork_loin | meat | no | | |
| salmon_fillet | seafood | no | | skinless salmon fillet |
| canned_tuna | seafood | no | | spicy yellowfin tuna, tuna |
| butter | dairy | no | 1 stick = 8 tbsp | unsalted butter, garlic butter |
| milk | dairy | no | | |
| egg | dairy | no | | eggs |
| provolone | dairy | no | | provolone cheese slices |
| mozzarella | dairy | no | | mozzarella cheese |
| cheddar | dairy | no | | cheddar cheese, shredded cheddar cheese, medium cheddar, sliced cheddar |
| colby_jack | dairy | no | | colby jack cheese |
| mexican_cheese_blend | dairy | no | | four cheese mexican, borden® cheese thick cut shredded four cheese mexican |
| shredded_cheese | dairy | no | | |
| cheese | dairy | no | | cheese of choice |
| cream_cheese | dairy | no | 1 brick = 8 oz | cream cheese block, bricks cream cheese |
| sour_cream | dairy | no | | |
| refrigerated_biscuits | dairy | no | | tube biscuits, biscuits, refrigerated biscuits cut into quarters |
| buns | bread | no | | bun, sandwich buns |
| tortilla | bread | no | | flour tortillas, corn tortillas, tortillas |
| bread | bread | no | | bread of choice |
| hoagie_roll | bread | no | | hoagie rolls, submarine buns |
| dinner_roll | bread | no | | dinner rolls |
| panko | bread | no | | panko breadcrumbs, seasoned bread crumbs, bread crumbs, gluten-free bread crumbs |
| frozen_meatballs | frozen | no | | |
| peas | frozen | no | | frozen peas |
| beef_broth | pantry | no | 1 can = 14.5 oz; 1 oz = 1 floz | canned beef broth |
| chicken_broth | pantry | no | 1 can = 14.5 oz; 1 oz = 1 floz | canned chicken broth |
| cream_of_chicken_soup | pantry | no | 1 can = 10.5 oz | |
| coconut_milk | pantry | no | 1 can = 14 oz; 1 oz = 1 floz | |
| pepperoncini | pantry | no | | pepperoncini pepper slices, pepperocinis |
| giardiniera | pantry | no | | chicago-style giardiniera |
| pickles | pantry | no | | pickle slices, boar's head pickle slices, pickle juice |
| italian_dressing_mix | pantry | no | | italian salad dressing mix, italian dressing mix |
| onion_soup_mix | pantry | no | | onion soup mix lipton's, lipton onion soup mix |
| ranch_seasoning | pantry | no | | |
| rotisserie_seasoning | pantry | no | | |
| taco_seasoning | pantry | no | | la preferida taco seasoning |
| taco_sauce | pantry | no | | el paso mild taco sauce, el paso hot taco sauce |
| enchilada_sauce | pantry | no | | red enchilada sauce |
| salsa | pantry | no | | chunky salsa |
| marinara | pantry | no | | marinara sauce, pasta sauce, pasta sauce or marinara sauce |
| tomato_sauce | pantry | no | 1 can = 8 oz; 1 oz = 1 floz | |
| tomato_paste | pantry | no | | |
| diced_tomatoes | pantry | no | | petite diced tomatoes |
| rotel | pantry | no | | rotel undrained |
| chili_beans | pantry | no | | |
| black_beans | pantry | no | | |
| pinto_beans | pantry | no | | |
| corn | pantry | no | | canned corn |
| elbow_noodles | pantry | no | | elbow noodles box, elbow macaroni |
| ramen_noodles | pantry | no | | maruchan ramen noodles |
| white_rice | pantry | no | | rice |
| soy_sauce | pantry | no | | reduced sodium soy sauce |
| sesame_oil | pantry | no | | |
| honey | pantry | no | | |
| red_wine | pantry | no | | dry red wine |
| vegetable_oil | pantry | yes | | oil, canola oil |
| olive_oil | pantry | yes | | garlic olive oil |
| worcestershire | pantry | yes | | worcestershire sauce |
| ketchup | pantry | yes | | |
| mustard | pantry | yes | | yellow mustard |
| mayo | pantry | yes | | mayonnaise |
| flour | pantry | yes | | all-purpose flour |
| cornstarch | pantry | yes | | corn starch |
| sugar | pantry | yes | | |
| brown_sugar | pantry | yes | | |
| balsamic_vinegar | pantry | yes | | |
| white_vinegar | pantry | yes | | |
| chili_powder | pantry | yes | | |
| cumin | pantry | yes | | ground cumin |
| paprika | pantry | yes | | |
| oregano | pantry | yes | | dried oregano |
| basil | pantry | yes | | dried basil |
| dried_thyme | pantry | yes | | |
| dried_parsley | pantry | yes | | |
| italian_seasoning | pantry | yes | | |
| bay_leaf | pantry | yes | | |
| celery_seed | pantry | yes | | |
| sesame_seeds | pantry | yes | | |
| garlic_powder | pantry | yes | | |
| garlic_salt | pantry | yes | | |
| onion_powder | pantry | yes | | |
| crushed_red_pepper | pantry | yes | | red pepper flakes |
| salt | pantry | yes | | kosher salt |
| pepper | pantry | yes | | black pepper, coarse ground black pepper |
| water | pantry | yes | | cold water |

## Notes on the rows that earned their place

**`salt, to taste`** is the case that proves the staple flag matters. It parses to a
quantity-less line and must never reach the list as an item to buy — but it also must not
be silently discarded, because the rule everywhere else is that dropping is worse than
flagging. Staples go to *probably have, check before you go*.

**`bell_pepper`** is the case that proves `each_equiv` matters. The fajitas ask for
`3 cups bell peppers, sliced` and the sausage and peppers asks for `1 green bell pepper`
plus `1 red bell pepper`. Without a conversion those are two incomparable lines; with one
they aggregate to 5 and the coupling between the two meals becomes visible.

**`garlic`** aggregates across three recipes in three different units — `4 cloves`,
`1 tbsp chopped`, `1 clove` — and still has to come out as *buy one head*. It takes two
chained `each_equiv` clauses to get there, which is why the field allows chaining at all.

**`garlic_powder`, `garlic_salt`, `onion_powder`, `onion_soup_mix`, `green_onion`,
`celery_seed`** exist because of a bug they caused. The normalizer falls back to probing
sub-phrases of an item name, and `onion powder` matched `onion` — putting a fresh onion in
the cart for a teaspoon of spice, silently, across thirteen lines of the corpus. A partial
match is now only accepted when every word it leaves behind is noise, so these surface as
unknown items until a row exists. **A mis-merge is worse than an unknown line**, because
an unknown line gets printed and a mis-merge does not.

**`dried_thyme` and `dried_parsley`** are the same bug found a second time. The normalizer
was treating `dried` and `fresh` as noise words, so `1 teaspoon dried thyme` resolved to
the produce-aisle `thyme` in four recipes — a bunch of fresh herbs bought for a teaspoon of
spice. Neither word is noise: they name which aisle you walk to. They are out of the
stopword list, and every real pair gets two rows.

**`pickles`** absorbs `pickle juice` deliberately: the juice comes out of the jar you were
already buying, so one line is right and two would be wrong.
