# Grocery Router deployment

The application is packaged as one container containing the Go API, built React application, and approved bootstrap corpus. PostgreSQL runs separately under CloudNativePG (CNPG).

## Runtime contract

- Listen address: `GROCERY_ROUTER_ADDRESS` (container default `0.0.0.0:8080`)
- PostgreSQL connection: `GROCERY_ROUTER_DATABASE_URL` (required in the container)
- Static assets: `GROCERY_ROUTER_WEB_ROOT` (container default `/app/web/dist`)
- Canonical HTTPS origin: `GROCERY_ROUTER_AUTH_ORIGIN`
- OIDC issuer: `GROCERY_ROUTER_AUTH_OIDC_ISSUER` (`https://accounts.google.com` for Google)
- Google client: `GROCERY_ROUTER_AUTH_GOOGLE_CLIENT_ID` and `GROCERY_ROUTER_AUTH_GOOGLE_CLIENT_SECRET`
- Session signing key: `GROCERY_ROUTER_AUTH_SESSION_SECRET` (base64url, at least 32 decoded bytes)
- Active verified-email policy: `GROCERY_ROUTER_AUTH_BOOTSTRAP_USERS` (owner JSON)
- Optional review release: `GROCERY_ROUTER_CATALOG_RELEASE` (stable development only unless the release is fully approved)
- Health endpoint: `GET /healthz`

The server fails startup if authentication configuration is absent or inconsistent. HTTPS deployments use host-only `__Host-` cookies; only an explicit localhost origin receives distinct non-secure development cookie names. There is no authentication-disabled mode.

The entrypoint applies pending Goose migrations and ingests the approved corpus only when the Coward household's recipe table is empty. When `GROCERY_ROUTER_CATALOG_RELEASE` is set, it then idempotently publishes that committed release before serving. Reviewable releases expose non-adoptable development previews and must not be configured in production. CNPG owns database storage and credentials; the application pod is stateless.

## Environments and cutover

- Production: `https://groceries.matthewcoward.com`
- Stable development: `https://groceries-dev.matthewcoward.com`

Production and stable development require separate Google OAuth clients, session keys, allowlists, and CNPG databases. Each Google client registers only its environment's exact callback:

```text
https://<environment-host>/api/v2/auth/google/callback
```

The reusable application-secret helper is `scripts/configure-auth-secrets.sh`. Secret values must not be committed.

Deployment resources are owned by the separate GitOps repository. Before application cutover that repository must provide stable development with its own CNPG `Cluster`, application `Deployment`, `Service`, and HTTPS `HTTPRoute`, wired to the development `grocery-router-auth` and database Secrets. Stable-dev acceptance must cover both owners on desktop and iPhone before any production image or route is changed.

At production cutover, first rehearse migrations against a production backup and verify row counts, grocery contribution integrity, overrides, removed/completed state, and both owner claims. Roll out the authenticated image and configuration atomically. The application exposes product operations only below authenticated `/api/v2/households/{householdID}` routes; legacy unauthenticated `/api` product routes return 404.

Dynamic PR hosts have no Google callback. They must remain fail-closed unless a separately approved test issuer and preview-only trust configuration are added.

## Image publication

GitHub Actions publishes commit-SHA and default-branch tags to `ghcr.io/mbcoward3/grocery-router`. GitOps controls promotion and rollout; application work in this repository must not directly deploy or mutate production resources.
