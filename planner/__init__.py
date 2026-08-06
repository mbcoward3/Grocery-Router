"""Two implementations behind one call.

`docs/architecture.md` gives this package one job: *"a deterministic ranker that
always works, and a model planner. The ranker is not a fallback to apologise for
- it has to be genuinely good, because it is what runs in a demo with no key."*

So this module is only the choice between them, and it is deliberately small.
The ranker lives in `pantry.rank()`, next to the corpus loaders and the `Meal` it
builds, because moving it here would have bought a tidier directory listing at
the price of a circular import and a diff across a hundred and twenty-nine
passing tests. The call everything goes through is still `pantry.propose()`.

**Selection, in order:** an explicit argument beats `PANTRY_PLANNER` in the
environment, which beats the default. The default is *use the model if a key is
present*, which is what makes the hosted demo work with no configuration at all
and makes a local run with a key work with no flag at all.

`PANTRY_PLANNER=ranker` is the switch that matters most in practice: it pins a
run deterministic even where a key exists, which is what CI and the test suite
want, and what anyone comparing the two implementations wants.
"""

from __future__ import annotations

import os

RANKER = "ranker"
MODEL = "model"
AUTO = "auto"


def configured() -> str:
    """What the environment asks for. Anything unrecognised means `auto`.

    A typo in an environment variable must not silently disable the model, and
    must not silently enable it either - `auto` is the honest reading of "I do
    not know what you meant", because it then decides on the key, which is a
    fact rather than a guess.
    """
    want = (os.environ.get("PANTRY_PLANNER") or "").strip().lower()
    return want if want in (RANKER, MODEL, AUTO) else AUTO


def has_key() -> bool:
    return bool(os.environ.get("ANTHROPIC_API_KEY"))


def which(explicit: str | None = None) -> str:
    """Which implementation plans this week: `"ranker"` or `"model"`.

    Asking for the model without a key still returns `"model"`. That is not an
    oversight - `propose()` needs the attempt to fail loudly and land in the
    decision log, because a household that set `PANTRY_PLANNER=model` and quietly
    got the ranker every week has been told nothing at all.
    """
    want = (explicit or "").strip().lower() or configured()
    if want == RANKER:
        return RANKER
    if want == MODEL:
        return MODEL
    return MODEL if has_key() else RANKER
