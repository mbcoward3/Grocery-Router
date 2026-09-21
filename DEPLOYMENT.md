# Grocery Router deployment

The application is packaged as one container containing the Go API, built React application, and approved bootstrap corpus. PostgreSQL runs separately under CloudNativePG (CNPG).

## Runtime contract

- Listen address: `GROCERY_ROUTER_ADDRESS` (container default `0.0.0.0:8080`)
- PostgreSQL connection: `GROCERY_ROUTER_DATABASE_URL` (required in the container)
- Static assets: `GROCERY_ROUTER_WEB_ROOT` (container default `/app/web/dist`)
- Health endpoint: `GET /healthz`

The entrypoint applies pending Goose migrations and ingests the approved corpus only when the recipe table is empty. CNPG owns database storage and credentials; the application pod is stateless.

GitHub Actions publishes commit-SHA and default-branch tags to the public package `ghcr.io/mbcoward3/grocery-router`.

A successful build from `main` commits the resulting OCI digest to the private `talos-cluster` GitOps repository. Flux then rolls out production. Production therefore follows `main` without giving GitHub-hosted runners direct Kubernetes access.

The Talos cluster installs the CNPG operator through Flux. Production uses a single-instance 5 Gi `Cluster` because the current Talos environment has one node; adding instances on that same node would not provide host-level high availability. The application receives CNPG's generated connection URI from the `grocery-router-db-app` Secret.

Pull requests build a `sha-<head-commit>` image. A separate `pull_request_target` workflow safely commits manifests derived only from trusted PR metadata; it does not execute PR code with the GitOps credential. Each preview receives:

- URL `http://pr-<number>.192-168-4-200.sslip.io`
- resources in the shared `grocery-router-dev` namespace
- a distinct single-instance 2 Gi CNPG cluster
- automatic removal when the PR closes or its 24-hour lease expires

The production deployment is available on the LAN at `http://groceries.192-168-4-200.sslip.io`. Kubernetes resources are defined in the separate `talos-cluster` GitOps repository.
