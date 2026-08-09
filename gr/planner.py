"""The one model call, and the code that refuses to trust it.

**The model picks meals. The model never produces a line of the shopping list.**

That is not a convention here, it is a process boundary. The planner subprocess launches
with `--tools ""`, which removes its file access entirely, so it cannot open a recipe file
even if the prompt begged it to. The prompt itself carries `corpus.md`, `profile.md`,
`candidates.md` and `sides.md` — and none of those holds an ingredient.

Three bugs this arrangement prevents, all of them real:

- **Invented ingredient coupling.** A model was once asked to show which meals shared
  ingredients, from an index containing no ingredients. It invented the coupling, then
  picked a recipe because the invention justified it. Coupling is now computed in
  `gr.shoplist` from real lines, and the provenance on every list row is that computation
  printed.
- **`onion powder` → `onion`.** Resolution is code, in `gr.items`, under a rule with
  sixteen tests behind it.
- **Invented staleness.** No last-cooked date exists in this repository. `_drop_recency`
  below removes any claim about one, whatever the model wrote.

Everything the model returns passes through `check()` before it reaches a list.
"""

from __future__ import annotations

import json
import re
import subprocess
from dataclasses import dataclass, field

from .repo import CorpusRow, Repo
from .shoplist import MealPlan

MODEL = "sonnet"
TIMEOUT_S = 300

# The planner's contract. The CLI validates the model's answer against this and forces a
# tool call, so a malformed answer is the CLI's problem rather than ours.
SCHEMA = {
    "type": "object",
    "properties": {
        "picks": {
            "type": "array",
            "items": {
                "type": "object",
                "properties": {
                    "slug": {"type": "string"},
                    "title": {"type": "string"},
                    "reason_kind": {
                        "type": "string",
                        # The kind set is closed. It has one entry per thing the planner
                        # can actually notice.
                        "enum": ["stale", "never", "protein", "cuisine", "yield",
                                 "passive", "low", "acquire", "plain"],
                    },
                    "reason": {"type": "string"},
                },
                "required": ["slug", "title", "reason_kind", "reason"],
                "additionalProperties": False,
            },
        }
    },
    "required": ["picks"],
    "additionalProperties": False,
}

# Any of these in a reason is a claim about when something was last cooked. No such date
# exists in this repository, so the claim is removed rather than believed.
_RECENCY = re.compile(
    r"\b("
    r"last (cooked|made|time|served|week|month|year)"
    r"|(haven'?t|hasn'?t|have not|has not|hadn'?t|had not)\s+(been\s+)?"
    r"(made|cooked|had|served|eaten|seen)"
    r"|not\s+(been\s+)?(made|cooked|served)\s+(since|in|for)"
    r"|since\s+(you|we|they|the household)\s+(made|cooked|had|last)"
    r"|(days?|weeks?|months?|years?)\s+ago"
    r"|in\s+(the\s+)?(last\s+)?(few\s+)?(weeks|months|years)\b"
    r"|in\s+(a\s+while|ages)"
    r"|been\s+(a\s+while|ages|too long|so long)"
    r"|recently|lately|overdue|stale|dormant|neglected"
    r")\b", re.I)

_PEANUT = re.compile(r"\bpeanut", re.I)


@dataclass
class PlannerResult:
    meals: list[MealPlan] = field(default_factory=list)
    dropped: list[tuple[str, str]] = field(default_factory=list)   # (what, why)
    notes: list[str] = field(default_factory=list)
    source: str = "planner"      # planner | code
    error: str = ""
    cost_usd: float | None = None
    duration_ms: int | None = None


# --- the prompt ------------------------------------------------------------

def _rows_block(rows: list[CorpusRow], untried: bool) -> str:
    lines = []
    for r in rows:
        mark = " [candidate]" if untried else ""
        lines.append(
            f"- slug: {r.slug} | {r.title}{mark} | protein: {r.protein} | "
            f"cuisine: {r.cuisine} | yield: {r.yield_raw} | "
            f"active: {r.active} | passive: {r.passive}"
            + (f" | notes: {r.notes}" if r.notes else "")
        )
    return "\n".join(lines)


