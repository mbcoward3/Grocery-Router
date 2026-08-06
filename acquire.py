#!/usr/bin/env python3
"""Run acquisition from the command line.

The package is `acquire/`; this is the entry point, so `./acquire.py` keeps
working now that searching grew from one endpoint into a ladder of strategies.
"""

import sys

from acquire import main

if __name__ == "__main__":
    sys.exit(main())
