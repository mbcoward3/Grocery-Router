"""Apply explicit SQL migrations to the configured production database."""

from __future__ import annotations

import argparse
from pathlib import Path

from . import storage as ST


def main(argv: list[str] | None = None) -> None:
    parser = argparse.ArgumentParser(description="Apply Grocery Router database migrations")
    parser.add_argument("--root", type=Path, default=Path("."))
    args = parser.parse_args(argv)
    root = args.root.resolve()
    store = ST.from_environment(root)
    if not isinstance(store, ST.DatabaseStore):
        raise ST.ConfigurationError("migrations require database storage")
    changed = store.migrate(root / "migrations")
    if changed:
        print("Applied migrations: " + ", ".join(changed))
    else:
        print("Database schema is current.")


if __name__ == "__main__":
    main()
