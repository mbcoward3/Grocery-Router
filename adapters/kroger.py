"""Kroger, behind the `Store` interface.

Their developer API, over `urllib`. Two scopes and the difference between them is
the whole safety story:

- **`product.compact`** comes from client credentials. Reads the catalogue: names,
  sizes, prices, promotions, aisle. This is all the shopping list needs.
- **`cart.basic:write`** requires a *user* authorization, obtained by that person
  in a browser. Nothing here can mint one.

So `docs/architecture.md` decision 4 - *the tool fills a cart and a human submits
it* - is not enforced by our restraint. There is no credential in this codebase
that can spend money, and adding prices does not create one.

**Cart writing is deliberately not implemented here.** The token it needs has to
come from a real OAuth redirect through a registered callback, which needs a
hosted URL this project does not have yet, and writing an untestable code path
against an API nobody has run is how a plausible-looking thing that has never
worked gets committed. `plan_cart` in `adapters/match.py` produces exactly what
would be sent, and the household reads it. The remaining step is a
`POST /v1/cart/add` with a user token and one line of code.

Standard library only.
"""

from __future__ import annotations

import base64
import json
import os
import time
import urllib.error
import urllib.parse
import urllib.request

from . import Product, Store, StoreUnavailable

BASE = os.environ.get("KROGER_BASE_URL", "https://api.kroger.com/v1").rstrip("/")
TIMEOUT = 15
_token: tuple[str, float] | None = None      # (token, expires_at)


def configured() -> bool:
    return bool(os.environ.get("KROGER_CLIENT_ID")
                and os.environ.get("KROGER_CLIENT_SECRET"))


def _get(url: str, headers: dict) -> dict:
    req = urllib.request.Request(url, headers=headers)
    try:
        with urllib.request.urlopen(req, timeout=TIMEOUT) as resp:
            return json.load(resp)
    except urllib.error.HTTPError as e:
        detail = e.read().decode(errors="replace")[:200]
        raise StoreUnavailable(f"HTTP {e.code}: {detail}") from e
    except (urllib.error.URLError, TimeoutError, OSError) as e:
        raise StoreUnavailable(str(getattr(e, "reason", e))) from e
    except json.JSONDecodeError as e:
        raise StoreUnavailable(f"the API returned something that was not JSON: {e}") from e


def token() -> str:
    """A client-credentials token, cached until shortly before it expires.

    Cached because a shopping list is twenty-odd searches and minting a token for
    each would be rude and slow. Expired early on purpose - a token that dies
    mid-list turns one clean failure into a partial one, and a partial price list
    is harder to read than none.
    """
    global _token
    if _token and _token[1] > time.time():
        return _token[0]
    cid = os.environ.get("KROGER_CLIENT_ID")
    secret = os.environ.get("KROGER_CLIENT_SECRET")
    if not (cid and secret):
        raise StoreUnavailable("KROGER_CLIENT_ID / KROGER_CLIENT_SECRET are not set")
    basic = base64.b64encode(f"{cid}:{secret}".encode()).decode()
    data = urllib.parse.urlencode({"grant_type": "client_credentials",
                                   "scope": "product.compact"}).encode()
    req = urllib.request.Request(f"{BASE}/connect/oauth2/token", data=data, headers={
        "Authorization": f"Basic {basic}",
        "Content-Type": "application/x-www-form-urlencoded"})
    try:
        with urllib.request.urlopen(req, timeout=TIMEOUT) as resp:
            body = json.load(resp)
    except urllib.error.HTTPError as e:
        raise StoreUnavailable(f"could not get a token: HTTP {e.code}") from e
    except (urllib.error.URLError, TimeoutError, OSError) as e:
        raise StoreUnavailable(f"could not reach Kroger: {getattr(e, 'reason', e)}") from e
    access = body.get("access_token")
    if not access:
        raise StoreUnavailable("Kroger returned no access token")
    _token = (access, time.time() + float(body.get("expires_in", 1800)) - 60)
    return access


def _money(value) -> float | None:
    try:
        return float(value)
    except (TypeError, ValueError):
        return None


def parse_product(node: dict) -> Product:
    """One catalogue entry, as this project's `Product`.

    Everything is read defensively and nothing is inferred. A field the store
    does not send comes back empty rather than filled with a plausible default -
    same rule the recipe capture follows, and for the same reason: a made-up size
    is worse than a missing one when the next step is buying it.
    """
    item = (node.get("items") or [{}])[0]
    price = item.get("price") or {}
    return Product(
        sku=str(node.get("productId", "")),
        name=str(node.get("description", "")),
        brand=str(node.get("brand", "")),
        size=str(item.get("size", "")),
        price=_money(price.get("regular")),
        promo=_money(price.get("promo")) or None,
        aisle=str((node.get("aisleLocations") or [{}])[0].get("description", "")),
        stock=str((item.get("inventory") or {}).get("stockLevel", "")),
    )


class Kroger(Store):
    name = "kroger"
    configured = True

    def __init__(self, location: str | None = None):
        self.location = location or os.environ.get("KROGER_LOCATION_ID", "")

    def search(self, term: str, limit: int = 8) -> list[Product]:
        params = {"filter.term": term, "filter.limit": str(limit)}
        if self.location:
            params["filter.locationId"] = self.location
        body = _get(f"{BASE}/products?" + urllib.parse.urlencode(params),
                    {"Authorization": f"Bearer {token()}", "Accept": "application/json"})
        out = [parse_product(n) for n in body.get("data", []) if isinstance(n, dict)]
        # A product with no price is a real answer from Kroger - it means that
        # store has not priced it, usually because no location was given. Kept
        # rather than filtered, so the matcher can still name the SKU and the
        # household can still see what it would be buying.
        return [p for p in out if p.sku]

    def promotions(self, terms: list[str]) -> list[Product]:
        """Whatever of these is actually on sale right now.

        Feeds the Step 0 briefing, which has been emitting invented `DEMO` lines
        since it was written. One real promotion is worth more than four
        convincing fake ones, and this is the function that replaces them.
        """
        out = []
        for term in terms:
            try:
                out += [p for p in self.search(term, limit=5) if p.on_sale]
            except StoreUnavailable:
                continue
        return out


def locations(zipcode: str, limit: int = 5) -> list[dict]:
    """Find a store id for `KROGER_LOCATION_ID`. Prices are per store.

    Not part of the `Store` interface: it runs once, by a person, while setting
    the thing up. `profile.md` has had `Store / pickup: Kroger, [... location]`
    with the location unfilled since it was written, and this is how that blank
    gets closed with a real id rather than a guess.
    """
    body = _get(f"{BASE}/locations?" + urllib.parse.urlencode(
        {"filter.zipCode.near": zipcode, "filter.limit": str(limit)}),
        {"Authorization": f"Bearer {token()}", "Accept": "application/json"})
    return [{"id": n.get("locationId", ""), "name": n.get("name", ""),
             "address": " ".join(str(v) for v in (n.get("address") or {}).values()
                                 if isinstance(v, str))}
            for n in body.get("data", []) if isinstance(n, dict)]
