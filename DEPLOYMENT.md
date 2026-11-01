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
- Health endpoint: `GET /healthz`

The server fails startup if authentication configuration is absent or inconsistent. HTTPS deployments use host-only `__Host-` cookies; only an explicit localhost origin receives distinct non-secure development cookie names. There is no authentication-disabled mode.

The entrypoint applies pending Goose migrations and ingests the approved corpus only when the Coward household's recipe table is empty. CNPG owns database storage and credentials; the application pod is stateless.

## Environments and cutover

- Production: `https://groceries.matthewcoward.com`
- Stable development: `https://groceries-dev.matthewcoward.com`

Production and stable development require separate Google OAuth clients, session keys, allowlists, and CNPG databases. Each Google client registers only its environment's exact callback:

```text
https://<environment-host>/api/v2/auth/google/callback
```

The reusable application-secret helper is `scripts/configure-auth-secrets.sh`. Secret values must not be committed.

Application packaging is owned by this repository through `charts/grocery-router`. The homelab GitOps repository owns namespaces, CNPG databases, authentication Secrets, network policy, Gateway listeners, and the small Flux OCIRepository/HelmRelease registration. Before application cutover that repository must provide stable development with its own CNPG `Cluster`, application release, and environment-specific Secrets. Stable-dev acceptance must cover both owners on desktop and iPhone before any production route is changed.

At production cutover, first rehearse migrations against a production backup and verify row counts, grocery contribution integrity, overrides, removed/completed state, and both owner claims. Roll out the authenticated image and configuration atomically. The application exposes product operations only below authenticated `/api/v2/households/{householdID}` routes; legacy unauthenticated `/api` product routes return 404.

Dynamic PR hosts have no Google callback. They must remain fail-closed unless a separately approved test issuer and preview-only trust configuration are added.

## Signed OCI Helm publication

Pull requests publish commit-SHA images for preview compatibility. Successful `main` builds publish and keylessly sign the image, package the chart with that exact image digest, publish and sign an immutable chart version, and only then promote `ghcr.io/mbcoward3/charts/grocery-router:latest`. Flux verifies that the chart was signed by this repository's `container.yml` workflow on `main` before Helm applies it.

The application workflow does not clone or mutate a GitOps repository. The separate legacy preview workflow still uses `GITOPS_DEPLOY_KEY`; disable or migrate that workflow before removing the secret. The chart intentionally owns only the Deployment, Service, and optional HTTPRoute; platform resources and secret values remain outside the chart.
