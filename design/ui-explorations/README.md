# Grocery Router UI explorations

Three standalone, dependency-free prototypes generated as an independent design pass. Open any
`.html` file directly in a browser; no server or build step is required. To switch concepts,
return to the file browser and open a different HTML file. Checked-in desktop and iPhone-sized
screenshots provide a quick static review:

| Concept | Desktop | Mobile |
| --- | --- | --- |
| A — Atlas | [`concept-a-desktop.png`](concept-a-desktop.png) | [`concept-a-mobile.png`](concept-a-mobile.png) |
| B — Relay | [`concept-b-desktop.png`](concept-b-desktop.png) | [`concept-b-mobile.png`](concept-b-mobile.png) |
| C — Stack | [`concept-c-desktop.png`](concept-c-desktop.png) | [`concept-c-mobile.png`](concept-c-mobile.png) |

## Concept A — Atlas (`concept-a.html`)

**Pattern:** Dense sidebar workspace

A conventional, highly legible desktop app shell with persistent screen navigation, a compact recipe table, grouped grocery sections, and a two-column recipe detail. On iPhone, the sidebar becomes a bottom tab bar and dense rows simplify without hiding essential actions.

**Strengths**
- Clearest information architecture and fastest screen-to-screen navigation
- Recipe pool supports add, individual remove/swap, whole-pool refresh, and an intentional empty/generation state
- Grocery provenance, editable quantities, manual lines, and completion are explicit but compact
- Most scalable foundation if more household tools are added later

**Tradeoff:** Familiar and highly usable, but the least distinctive of the three.

## Concept B — Relay (`concept-b.html`)

**Pattern:** Command-center split pane

A narrow icon rail, recipe index, and focused detail canvas make the week behave like a compact working set. Desktop prioritizes rapid scanning and selection; mobile intentionally changes to full-screen views rather than squeezing the split pane.

**Strengths**
- Best desktop recipe browsing: week context remains visible while reading a recipe
- Highest information density without feeling spreadsheet-like
- Search/filter and selection behavior make a larger recipe pool feel manageable
- Strongest expression of a keyboard-oriented, serious utility

**Tradeoff:** The persistent recipe index gives Recipe Detail more visual priority than the Week pool and is less useful for households with only a few recipes.

## Concept C — Stack (`concept-c.html`)

**Pattern:** Mobile-first stacked canvas

A card deck presents the week as a flexible set rather than a schedule. Desktop expands it into a spacious two-column canvas; mobile turns it into a thumb-friendly horizontal deck with persistent bottom navigation. A restrained lime accent differentiates action and progress states.

**Strengths**
- Strongest iPhone experience and clearest one-hand grocery completion
- Most distinctive visual identity while retaining compact dark surfaces and crisp feedback
- Recipe cards communicate “unordered pool” especially well
- Grocery progress and manual-item entry are easy to understand at a glance

**Tradeoff:** Horizontal card browsing is more expressive than efficient for large weekly pools.

## Selected direction

**Concept A (Atlas) is the production starting point.** Borrow Concept C’s strongest mobile
grocery interactions where they improve one-handed use, but preserve Atlas’s visual language
and information architecture rather than creating a hybrid aesthetic.

Atlas best matches the product’s small, fixed screen model: it is compact, immediately
understandable, and makes every key week action available without turning the pool into a
schedule. Its structure also provides a strong shell for the three fixed product screens.
Concept C remains a mobile interaction reference, especially for completion targets and card
simplification. Relay’s split-pane model overemphasizes recipe inspection for the likely pool
size.

## Interaction checklist

All concepts include:
- Week, Groceries, and Recipe Detail navigation
- Populated week pool plus a clear-to-empty generation state
- Add, remove, individual swap, and whole-pool regeneration cues
- Alphabetical store sections, persistent completed lines, editable quantity cues, expandable recipe provenance, and manual grocery lines
- Source, yield, timing, ordered ingredient sections, and ordered instructions
- Keyboard focus states, semantic controls, reduced-motion handling, and responsive desktop/iPhone layouts
