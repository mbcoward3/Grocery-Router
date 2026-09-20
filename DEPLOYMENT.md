# Grocery Router deployment

Deployment is a separately approved phase beyond the local-only v1 product scope. The application is packaged as one container containing the Go API, built React application, and approved bootstrap corpus.

## Runtime contract

- Listen address: `GROCERY_ROUTER_ADDRESS` (container default `0.0.0.0:8080`)
- SQLite path: `GROCERY_ROUTER_DATABASE` (container default `/data/grocery-router.db`)
- Static assets: `GROCERY_ROUTER_WEB_ROOT` (container default `/app/web/dist`)
- Health endpoint: `GET /healthz`
- Persistent storage: mount a writable volume at `/data`

On an empty volume the entrypoint migrates SQLite and ingests the approved corpus. On an existing volume it applies pending migrations without reimporting the corpus.

GitHub Actions publishes commit-SHA and default-branch tags to the public package `ghcr.io/mbcoward3/grocery-router`.

A successful build from `main` commits the resulting OCI digest to the private `talos-cluster` GitOps repository. Flux then rolls out production. Production therefore follows `main` without giving GitHub-hosted runners direct Kubernetes access.

Pull requests build a `sha-<head-commit>` image. A separate `pull_request_target` workflow safely commits manifests derived only from trusted PR metadata; it does not execute PR code with the GitOps credential. Each preview receives:

- URL `http://pr-<number>.192-168-4-200.sslip.io`
- resources in the shared `grocery-router-dev` namespace
- a distinct 2 Gi SQLite PVC
- automatic removal when the PR closes or its 24-hour lease expires

The production deployment is available on the LAN at `http://groceries.192-168-4-200.sslip.io`. All Kubernetes resources are defined in the separate `talos-cluster` GitOps repository.