def build_prompt(repo: Repo, nights: int, guests: int,
                 last_week: str = "", avoid: list[str] | None = None) -> str:
    """Assemble the planner's whole world. **No ingredient enters this string.**"""
    profile = (repo.root / "profile.md")
    profile_text = profile.read_text(encoding="utf-8") if profile.exists() else ""
    sides_note = (
        f"{len(repo.sides)} sides are recorded."
        if repo.sides else
        "NO sides are recorded at all. sides.md is empty on purpose. Do not propose a "
        "side, do not name a vegetable, and do not suggest what to serve alongside "
        "anything. The household knows the lists run short and will type its sides in."
    )
    avoid_block = ""
    if avoid:
        avoid_block = (
            "\nDo not pick these slugs this time — the household asked for something "
            "else:\n" + "\n".join(f"- {s}" for s in avoid) + "\n"
        )

    return f"""You plan one week of dinners for one household. Pick {nights} meals.

## The household's profile

{profile_text}

## The corpus — recipes this household has cooked and liked

{_rows_block(repo.corpus, untried=False)}

## Candidates — NOT yet cooked here. Mark these clearly.

{_rows_block(repo.candidates, untried=True)}

## Sides

{sides_note}

## Last week

{last_week or "No previous week file exists."}
{avoid_block}
## What to do

Pick exactly {nights} meals for a week feeding about {repo.target_ae(guests):.1f}
adult-equivalents ({guests} guest(s) included). Return them as `picks`.

**Use the `slug` values exactly as written above.** A slug that is not on either list
above will be dropped by code, and nothing will be substituted for it.

Each pick needs a `reason_kind` from the closed set and a `reason` in prose.

**The reason is the product.** It is the part the household reads and the only thing
that makes a proposal worth more than a random draw. Five true sentences that are all
the same sentence are no reasons at all. Say something specific about *this* dish in
*this* week — what it does for the week that the others do not.

## Rules you must not break

1. **Never claim anything about when a dish was last cooked.** There are no
   last-cooked dates in this data. Not one. A recipe with no date is *unranked*, never
   *stale* and never *overdue*. Use `never`, never `stale`. Any recency claim you write
   will be stripped out by code before the household sees it.
2. **Never mention an ingredient, a quantity, or which meals share ingredients.** You
   have not been shown any ingredient list and you cannot open one. Ingredient overlap
   is computed from the recipe files afterwards and printed on the shopping list.
3. **No peanuts.** This is an allergy, not a preference.
4. Every meal must be family-edible — two adults, a three-year-old, a one-year-old.
5. Weeknight meals must be low or medium **active** time. Passive time is not capped:
   a slow cooker or a long braise is a fine weeknight meal here.
6. Vary the protein across the week because variety is good planning. There is no beef
   quota and beef is half the corpus by the household's own choice.
7. At most two meals whose yield is `unknown`. The list cannot scale those, and it says
   so on every one.
8. Prefer surfacing dishes the household has not reached for. Recall is the whole point
   of this tool — the household cooks about 15 of the recipes it knows and likes.
"""


# --- the call --------------------------------------------------------------

def call_claude(prompt: str, schema: dict, model: str = MODEL,
                timeout: int = TIMEOUT_S) -> tuple[dict | None, str, dict]:
    """Run one `claude -p` call and return `(structured_output, error, envelope)`.

    Every argument and every guard here is load-bearing:

    - `--tools ""` removes file access. This is what makes the split real.
    - `stdin` from `/dev/null`, or the CLI waits three seconds on every call.
    - **stdout is captured on its own.** Merging stderr corrupts the JSON — the CLI
      writes warnings there and `json.loads` then dies at character 0.
    - **Trust the exit code and `is_error`, never `subtype`.** On an observed failure the
      envelope said `"subtype":"success"` alongside `"is_error":true` and a 404.
    - A malformed schema produces **empty stdout** and writes to stderr only, so an
      empty read is handled as its own case rather than as a parse error.
    """
    cmd = [
        "claude", "-p", prompt,
        "--output-format", "json",
        "--json-schema", json.dumps(schema),
        "--tools", "",
        "--model", model,
    ]
    try:
        with open("/dev/null") as devnull:
            proc = subprocess.run(cmd, stdin=devnull, stdout=subprocess.PIPE,
                                  stderr=subprocess.PIPE, timeout=timeout)
    except FileNotFoundError:
        return None, "the `claude` CLI is not on PATH", {}
    except subprocess.TimeoutExpired:
        return None, f"the planner did not answer within {timeout}s", {}

    stdout = proc.stdout.decode("utf-8", "replace").strip()
    stderr = proc.stderr.decode("utf-8", "replace").strip()

    if not stdout:
        return None, stderr or f"the planner wrote nothing (exit {proc.returncode})", {}

    try:
        envelope = json.loads(stdout)
    except json.JSONDecodeError as exc:
        return None, f"the planner's answer was not JSON: {exc}", {}

    if proc.returncode != 0 or envelope.get("is_error"):
        message = envelope.get("result") or "; ".join(
            str(e) for e in envelope.get("errors", [])) or stderr
        status = envelope.get("api_error_status")
        return None, f"the planner failed{f' ({status})' if status else ''}: {message}", envelope

    structured = envelope.get("structured_output")
    if not isinstance(structured, dict):
        return None, "the planner returned no structured_output", envelope

    return structured, "", envelope


# --- the code-side checks --------------------------------------------------

def _drop_recency(text: str) -> tuple[str, bool]:
    """Remove any sentence claiming a recency. Four lines, and it stays true forever.

    A test once showed the model behaving well without this guard — it wrote *"no
    last-cooked date is on record for this dish"*, which is correct. One sample, on one
    model, is not proof of safety, and the opposite has happened here before. The check
    costs nothing and it is the only part that survives a model change.
    """
    sentences = re.split(r"(?<=[.!?])\s+", text.strip())
    kept = [s for s in sentences if not _RECENCY.search(s)]
    if len(kept) == len(sentences):
        return text, False
    remaining = " ".join(kept).strip()
    if not remaining:
        remaining = ("code removed a claim about timing here. This repository records no "
                     "dates at all, so the dish is simply unranked.")
    return remaining, True


