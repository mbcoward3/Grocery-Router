#!/usr/bin/env python3
"""Read the decision log back.

    ./review.py                # everything
    ./review.py --reasons      # which reasons get accepted and which get dropped
    ./review.py --breadth      # is the repertoire actually widening
    ./review.py --json

`decisions.jsonl` was built on one argument: **a decision that was not recorded
cannot be recovered.** It has been recording every proposal, drop, dial change and
outcome since day one, and until now nothing read it. `docs/brief-next.md` §7 is
blunt about the cost - the session's metrics strip showed five numbers computed
off the *corpus*, which is a description of what the household owns rather than
of what the tool did.

Three questions, and they are the ones the brief asks:

**Which reasons get accepted, and which get dropped.** The whole product claim is
that the reason is the product - a forgotten recipe lands because you were told
*you haven't made this since March*. That claim is testable now: every proposal
records the *kind* of reason it surfaced under, and every drop records the kind it
was shown. If `stale` gets accepted and `plain` gets dropped, that is the claim
holding. If they are the same, it is not.

**Whether breadth is actually increasing.** The opponent is *buy it again*, so the
measure is distinct recipes reaching the table over time, not meals served.

**Whether the planner change was worth it.** The model planner and the ranker are
both in the log by name, so their accept rates can be compared against real weeks
instead of argued about. That is the thing this file was built to make possible
and the reason it survived the reversal back to markdown.

**Nothing here writes.** It is a reader over an append-only file, so it is safe to
run at any time and cannot corrupt the evidence it is reading.

Standard library only.
"""

from __future__ import annotations

import argparse
import json
import sys
from collections import Counter, defaultdict

import household
import pantry

# What each reason kind is claiming, for a report a person reads. The kinds
# themselves come from `pantry._reasons`.
KIND_MEANING = {
    "stale": "hasn't been cooked in a while",
    "never": "no record of being cooked yet",
    "protein": "the only one of its protein this week",
    "cuisine": "the only one of its cuisine this week",
    "yield": "one cook, two nights",
    "passive": "long unattended, short hands-on",
    "low": "low active — a bad day can't break it",
    "acquire": "new here, widening the corpus",
    "plain": "nothing ruled it out",
    "model": "written by the model planner",
}


def proposals(hh) -> list[dict]:
    return pantry.decisions(hh, {"proposed"})


def offers(hh) -> list[dict]:
    """Every meal ever put in front of the household, newest last.

    One row per *offer*, not per recipe: the same recipe proposed in three weeks
    is three offers, and that is the right denominator for an accept rate.
    """
    out = []
    for rec in proposals(hh):
        for added in rec.get("added", []):
            out.append({"at": rec.get("at", ""), "recipe": added.get("recipe", ""),
                        "kind": added.get("kind", ""), "reason": added.get("reason", ""),
                        "candidate": bool(added.get("candidate")),
                        "protein": added.get("protein", ""),
                        "cuisine": added.get("cuisine", ""),
                        "planner": rec.get("planner", "ranker")})
    return out


def kind_of(hh, recipe: str) -> str:
    """The reason kind this recipe was most recently proposed under.

    Used by the drop route, so a turn-down records what it was turned down
    *against*. Joined out of the log rather than carried in the week file: the
    week's meal lines have a fixed format, and the log is append-only and already
    holds the answer.
    """
    for offer in reversed(offers(hh)):
        if offer["recipe"] == recipe:
            return offer["kind"]
    return ""


# --------------------------------------------------------------------------- #
# Which reasons work
# --------------------------------------------------------------------------- #

