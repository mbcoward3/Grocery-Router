# Grocery Router web

The production frontend is a strict TypeScript React application built with Vite. Atlas is the
selected visual baseline under `design/ui-explorations/concept-a.html`.

## Stack

- TanStack Router for typed routes
- TanStack Query for API-backed server state
- OpenAPI-generated transport types in `src/api/schema.d.ts`
- Plain responsive CSS for the deliberately small Atlas visual vocabulary
- Vitest and Testing Library for component and client tests

SQLite-backed values belong in TanStack Query rather than a second global store. Local UI state,
such as an open picker, stays in the owning component. Add another TanStack package only when a
current product requirement needs it; charting and other deferred features do not justify idle
dependencies.

## Development

From the repository root:

```sh
task web-install
task serve       # terminal one; expects an ingested database
task web-dev     # terminal two
```

Vite serves `http://localhost:5173` and proxies `/api` to `http://127.0.0.1:8080`.

Regenerate API declarations after changing `api/openapi.yaml`:

```sh
task web-generate
```

Do not edit `src/api/schema.d.ts` directly.
