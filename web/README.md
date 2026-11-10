# Grocery Router web

The production frontend is a strict TypeScript React application built with Vite. Atlas is the
selected visual baseline under `design/ui-explorations/concept-a.html`.

## Stack

- TanStack Router for typed routes
- TanStack Query for API-backed server state
- OpenAPI-generated transport types in `src/api/schema.d.ts`
- Plain responsive CSS for the deliberately small Atlas visual vocabulary
- Vitest and Testing Library for component and client tests

PostgreSQL-backed values belong in TanStack Query rather than a second global store. Local UI state,
such as an open picker, stays in the owning component. Add another TanStack package only when a
current product requirement needs it; charting and other deferred features do not justify idle
dependencies.

## Development

From the repository root:

```sh
task web-install
task serve       # terminal one; expects an ingested database and the auth environment below
task web-dev     # terminal two
```

Vite serves `http://localhost:5173` and proxies `/api` to `http://127.0.0.1:8080`.

`serve` refuses to start unless the authentication environment described in
[`DEPLOYMENT.md`](../DEPLOYMENT.md) is set; there is no authentication-disabled mode. `task dev`
does not set these either, so it needs the same variables. For local development, export:

- `GROCERY_ROUTER_AUTH_ORIGIN=http://localhost:5173` (plain HTTP is accepted only for
  `localhost` or `127.0.0.1`)
- `GROCERY_ROUTER_AUTH_GOOGLE_CLIENT_ID` and `GROCERY_ROUTER_AUTH_GOOGLE_CLIENT_SECRET` from a
  Google OAuth client that registers `http://localhost:5173/api/v2/auth/google/callback`
- `GROCERY_ROUTER_AUTH_SESSION_SECRET`: base64url, at least 32 decoded bytes
- `GROCERY_ROUTER_AUTH_BOOTSTRAP_USERS`: the Google email addresses allowed to sign in
- `GROCERY_ROUTER_AUTH_BOOTSTRAP_HOUSEHOLD=Coward`

`GROCERY_ROUTER_AUTH_OIDC_ISSUER` defaults to `https://accounts.google.com`. `serve` contacts the
issuer at startup, so it also needs network access.

Regenerate API declarations after changing `api/openapi.yaml`:

```sh
task web-generate
```

Do not edit `src/api/schema.d.ts` directly.