def reasons(hh) -> list[dict]:
    """Accept rate per reason kind, worst first.

    `dropped` is an explicit turn-down in the session. `kept` is a cook the
    household said they kept. **They are different denominators and both are
    reported** - a meal that was neither dropped nor cooked is not evidence
    either way, and rolling it into one number would invent a verdict for a week
    nobody finished.
    """
    offered = Counter()
    by_recipe: dict[str, list[str]] = defaultdict(list)
    for offer in offers(hh):
        if not offer["kind"]:
            continue
        offered[offer["kind"]] += 1
        by_recipe[offer["recipe"]].append(offer["kind"])

    dropped = Counter()
    for rec in pantry.decisions(hh, {"drop"}):
        kind = rec.get("reason_kind") or (by_recipe.get(rec.get("recipe", "")) or [""])[-1]
        if kind:
            dropped[kind] += 1

    kept = Counter()
    for rec in pantry.decisions(hh, {"feedback_applied"}):
        if not str(rec.get("outcome", "")).startswith("kept"):
            continue
        kinds = by_recipe.get(rec.get("recipe", ""))
        if kinds:
            kept[kinds[-1]] += 1

    out = []
    for kind, count in offered.most_common():
        out.append({
            "kind": kind,
            "means": KIND_MEANING.get(kind, ""),
            "offered": count,
            "dropped": dropped[kind],
            "kept": kept[kind],
            "accept_rate": round(100 * (count - dropped[kind]) / count),
        })
    out.sort(key=lambda r: (r["accept_rate"], -r["offered"]))
    return out


# --------------------------------------------------------------------------- #
# Whether breadth is increasing
# --------------------------------------------------------------------------- #

def breadth(hh) -> list[dict]:
    """Distinct recipes reaching the table, week by week.

    **The measure the product claim actually needs.** The opponent is *buy it
    again*, so meals served says nothing - five dinners a week is five dinners a
    week whether they are the same five every time or never the same twice. What
    matters is `new`: recipes surfaced for the first time.
    """
    weeks: dict[str, dict] = {}
    seen: set[str] = set()
    for rec in proposals(hh):
        day = rec.get("week") or str(rec.get("at", ""))[:10]
        week = weeks.setdefault(day, {"week": day, "offered": 0, "new": 0,
                                      "proteins": set(), "cuisines": set(),
                                      "candidates": 0})
        for added in rec.get("added", []):
            slug = added.get("recipe", "")
            week["offered"] += 1
            if slug and slug not in seen:
                seen.add(slug)
                week["new"] += 1
            if added.get("protein"):
                week["proteins"].add(added["protein"])
            if added.get("cuisine"):
                week["cuisines"].add(added["cuisine"])
            if added.get("candidate"):
                week["candidates"] += 1
    out = []
    running = 0
    for day in sorted(weeks):
        row = weeks[day]
        running += row["new"]
        out.append({"week": day, "offered": row["offered"], "new": row["new"],
                    "distinct_so_far": running,
                    "proteins": len(row["proteins"]),
                    "cuisines": len(row["cuisines"]),
                    "candidates": row["candidates"]})
    return out


# --------------------------------------------------------------------------- #
# Whether the planner change was worth it
# --------------------------------------------------------------------------- #

def planners(hh) -> dict:
    """The model against the ranker, on real weeks.

    This is the comparison `decisions.jsonl` exists for. It is reported even when
    one side has no weeks yet, because *no model has ever planned here* is itself
    the answer to how well the model planner is doing.
    """
    out: dict[str, dict] = {}
    dropped = Counter(r.get("recipe", "") for r in pantry.decisions(hh, {"drop"}))
    for offer in offers(hh):
        row = out.setdefault(offer["planner"],
                             {"planner": offer["planner"], "offered": 0,
                              "dropped": 0, "weeks": set()})
        row["offered"] += 1
        row["weeks"].add(offer["at"][:10])
        if dropped.get(offer["recipe"]):
            row["dropped"] += 1
    fallbacks = [r for r in proposals(hh) if r.get("fallback")]
    return {
        "by_planner": [
            {**{k: v for k, v in row.items() if k != "weeks"},
             "weeks": len(row["weeks"]),
             "accept_rate": round(100 * (row["offered"] - row["dropped"])
                                  / row["offered"]) if row["offered"] else None}
            for row in out.values()
        ],
        "asked_for_a_model": sum(1 for r in proposals(hh) if r.get("asked") == "model"),
        "fell_back": len(fallbacks),
        "why": Counter(r["fallback"] for r in fallbacks).most_common(3),
    }


def acquisition(hh) -> dict:
    """Where new recipes came from. `found_by` tells searching from pasting."""
    got = pantry.decisions(hh, {"acquired"})
    return {
        "total": len(got),
        "by_route": dict(Counter(r.get("found_by", "?") for r in got)),
        "sources": Counter(
            str(r.get("source", "")).split("/")[2] for r in got
            if str(r.get("source", "")).count("/") > 2).most_common(5),
        "promoted": len(pantry.decisions(hh, {"promote"})),
    }


