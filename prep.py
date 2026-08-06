#!/usr/bin/env python3
"""Step 0 — the prep job. Runs unattended before a session, leaves a briefing.

    ./prep.py            # write .cache/briefing.md
    ./prep.py --print

The session must never wait on this. Everything here **degrades rather than
blocks**: a stage that cannot run writes what it knows and says what it does not,
and a session with no briefing at all is a normal session with one less card.

Two of the three stages are real. The sale hunt is not — there is no Kroger
adapter yet, so it emits clearly-labelled DEMO lines rather than nothing, because
a dead card teaches you less about the design than a fake one.
"""

import argparse
import datetime as dt
import random

import household
import pantry

DEMO_SALES = [
    ("chuck roast", "$3.99/lb", ["crock-pot-italian-beef", "beef-pot-roast", "beef-dip-sammies"]),
    ("bell peppers", "3 for $2", ["sheet-pan-chicken-fajitas", "sausage-and-peppers"]),
    ("italian sausage", "$1 off", ["sausage-and-peppers", "zuppa-toscana"]),
    ("salmon fillets", "$8.99/lb", ["easy-salmon-dinner", "parchment-garlic-butter-salmon"]),
]


def staleness(hh, today):
    """Real. What has fallen out of rotation, read off the corpus."""
    out, never = [], 0
    for row in pantry.load_corpus(hh):
        gap = pantry.days_since(row, today)
        if gap is None:
            never += 1
        elif gap > 60:
            out.append((gap, row["recipe"]))
    lines = [f"{name} — not cooked in {gap // 30} months" for gap, name in
             sorted(out, reverse=True)[:4]]
    if never:
        lines.append(f"{never} recipes have no last-cooked date yet — unranked, not stale")
    return lines


def sales(hh, today):
    """Real when a store is configured, and clearly fake when one is not.

    This function has emitted invented `DEMO` lines since it was written, on the
    grounds that a dead card teaches less about the design than a labelled fake
    one. That was the right call while there was no adapter. There is one now, so
    a household with Kroger credentials gets prices that are true, and one
    without gets exactly what it got before - **still labelled, because a
    plausible number nobody can trace is the failure this project is built
    around.**
    """
    corpus = {r["slug"] for r in pantry.load_corpus(hh)}
    try:
        import adapters
        store = adapters.store()
        if store.configured:
            terms = sorted({(r.get("protein") or "").strip()
                            for r in pantry.load_corpus(hh)} - {""})
            on_sale = store.promotions(terms)
            lines = []
            for prod in on_sale[:4]:
                hits = [r["recipe"] for r in pantry.load_corpus(hh)
                        if (r.get("protein") or "").lower() in prod.name.lower()]
                saving = f"${prod.promo:.2f} (was ${prod.price:.2f})"
                lines.append(f"{prod.name} {saving}"
                             + (f" → {', '.join(hits[:2])}" if hits else ""))
            if lines:
                return lines
            return ["nothing on your list is on sale at that store this week"]
    except Exception as exc:                       # degrades, never blocks
        return [f"no prices — the store could not be reached ({exc})"]

    rng = random.Random(today.isoformat())
    lines = []
    for item, price, slugs in rng.sample(DEMO_SALES, 3):
        hits = [s.replace("-", " ") for s in slugs if s in corpus]
        if hits:
            lines.append(f"DEMO — {item} {price} → {', '.join(hits[:2])}")
    return lines


def open_loops(hh, today):
    """Real. Questions a week can close in one tap, pulled from the recipe files."""
    lines = []
    unknown = [r["recipe"] for r in pantry.load_corpus(hh)
               if r.get("yield", "").startswith("unknown")]
    if unknown:
        lines.append(f"{len(unknown)} recipes still have no yield — the question answers "
                     f"itself the first time each one is cooked")
    portions = [r["recipe"] for r in pantry.load_corpus(hh)
                if r.get("yield", "") and not r["yield"][0].isdigit() is False
                and ("enchilada" in r["yield"] or "slider" in r["yield"])]
    for name in portions:
        lines.append(f"{name} — how many per adult? one number, reusable forever")
    return lines


def build(hh, today=None):
    today = today or dt.date.today()
    blocks = [("On sale", sales(hh, today)),
              ("Fallen out of rotation", staleness(hh, today)),
              ("Open questions", open_loops(hh, today))]
    out = [f"# Briefing", "", f"generated: {dt.datetime.now().isoformat(timespec='minutes')}",
           "", "*Cached by `prep.py`. The session reads this and never waits for it.*", ""]
    for title, lines in blocks:
        out.append(f"## {title}")
        out.append("")
        out.extend(f"- {l}" for l in (lines or ["- nothing"]))
        out.append("")
    return "\n".join(out)


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("--print", dest="show", action="store_true")
    args = ap.parse_args()
    hh = household.here()
    text = build(hh)
    if args.show:
        print(text)
        return
    hh.cache.mkdir(parents=True, exist_ok=True)
    (hh.cache / "briefing.md").write_text(text, encoding="utf-8")
    print(f"wrote {hh.cache / 'briefing.md'}")


if __name__ == "__main__":
    main()
