# Grocery Router deployment

Deployment is a separately approved phase beyond the local-only v1 product scope. The application is packaged as one container containing the Go API, built React application, and approved bootstrap corpus.

## Runtime contract

- Listen address: `GROCERY_ROUTER_ADDRESS` (container default `0.0.0.0:8080`)
- SQLite path: `GROCERY_ROUTER_DATABASE` (container default `/data/grocery-router.db`)
- Static assets: `GROCERY_ROUTER_WEB_ROOT` (container default `/app/web/dist`)
- Health endpoint: `GET /healthz`
- Persistent storage: mount a writable volume at `/data`

On an empty volume the entrypoint migrates SQLite and ingests the approved corpus. On an existing volume it applies pending migrations without reimporting the corpus.

GitHub Actions publishes branch, commit-SHA, and default-branch tags to `ghcr.io/mbcoward3/grocery-router`. Kubernetes deployments should use immutable `sha-<commit>` tags rather than mutable branch tags.

Development previews and the production deployment are defined in the separate `talos-cluster` GitOps repository.