def summary(hh) -> dict:
    weeks = breadth(hh)
    return {
        "decisions": len(pantry.decisions(hh)),
        "weeks_logged": len(weeks),
        "offers": len(offers(hh)),
        "distinct_recipes": weeks[-1]["distinct_so_far"] if weeks else 0,
        "reasons": reasons(hh),
        "breadth": weeks,
        "planners": planners(hh),
        "acquisition": acquisition(hh),
    }


# --------------------------------------------------------------------------- #

def _table(rows: list[dict], columns: list[tuple[str, str]]) -> str:
    if not rows:
        return "  (nothing logged yet)"
    widths = {key: max(len(label), *(len(str(r.get(key, ""))) for r in rows))
              for key, label in columns}
    out = ["  " + "  ".join(label.ljust(widths[key]) for key, label in columns),
           "  " + "  ".join("-" * widths[key] for key, _ in columns)]
    for row in rows:
        out.append("  " + "  ".join(str(row.get(key, "")).ljust(widths[key])
                                    for key, _ in columns))
    return "\n".join(out)


def main():
    p = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    p.add_argument("--reasons", action="store_true")
    p.add_argument("--breadth", action="store_true")
    p.add_argument("--planners", action="store_true")
    p.add_argument("--json", action="store_true")
    args = p.parse_args()
    hh = household.here()

    data = summary(hh)
    if args.json:
        print(json.dumps(data, indent=2, default=str))
        return 0

    if not data["decisions"]:
        print("Nothing in decisions.jsonl yet. It fills as the session gets used —\n"
              "and it cannot be backfilled, which is why it was built first.",
              file=sys.stderr)
        return 1

    everything = not (args.reasons or args.breadth or args.planners)

    if everything:
        print(f"\n{data['decisions']} decisions · {data['weeks_logged']} week(s) "
              f"· {data['offers']} meals offered · "
              f"{data['distinct_recipes']} distinct recipes reached the table")

    if everything or args.reasons:
        print("\nWhich reasons land\n"
              "  The claim is that the reason is the product. If every kind scores\n"
              "  the same, the claim is not doing any work.\n")
        print(_table(data["reasons"], [("kind", "kind"), ("means", "what it says"),
                                       ("offered", "offered"), ("dropped", "dropped"),
                                       ("kept", "kept"), ("accept_rate", "accept %")]))

    if everything or args.breadth:
        print("\nBreadth\n"
              "  The opponent is buying the same things again, so `new` is the\n"
              "  column that matters. `offered` can stay flat forever.\n")
        print(_table(data["breadth"], [("week", "week"), ("offered", "offered"),
                                       ("new", "new"), ("distinct_so_far", "distinct"),
                                       ("proteins", "proteins"), ("cuisines", "cuisines"),
                                       ("candidates", "candidates")]))

    if everything or args.planners:
        pl = data["planners"]
        print("\nPlanner\n"
              "  Two implementations behind one call, compared on real weeks\n"
              "  rather than on opinion.\n")
        print(_table(pl["by_planner"], [("planner", "planner"), ("weeks", "weeks"),
                                        ("offered", "offered"), ("dropped", "dropped"),
                                        ("accept_rate", "accept %")]))
        if pl["asked_for_a_model"]:
            print(f"\n  asked for a model {pl['asked_for_a_model']} time(s), "
                  f"fell back {pl['fell_back']}")
            for why, n in pl["why"]:
                print(f"    {n}x  {why}")

    if everything:
        acq = data["acquisition"]
        print(f"\nAcquisition\n  {acq['total']} recipe(s) added — "
              f"{acq['by_route'] or 'none yet'}; {acq['promoted']} promoted to the corpus")
        if acq["total"] and not acq["promoted"]:
            print("  Nothing acquired has been cooked and kept yet. The tool can find\n"
                  "  recipes faster than the household can prove them, which is the\n"
                  "  expected shape and worth watching.")
    print()
    return 0


if __name__ == "__main__":
    sys.exit(main())
