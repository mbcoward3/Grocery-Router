"""Which household a call is about.

**This exists because of a bug, not because of a database.**

`pantry.py` used to hold the answer in module-level globals — `ROOT`, `CORPUS`,
`WEEKS` and five others — and `app.py` reached in and reassigned them. That is
fine for exactly one household and wrong for two, because `app.py` is a
`ThreadingHTTPServer` and has always been concurrent: two requests in flight,
one repoints `pantry.CORPUS` while the other is between a read and a write, and
household A's corpus is written into household B's file. Not a crash, not a
traceback — a stranger's stated allergy in somebody else's kitchen, silently.
`docs/multi-tenancy.md` has the long version. It is the first item there and it
had to land before anything has two rows in it.

So the household is a **required first argument** to everything that touches
household data. Not a context variable, not a thread-local, not a global with a
setter — those all fix the concurrency and keep the thing that made it possible,
which is that a function reading a household does not say so. And they share the
worse property: forgetting to set one does not fail, it quietly uses whatever
was there. This repo's own list of traps says the failure is always *a plausible
value where there should have been a gap*. A missing argument is a `TypeError`
at the call, which is the gap.

The cost, honestly: every signature in `pantry.py` and half of `shop.py` grew a
parameter, and every test harness had to build one. That was a wide diff for no
new behaviour. It buys the property that there is now no way to read household
data without naming the household, and that property is worth more the moment a
second one exists.

**What is *not* here:** the host-keyed caches in `acquire/adapters.py` and the
OAuth token in `adapters/kroger.py`. Those are keyed by hostname and by
platform credential, not by tenant, and they are shared on purpose — 500
households must not each re-probe `thecountrycook.net`, and `_last_hit` is the
courtesy delay between requests, so making it per-tenant would turn a polite
client into a scraper that looks polite. Tenant state gets threaded; host state
stays shared. Backwards in either direction is a different bug.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from pathlib import Path

REPO = Path(__file__).resolve().parent


@dataclass
class Household:
    """Where one household's files live, and who they belong to.

    `id` and `tier` are unused by the file backend and are here anyway: they are
    what the Postgres backend keys on and what billing reads, and adding them
    later would mean touching every construction site a second time. `tier`
    already has a meaning the code implements — `docs/multi-tenancy.md` maps the
    free tier onto the deterministic ranker and the paid tier onto the model
    planner, which is a selection `planner/` has made since before anyone
    thought about money.
    """

    root: Path
    id: str = "household"
    tier: str = "free"

    # The recipe-title index, cached per household. It used to be
    # `pantry._FILE_INDEX`, a module global that four separate places had to
    # remember to reset to `None` after writing a recipe file — and which under
    # two households would have served one of them the other's filenames.
    # `pantry.file_index()` fills it; nothing else should touch it.
    index: dict[str, str] | None = field(default=None, repr=False, compare=False)

    def __post_init__(self) -> None:
        self.root = Path(self.root)

    # --- the files, all derived, none stored ------------------------------- #
    @property
    def corpus(self) -> Path:
        return self.root / "corpus.md"

    @property
    def candidates(self) -> Path:
        return self.root / "candidates.md"

    @property
    def sides(self) -> Path:
        return self.root / "sides.md"

    @property
    def profile(self) -> Path:
        return self.root / "profile.md"

    @property
    def items(self) -> Path:
        return self.root / "items.md"

    @property
    def recipes(self) -> Path:
        return self.root / "recipes"

    @property
    def weeks(self) -> Path:
        return self.root / "weeks"

    @property
    def cache(self) -> Path:
        return self.root / ".cache"

    @property
    def decisions(self) -> Path:
        return self.root / "decisions.jsonl"

    def forget(self) -> None:
        """Drop the cached recipe index, after writing a recipe file.

        One household's cache, so this can no longer clear somebody else's.
        """
        self.index = None


def here(root: Path | str | None = None) -> Household:
    """The household this checkout belongs to.

    Every CLI in this project — `plan.py`, `shop.py`, `acquire.py`, `review.py`,
    `onboard.py`, `prep.py` — is a single-household tool run from inside the
    household's own repo, and stays one. This is where they get theirs, and it
    is the *only* place a default household is constructed: a library function
    that quietly defaults to `here()` would put the implicit global straight
    back, one call site at a time.
    """
    return Household(root=Path(root) if root else REPO, id="household")