def check(repo: Repo, picks: list[dict], nights: int) -> PlannerResult:
    """Everything the model said, checked against the data before anybody shops.

    A pick that fails a check is **dropped and never nudged**. Nothing is substituted for
    it here; `fill()` does that afterwards, from the corpus, and says so.
    """
    result = PlannerResult()
    by_slug = {r.slug: r for r in repo.all_rows}
    seen: set[str] = set()
    unknown_yield = 0

    for pick in picks:
        slug = (pick.get("slug") or "").strip()
        title = (pick.get("title") or slug).strip()
        row = by_slug.get(slug)

        if row is None:
            result.dropped.append((title or slug, "no corpus or candidates row has that slug"))
            continue
        if slug in seen:
            result.dropped.append((row.title, "proposed twice"))
            continue
        if slug not in repo.recipes:
            result.dropped.append((row.title, "the Slug column names no file in recipes/"))
            continue

        reason = (pick.get("reason") or "").strip()
        kind = (pick.get("reason_kind") or "plain").strip()

        if _PEANUT.search(reason) or _PEANUT.search(row.notes):
            result.dropped.append((row.title, "peanut — a stated allergy"))
            continue

        if row.yield_.shape == "unknown":
            unknown_yield += 1
            if unknown_yield > 2:
                result.dropped.append((
                    row.title,
                    "more than two meals with an unknown yield — the list cannot scale "
                    "them, and the planner drifts toward them"))
                continue

        if kind == "stale":
            kind = "never"
            result.notes.append(
                f"{row.title}: the reason kind was `stale`, and nothing in this data can "
                f"be stale. Recorded as `never`.")

        reason, stripped = _drop_recency(reason)
        if stripped:
            result.notes.append(
                f"{row.title}: a claim about when this was last cooked was removed. No "
                f"last-cooked dates exist in this repository.")

        seen.add(slug)
        result.meals.append(MealPlan(
            slug=slug, title=row.title, reason_kind=kind, reason=reason,
            yield_raw=row.yield_raw, scale=None, untried=row.untried,
        ))

    return result


def fill(repo: Repo, result: PlannerResult, nights: int) -> PlannerResult:
    """Top the week back up from the corpus after drops, deterministically.

    Code does the filling rather than a second model call, and every filled meal says on
    its face that code chose it. Order favours a protein the week is short of, then a
    known yield, because those are the two things the household can check.
    """
    if len(result.meals) >= nights:
        result.meals = result.meals[:nights]
        return result

    chosen = {m.slug for m in result.meals}
    proteins = [repo.row(m.slug).protein for m in result.meals if repo.row(m.slug)]

    def rank(row: CorpusRow) -> tuple:
        return (proteins.count(row.protein),
                0 if row.yield_.shape != "unknown" else 1,
                row.untried,
                row.title)

    unknowns = sum(1 for m in result.meals
                   if (repo.row(m.slug) or CorpusRow("", "", "", "", "", "", "", "", "")).yield_.shape == "unknown")

    for row in sorted(repo.all_rows, key=rank):
        if len(result.meals) >= nights:
            break
        if row.slug in chosen or row.slug not in repo.recipes:
            continue
        if row.yield_.shape == "unknown":
            if unknowns >= 2:
                continue
            unknowns += 1
        chosen.add(row.slug)
        proteins.append(row.protein)
        result.meals.append(MealPlan(
            slug=row.slug, title=row.title, reason_kind="plain",
            reason=("chosen by code, not by the planner — a pick was dropped by a "
                    "constraint check and this filled the gap."),
            yield_raw=row.yield_raw, scale=None, untried=row.untried,
        ))
    return result


def plan(repo: Repo, nights: int = 5, guests: int = 0,
         last_week: str = "", avoid: list[str] | None = None,
         model: str = MODEL) -> PlannerResult:
    """Plan one week: one model call, then every check, then a deterministic top-up.

    When the call fails the week is still planned — by code, labelled as such. A tool
    that cannot produce a week because a subprocess failed is not a tool the household
    can shop with on a Sunday morning.
    """
    prompt = build_prompt(repo, nights, guests, last_week, avoid)
    structured, error, envelope = call_claude(prompt, SCHEMA, model=model)

    if structured is None:
        result = PlannerResult(source="code", error=error)
        result.notes.append(
            f"The planner did not answer, so code chose this week instead: {error}. "
            f"The meals below are a plain sweep of the corpus, not a considered "
            f"selection, and the reasons say so.")
        return fill(repo, result, nights)

    result = check(repo, structured.get("picks", []), nights)
    result.cost_usd = envelope.get("total_cost_usd")
    result.duration_ms = envelope.get("duration_ms")
    return fill(repo, result, nights)
