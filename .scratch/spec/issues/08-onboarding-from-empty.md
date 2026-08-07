# Onboarding a household from empty

Type: grilling
Status: open
Blocked by: 01

## Question

What does the tool ask a household that has nothing, and what does it refuse to work
without?

v1 serves one household whose data already exists (ticket 07), so this is easy to skip —
and skipping it is exactly how cold start silently rots, with a populated corpus sitting
right there tempting every decision. The spec must define it and the build must test it at
corpus size zero deliberately.

Three of these are correctness bugs the old repo carried, not features:

1. **Household composition replaces a compiled-in constant.** `BASE_AE = 2.5` was one
   family's size, commented *"2 adults, a 3-year-old, a 1-year-old"*, read in four places
   including both shopping-list paths. Every other household would get quantities scaled
   for somebody else's children. Composition moves onto the profile.
2. **Allergens generalise past one allergen.** The old code hardcoded peanut — a shellfish
   or gluten family would declare it and the system would enforce **nothing**, while
   looking like it handles allergies. Settle the blocking tier (a curated set with
   maintained term lists) and the warning tier (anything typed freehand, warns and never
   blocks). Nothing may ever claim a dish is *safe*; only that nothing was seen.
3. **"None" must be a recorded answer, not an empty section.** If *unanswered* and
   *actively said no allergies* look the same in the database, the gate bought nothing. A
   defaulted `no allergens` is the worst possible place in this product for a plausible
   value where there should be a gap.

Then settle:

- **What is gated and what degrades.** The house posture is *degrade, never block*: no
  quantities until composition is given, each shortfall visible with a link to the section
  that fixes it. **The prompt to finish onboarding is the product being visibly incomplete,
  not a banner.** Allergens are the one hard gate.
- **How the profile gets built from free text.** This is model job three. The old interview
  worked one question at a time, and that is why it produced the corrections that reshaped
  the design — including one the household volunteered unprompted. A batched form
  suppresses exactly that signal.
- **The safe first week.** With nothing proven, every proposal is a gamble, and two flops
  in week one loses a household. Propose fewer nights, lean on food that is hard to get
  wrong, and say so.
