#!/usr/bin/env python3
"""Render Recipes.pdf to page images and dump its text layer.

The one step in onboarding that is not standard library. `onboard.py` takes
screenshots as image files or as transcripts; this is what turns the household's
saved recipe document into those inputs, so the run is reproducible from the
committed PDF alone.

    pip install pypdfium2
    ./sources/extract_pages.py --out /tmp/recipe-pages

Page images are deliberately not committed - the PDF already is, and re-rendering
it is one command. Page 1 and pages 9-11 carry a text layer (the eight saved
links and the six typed recipes); every other page is a screenshot with no text
behind it.
"""

import argparse
from pathlib import Path

import pypdfium2 as pdfium

ROOT = Path(__file__).resolve().parent


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--pdf", type=Path, default=ROOT / "Recipes.pdf")
    p.add_argument("--out", type=Path, required=True, help="directory for page PNGs")
    p.add_argument("--scale", type=float, default=2.0, help="render scale (default 2.0)")
    p.add_argument("--text", action="store_true", help="also print the text layer")
    args = p.parse_args()

    args.out.mkdir(parents=True, exist_ok=True)
    doc = pdfium.PdfDocument(str(args.pdf))
    for i, page in enumerate(doc, start=1):
        page.render(scale=args.scale).to_pil().save(args.out / f"p{i:02d}.png")
        if args.text:
            text = page.get_textpage().get_text_range().strip()
            print(f"===== page {i} =====")
            if text:
                print(text)
    print(f"{len(doc)} page(s) -> {args.out}")


if __name__ == "__main__":
    main()
