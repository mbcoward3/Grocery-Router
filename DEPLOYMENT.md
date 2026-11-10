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

## Household time zone

Each household's week begins at midnight on Sunday in its own time zone, stored in PostgreSQL (default `UTC`). Household members set it on the Settings screen, together with the number of recipes a new week offers (D049). The operator command remains for setting it without the app, for example:

```sh
kubectl exec -n <namespace> deploy/<release> -- grocery-router household-timezone America/Chicago
```

The container already has `GROCERY_ROUTER_DATABASE_URL`. `--household <id>` selects another household; it defaults to the Coward household. An unknown zone or household exits non-zero and changes nothing. Until the zone is set on the Settings screen or by the command, weeks roll over at UTC midnight as before.

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

Successful same-repository PR builds (opened, reopened, and every subsequent push), `main` pushes, and manual runs promote their immutable image digest to the shared development Deployment in `talos-cluster` at `apps/grocery-router/dev/deployment.yaml`. The last successful promotion wins at `https://groceries-dev.matthewcoward.com`; there is no per-PR environment. Validation and image publication must succeed before promotion. Fork PRs validate only and cannot publish or deploy. Deployments are serialized, and GitOps push conflicts are retried. Flux reconciliation is asynchronous; a successful workflow means the GitOps update was accepted, not that rollout health was verified.

All builds share the existing development database, Google client, authentication Secret, and origin. Schema/data changes persist between PR deployments: deploying older code does not roll back migrations. Only the image field is changed; production resources and development secrets are untouched.

 Successful `main` builds publish and keylessly sign the image, package the chart with that exact image digest, publish and sign an immutable chart version, and only then promote `ghcr.io/mbcoward3/charts/grocery-router:latest`. Flux verifies that the chart was signed by this repository's `container.yml` workflow on `main` before Helm applies it.

The development promotion job requires the repository secret `GITOPS_DEPLOY_KEY` with write access to `mbcoward3/talos-cluster`. Production continues using signed OCI chart promotion, not GitOps writeback. The legacy per-PR preview workflow is removed; its rendering/pruning scripts remain available for cleanup of existing `apps/grocery-router/previews/pr-*` resources in GitOps. Remove those stale preview directories and regenerate their parent kustomization during cleanup; no new previews or scheduled pruning will run. The chart intentionally owns only the Deployment, Service, and optional HTTPRoute; platform resources and secret values remain outside the chart.
