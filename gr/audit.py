"""The audit `items.md` describes: parse every recipe and print every line with no row.

`items.md` says the table "grows on parse failure … a miss is visible rather than silent.
Adding a row is the fix, and it is meant to be routine." This is the command that makes a
miss visible.

    python3 -m gr.audit
"""

from __future__ import annotations

import sys
from pathlib import Path

from . import repo as R


def main(root: str = ".") -> int:
    repo = R.load(Path(root))
    total = resolved = 0
    misses: list[tuple[str, str, str]] = []

    for slug in sorted(repo.recipes):
        for line in repo.recipes[slug].lines:
            total += 1
            if line.resolved:
                resolved += 1
            else:
                misses.append((slug, line.raw.lstrip("- ").strip(),
                               line.refusal or "no items.md row"))

    for slug, raw, reason in misses:
        print(f"  MISS  [{slug}] {raw}\n        -> {reason}")

    rate = resolved / total if total else 0.0
    print(f"\nitems.md resolution: {resolved} of {total} ingredient lines ({rate:.1%})")

    missing = repo.missing_recipe_files()
    if missing:
        print("\nRows whose Slug names no file in recipes/:")
        for row in missing:
            print(f"  {row.title} -> {row.slug}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1] if len(sys.argv) > 1 else "."))
