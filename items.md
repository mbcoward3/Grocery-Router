# Canonical items

The normalization target for Step 2. See `docs/step2-design.md` §3.

One row per canonical item. Four jobs: **merge** synonyms so the same thing from two
recipes becomes one line, **convert** between units so cups and counts can be added,
**group** the list by aisle, and **flag** staples instead of dropping them.

Seeded from the four recipes in `recipes/`. It grows on parse failure — an unrecognized
item defaults to `other` / not-staple and gets reported, so a miss is visible rather than
silent. Adding a row is the fix, and it is meant to be routine.

`each_equiv` is how unit reconciliation actually works: per-item, not a general
conversion table. Only fill it in where a recipe somewhere expresses the item in more than
one unit.

| canonical | aisle | staple | each_equiv | synonyms |
|---|---|---|---|---|
| bell_pepper | produce | no | 1 ea = 1 cup sliced | green bell pepper, red bell pepper, bell peppers |
| onion | produce | no | 1 ea = 1 cup sliced | white onion, yellow onion, large white onion |
| garlic | produce | no | 1 head = 10 cloves | cloves garlic, garlic chopped, garlic minced |
| lime_juice | produce | no | 1 lime = 2 tbsp juice | lime juice |
| potato | produce | no | | small white potato, white potato |
| parsley | produce | no | 1 bunch = 8 tbsp chopped | fresh parsley |
| thyme | produce | no | | fresh thyme |
| rosemary | produce | no | | fresh rosemary |
| chuck_roast | meat | no | | beef chuck roast |
| chicken_breast | meat | no | | boneless skinless chicken breasts |
| italian_sausage | meat | no | | italian sausages, large italian sausages |
| salmon_fillet | seafood | no | | skinless salmon fillet |
| butter | dairy | no | 1 stick = 8 tbsp | unsalted butter |
| provolone | dairy | no | | provolone cheese slices |
| beef_broth | pantry | no | 1 can = 14.5 oz | canned beef broth |
| pepperoncini | pantry | no | | pepperoncini pepper slices |
| giardiniera | pantry | no | | chicago-style giardiniera |
| italian_dressing_mix | pantry | no | | envelope italian salad dressing mix |
| tortillas | bread | no | | flour tortillas, corn tortillas |
| buns | bread | no | | sandwich buns |
| vegetable_oil | pantry | yes | | |
| olive_oil | pantry | yes | | garlic olive oil |
| worcestershire | pantry | yes | | worcestershire sauce |
| chili_powder | pantry | yes | | |
| cumin | pantry | yes | | ground cumin |
| salt | pantry | yes | | |
| pepper | pantry | yes | | black pepper |

## Notes on the seed rows

**`salt, to taste`** is the case that proves the staple flag matters. It parses to a
quantity-less line and must never reach the list as an item to buy — but it also must not
be silently discarded, because the rule everywhere else is that dropping is worse than
flagging. Staples go to *probably have, check before you go*.

**`bell_pepper`** is the case that proves `each_equiv` matters. The fajitas ask for
`3 cups bell peppers, sliced` and the sausage and peppers asks for `1 green bell pepper`
plus `1 red bell pepper`. Without a conversion those are two incomparable lines; with one
they aggregate to 5 and the coupling between the two meals becomes visible.

**`garlic`** aggregates across three of the four recipes in different units — `4 cloves`,
`1 tbsp chopped`, `1 clove` — and still has to come out as *buy one head*.
