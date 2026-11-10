#!/usr/bin/env python3
"""Update only the shared development HelmRelease's image digest."""
import pathlib
import re
import sys


def update(path: pathlib.Path, digest: str) -> None:
    if not re.fullmatch(r"sha256:[0-9a-f]{64}", digest):
        raise ValueError("invalid image digest")
    text = path.read_text()
    updated, count = re.subn(
        r'(?m)^(    image:\n      digest: )"sha256:[0-9a-f]{64}"$',
        rf'\g<1>"{digest}"',
        text,
    )
    if count != 1:
        raise ValueError(f"expected one development image digest, replaced {count}")
    path.write_text(updated)


if __name__ == "__main__":
    update(pathlib.Path(sys.argv[1]), sys.argv[2])
