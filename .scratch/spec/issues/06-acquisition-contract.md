# The acquisition agent contract

Type: grilling
Status: open
Blocked by: 01

## Question

How does the tool find a recipe nobody bookmarked, and what stops it inventing one?

Decision 11 replaced the old per-site search ladder with the Anthropic server-side web
search tool. Decision 12 keeps the guard rail that made the old one safe: **guess loose,
verify hard** — the finder only ever proposes URLs, and capture refuses any page carrying
no machine-readable schema.org recipe. The sloppiest possible search costs a wasted request
and can never cost a wrong ingredient.

Settle:

- **What a gap is.** The old version derived gaps from the week — a missing protein, a
  missing cuisine. With provenance and no ranker, what asks for a new recipe, and when?
- **The search surface.** Server-side web search can reach the whole internet. The old one
  could only reach sites the household already cooked from, which was a real trust signal.
  Does v1 keep an allowed-domains list, seed it from the corpus, or open it up?
- **The refusals, and what each one tells the household.** The old set: a declared
  allergen, a page with no machine-readable recipe, something already in the corpus, the
  wrong protein for the gap, and no identifiable protein at all.
- **The blunt gate.** That last refusal exists because full-text search returns cake — but
  it also refuses a genuinely vegetarian main. Does v1 fix it, or state the limit?
- **Pasted links are different.** Hard constraints still apply to something a person chose;
  relevance filters do not. A person choosing a link is not noise.
- **Where it lands.** Nothing acquired enters the week directly. It has to win a slot.
- **The tool loop.** Which Anthropic tools, what the agent returns, and what the code does
  with it. Read the `claude-api` skill first — the web search tool version and its options
  matter here.
