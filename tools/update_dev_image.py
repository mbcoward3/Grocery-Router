#!/usr/bin/env python3
"""Update only the shared development Deployment's immutable application image."""
import pathlib
import re
import sys


def update(path: pathlib.Path, digest: str) -> None:
    if not re.fullmatch(r"sha256:[0-9a-f]{64}", digest):
        raise ValueError("invalid image digest")
    text = path.read_text()
    updated, count = re.subn(
        r"(?m)^(\s*image: )ghcr\.io/mbcoward3/grocery-router(?::[^\s]+|@sha256:[0-9a-f]{64})(\s*)$",
        rf"\g<1>ghcr.io/mbcoward3/grocery-router@{digest}\g<2>",
        text,
    )
    if count != 1:
        raise ValueError(f"expected one development image, replaced {count}")
    path.write_text(updated)


if __name__ == "__main__":
    update(pathlib.Path(sys.argv[1]), sys.argv[2])
