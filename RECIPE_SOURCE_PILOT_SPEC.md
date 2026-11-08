# Grocery Router — Recipe Source Extraction Pilot

Status: **approved standalone pilot; implementation authorized**

Original product authority: [`V1_SPEC.md`](V1_SPEC.md)

## 1. Outcome

Evaluate a small, repeatable Go-only path for acquiring recipe evidence from explicitly selected
public web pages. A checked-in manifest names every page. The pilot safely fetches each page,
finds source-provided Schema.org `Recipe` JSON-LD, represents it without interpreting recipe
content, and writes human-inspectable JSON artifacts.

This is evidence acquisition, not recipe onboarding. Pilot output is not approved corpus data and
is never read by the application or ingested into PostgreSQL.

## 2. Scope

Included:

- a versioned YAML manifest of stable source IDs and public HTTPS recipe URLs;
- sequential, bounded HTTP fetching with SSRF and redirect protections;
- extraction of `Recipe` nodes from JSON-LD script blocks, including arrays and `@graph` nodes;
- a source-faithful `SourceRecipe` whose selected Schema.org properties remain raw JSON values and
  whose complete recipe object is retained;
- request, redirect, response, canonical-link, JSON-LD location, timestamp, size, and SHA-256
  provenance;
- one inspectable JSON artifact per manifest source plus a run index; and
- tests using local fixtures and injected fetch results rather than live websites.

Explicitly excluded:

- Jev or any other agent/runtime extraction service;
- recipe categorization, canonical grocery mapping, ingredient parsing, quantity interpretation,
  normalization, repair, completion, scoring, or inferred values;
- PostgreSQL, migrations, sqlc, API, authentication, or UI changes;
- automatic corpus updates or approval;
- crawling, sitemap discovery, arbitrary user-submitted URLs, browser automation, paywall or bot
  challenge bypass, and retries intended to evade source controls; and
- Python or `recipe-scrapers` as an initial dependency. A later evaluation may run the Python
  library independently against the same manifest and compare its output with these artifacts.

## 3. Manifest contract

`pilot/recipe-sources.yaml` uses this strict shape:

```yaml
version: 1
sources:
  - id: stable-kebab-case-id
    url: https://public.example/recipe
```

IDs are unique and safe as filenames. URLs must be absolute HTTPS URLs, have no credentials, and
use the default HTTPS port. Unknown fields, duplicate IDs or URLs, fragments, and empty manifests
are rejected before any fetch begins. Manifest order controls fetch and index order.

The checked-in manifest is an explicit operator allowlist. The command does not discover links to
fetch.

## 4. Safe-fetch contract

For the initial pilot:

- requests are sequential and use `GET` with a clear pilot user agent;
- each request has a 20-second deadline;
- redirects are limited to five and every redirect is revalidated under the same URL and network
  rules;
- proxy environment variables are not used;
- DNS is resolved immediately before dialing; loopback, private, link-local, multicast,
  unspecified, documentation, benchmarking, carrier-grade NAT, and other reserved destinations are
  rejected, including when only one answer is non-public;
- only default-port HTTPS is fetched;
- only successful HTML/XHTML responses are accepted;
- decoded response bodies are limited to 5 MiB; and
- there is no retry, concurrency, cookie jar, authentication, or execution of page script.

A source failure is recorded in that source's artifact. Remaining manifest entries are attempted,
and the command exits unsuccessfully after writing a complete run report if any entry failed.

## 5. Extraction and fidelity

Only `<script type="application/ld+json">` content is considered. Matching is case-insensitive and
allows media-type parameters. A JSON object is a recipe when its `@type` is `Recipe`, the full
`schema.org/Recipe` IRI, or contains one of those values in an array, compared case-insensitively. JSON arrays and nested objects (including
`@graph`) are walked in source order where JSON defines order.

No fallback extraction from visible HTML, Open Graph, microdata, or source-specific selectors is
allowed. Malformed non-empty JSON-LD blocks are reported as warnings. A page fails extraction when
it yields no Recipe node.

`SourceRecipe` copies common Schema.org properties as `json.RawMessage`, preserving whether the
source supplied a string, array, object, number, or null. The complete Recipe object is also
retained as raw JSON, so unknown source properties are not discarded. Durations, yields,
ingredients, instructions, images, authors, ratings, and nutrition are not parsed or rewritten.

## 6. Provenance and output

Each invocation creates a new UTC-named directory beneath the configured output root. It contains:

- `<source-id>.json` for every source, including failures; and
- `index.json` with manifest path and digest, run time, implementation identifier, ordered artifact
  list, and success/failure counts.

Successful source artifacts retain:

- the exact manifest ID and URL;
- requested and final response URLs;
- every redirect status and URL pair;
- fetch timestamp, status, media type, selected validators, decoded byte count, and body SHA-256;
- source-provided canonical-link text and a mechanically resolved canonical URL when valid;
- every JSON-LD block's index, size, digest, and parse error if any; and
- each Recipe node's script index, JSON location, complete containing JSON-LD value, and
  source-faithful `SourceRecipe`.

Runtime output defaults under `.local-run/` and is intentionally not committed. Output is evidence,
not authority.

## 7. Command

```sh
go run ./cmd/grocery-router recipe-source-pilot \
  --manifest pilot/recipe-sources.yaml \
  --output .local-run/recipe-source-pilot
```

`task recipe-source-pilot` is the standard wrapper.

## 8. Acceptance

- strict manifest validation occurs before network access;
- private/reserved destinations and unsafe redirects are rejected;
- body, redirect, content-type, status, and time bounds are enforced;
- fixtures cover a direct Recipe object, arrays/`@graph`, multiple recipes, malformed JSON-LD,
  non-Recipe JSON-LD, and source values with differing JSON shapes;
- tests prove raw unknown fields and URL/provenance metadata survive output serialization;
- a failed source still receives an artifact and does not prevent later entries from running;
- the command returns non-zero when any source fails; and
- `go test ./...` and repository lint pass without Python recipe extraction dependencies.

## 9. Later comparison

A later, separately invoked benchmark may compare Python `recipe-scrapers` with this pilot using the
same manifest and captured evidence. That comparison must report field coverage and source fidelity;
it must not silently promote either tool's interpreted ingredient or category data into the corpus.
