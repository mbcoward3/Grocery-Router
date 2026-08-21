# Grocery Router UI exploration brief

## Purpose

Explore several materially different executions before selecting the production frontend
composition. These are visual and interaction prototypes, not alternate product scopes.

## Product frame

Grocery Router has exactly three v1 screens:

1. Week
2. Groceries
3. Recipe Detail

The Week screen is an unordered recipe pool. It must never imply that recipes are assigned to
weekdays. The Groceries screen is a compact, store-section-grouped checklist. Recipe Detail
shows approved source information, ingredients, and ordered instructions.

## Primary reference

Linear is the primary visual and interaction reference, not a loose mood-board. Borrow
aggressively from its:

- compact density and clear hierarchy;
- typography and refined dark surfaces;
- subtle borders and restrained accent color;
- crisp interaction feedback;
- purposeful, restrained motion; and
- professional tone with little instructional copy.

Do not copy Linear's branding or import controls that do not serve this household workflow.
The result should feel purpose-built rather than themed.

## Required states and cues

### Week

- Intentional empty state with recipe-count selection and generation action
- Populated unordered recipe pool
- Add a specific recipe
- Remove an occurrence
- Random swap and specific swap
- Full regeneration
- Stable controls while mutations complete

### Groceries

- Alphabetical store-section groups
- Compact generated lines with readable exact quantities
- Large, one-hand-friendly completion targets
- Completed lines retained in place and de-emphasized
- Expandable recipe contribution provenance
- Manual week-only line
- Remove/restore and week-only quantity override cues

### Recipe Detail

- Recipe name and optional image
- Source, yield, hands-on time, and unattended time when present
- Ordered ingredient sections
- Ordered instruction sections and steps

## Responsive and accessibility requirements

- Desktop should be polished and information-dense.
- iPhone layouts must be intentionally composed, not scaled-down desktop.
- No essential hover-only controls or horizontal tables.
- Use semantic controls, visible focus, sufficient contrast, and reduced-motion support.
- Navigation between Week and Groceries should remain easy while shopping one-handed.

## Exploration axes

Produce three distinct concepts while retaining the common Linear-inspired direction:

- **A — Workspace:** dense sidebar shell and structured central canvas
- **B — Command center:** split-pane planning with stronger contextual actions
- **C — Mobile canvas:** stacked, touch-forward composition that expands elegantly on desktop

Use representative real-world recipe and grocery content. Each concept should be complete enough
to compare hierarchy, density, navigation, recipe actions, checklist behavior, and responsive
composition—not merely colors.

## Review criteria

Choose or combine concepts based on:

1. immediate comprehension of the current week;
2. speed and confidence of recipe mutations;
3. grocery-list usability on iPhone;
4. provenance clarity without default-view clutter;
5. fidelity to Linear's visual grammar;
6. accessibility and interaction stability; and
7. feasibility without a generalized design system.
