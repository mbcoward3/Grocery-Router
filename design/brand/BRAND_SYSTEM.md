# Grocery Router v1 brand system

Status: approved provisional v1 identity

## Mark

The v1 mark is Concept I: a basket handle above two grocery-list rows. It is intentionally
constructed from only three strokes. The upper row is wider than the lower row so the mark
suggests a tapered basket without adding an enclosing outline.

The mark is provisional in the sense that its geometry may receive optical corrections after
real use, but its basket/list concept and three-stroke construction are decided for v1.

## Palette

| Token | Value | Purpose |
| --- | --- | --- |
| Market teal | `#50BFA5` | Brand tile and primary actions |
| Market teal hover | `#62CCB2` | Primary-action hover |
| Market teal soft | `#8DDECB` | Eyebrows, links, and focus rings |
| Market ink | `#071A15` | Mark and content on teal |
| Canvas | `#0B0C0E` | Application background |
| Sidebar | `#0E1012` | Navigation surface |
| Panel | `#111316` | Primary cards and grouped content |
| Paper | `#F1F2F4` | Primary UI text |
| Sprout | `#69C39C` | Completed/success state only |

Teal replaces the purple prototype, the Anthropic-adjacent coral exploration, and a discarded
gold pass that made the three-stroke mark resemble a beehive. It remains fresh and household-
appropriate while supporting dark ink for accessible primary actions.

## Assets

- `web/public/brand/grocery-router-mark.svg`: primary mark on Market teal surfaces
- `web/public/brand/grocery-router-mark-mono.svg`: one-color/reversed mark
- `web/public/brand/grocery-router-lockup.svg`: horizontal dark-background lockup
- `web/public/favicon.svg`: optically adjusted 32-unit favicon in a Market teal tile
- `web/public/apple-touch-icon.svg`: 180-unit touch-icon source
- `web/public/apple-touch-icon.png`: rendered iOS touch icon
- `design/brand/logo-concept-i-three-shape.png`: approved exploration reference

## Usage

- Use the favicon drawing below 32 px; do not mechanically shrink the 64-unit mark.
- Preserve clear space of at least one stroke width around the mark.
- Use the full mark only for product identity, never as a generic basket or list action.
- Do not add a checkmark, route nodes, basket outline, gradients, shadows, or internal color.
- On teal, the mark is Market ink. On dark surfaces without a tile, it may be Paper or teal.
- Keep success green semantically separate from brand teal.

## Typography and iconography

The v1 wordmark uses the same Inter/system sans stack as the interface at weight 650–700.
Product icons use a 24 px grid, rounded joins and caps, and approximately 1.5–1.75 px strokes.
The brand mark is deliberately heavier and must not dictate ordinary icon stroke weight.
