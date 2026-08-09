# Optional Kubernetes, Talos, and Flux guide

This material preserves the already-landed cluster path for a possible future migration.
It is **not required** for local use or the current single-VPS deployment. Start with the
[Compose operator guide](platform.md). If a cluster is later authorized, CI publishes an
image; Git records the desired image; Flux pulls Git and reconciles Kubernetes. CI never
receives a kubeconfig. CockroachDB Serverless holds generated plans, shopping ticks, and
decision events through a PostgreSQL-compatible `DATABASE_URL`; catalogue/profile/recipe
markdown remains image input.

## Pinned prerequisites

Versions were reviewed on 2026-08-09. `scripts/platform-versions.env` is the single pin.

| Tool | Pin / requirement |
|---|---|
| Talos / `talosctl` | v1.13.8 |
| Kubernetes / `kubectl` | v1.36.3 |
| Flux | v2.9.4 |
| Docker | 18.03+ (Talos' documented minimum) |
| Python | 3.12 |

Install tools without changing host-wide configuration (Linux amd64 shown):

```sh
mkdir -p .local/bin
curl -fL https://github.com/siderolabs/talos/releases/download/v1.13.8/talosctl-linux-amd64 -o .local/bin/talosctl
curl -fL https://dl.k8s.io/release/v1.36.3/bin/linux/amd64/kubectl -o .local/bin/kubectl
curl -fL https://github.com/fluxcd/flux2/releases/download/v2.9.4/flux_2.9.4_linux_amd64.tar.gz -o .local/flux.tgz
tar -xzf .local/flux.tgz -C .local/bin flux
chmod +x .local/bin/talosctl .local/bin/kubectl .local/bin/flux
export PATH="$PWD/.local/bin:$PATH"
```

References used for version-sensitive choices:

- [Talos v1.13 Docker platform guide](https://docs.siderolabs.com/talos/v1.13/platform-specific-installations/local-platforms/docker)
- [Talos v1.13.8 release](https://github.com/siderolabs/talos/releases/tag/v1.13.8)
- [Flux GitHub bootstrap](https://fluxcd.io/flux/installation/bootstrap/github/)
- [Kubernetes Kustomize guide](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/kustomization/)
- [CockroachDB connection strings](https://www.cockroachlabs.com/docs/stable/connect-to-the-database.html)
- [Cockroach Cloud Basic cluster](https://www.cockroachlabs.com/docs/cockroachcloud/create-a-basic-cluster)
- [GitHub's GHCR Actions guide](https://docs.github.com/en/actions/use-cases-and-examples/publishing-packages/publishing-docker-images)

## 1. Validate and run locally

The development default deliberately remains reviewable file storage. Production can never
select it: `APP_ENV=production` without a TLS-verified `DATABASE_URL` exits with a
configuration error.

```sh
python3 -m unittest discover -s tests
python3 -m gr.audit
./scripts/validate-manifests.sh

docker build --pull -t grocery-router:dev .
docker run --rm --name grocery-router-dev-app \
  -e APP_ENV=development \
  -e GROCERY_ROUTER_STORAGE=file \
  -p 8765:8765 \
  grocery-router:dev
# In another shell:
curl --fail http://127.0.0.1:8765/health/ready
```

The image runs as UID/GID 10001, installs only hash-locked dependencies, contains no
credentials, and receives all environment-specific configuration at runtime.

## 2. Optionally provision the named local Talos cluster

For cluster-path development only, the upstream path is `talosctl cluster create docker`.
The wrapper adds a unique name,
CIDR, isolated task-local Talos/kubeconfig files, and only the NodePort used by this app:

```sh
./scripts/talos-local-up.sh
```

It creates exactly `grocery-router-dev` on `10.77.0.0/24`, waits for its one control plane
and one worker, checks both APIs, runs `flux check --pre`, and client-validates the local
overlay. If task-managed state already exists it validates rather than replacing it. If a
same-named Docker object exists without that state, it refuses to adopt or alter it.

Useful checks:

```sh
export TALOSCONFIG="$PWD/.local/talos/talosconfig"
export KUBECONFIG="$PWD/.local/talos/kubeconfig"
talosctl --context grocery-router-dev version
kubectl --context admin@grocery-router-dev get nodes -o wide
```

Cleanup is never automatic. It requires the exact cluster name and calls only Talos'
scoped destroy command:

```sh
./scripts/talos-local-down.sh --confirm grocery-router-dev
```

## 3. Select an immutable desired image

A push to `main` publishes two tags for one digest:

```text
ghcr.io/mbcoward3/grocery-router:sha-<40-character-commit>
ghcr.io/mbcoward3/grocery-router:main-<run-number>-sha-<40-character-commit>
```

The workflow summary prints `tag@sha256:digest`. It does not contact Kubernetes. Ensure
the GHCR package is readable by the cluster, then replace the all-zero `newTag` in
`deploy/overlays/local/kustomization.yaml` with the published `sha-...` tag in a reviewed
PR. Promote to a real cluster by changing only that cluster's overlay.

## 4. Bootstrap the database secret and migrate

Do not create an account or cluster from automation. In the Cockroach Cloud console,
create/select the approved Serverless/Basic cluster and SQL user, copy its PostgreSQL URL,
and ensure it contains `sslmode=verify-full`. Never commit it.

For a direct local process, put only the URL in an ignored, owner-readable file:

```sh
mkdir -p .local
umask 077
read -rsp 'CockroachDB DATABASE_URL: ' DATABASE_URL && echo
printf '%s\n' "$DATABASE_URL" > .local/database-url
export APP_ENV=production GROCERY_ROUTER_STORAGE=database
export DATABASE_URL="$(cat .local/database-url)"
python3 -m pip install --target .local/python --require-hashes -r requirements.lock
PYTHONPATH="$PWD/.local/python" python3 -m gr.migrate
PYTHONPATH="$PWD/.local/python" python3 -m gr.web
unset DATABASE_URL
```

For Kubernetes, bootstrap the namespace and Secret imperatively. The secret value is read
without placing it in shell history or Git:

```sh
export KUBECONFIG="$PWD/.local/talos/kubeconfig"
kubectl apply -f deploy/base/namespace.yaml
read -rsp 'CockroachDB DATABASE_URL: ' DATABASE_URL && echo
kubectl -n grocery-router create secret generic grocery-router-secrets \
  --from-literal=database-url="$DATABASE_URL" \
  --dry-run=client -o yaml | kubectl apply -f -
unset DATABASE_URL
```

`grocery-router-secrets/database-url` is the contract every overlay uses. The Deployment's
non-root, read-only init container runs `python -m gr.migrate` before the app. Migrations
are numbered SQL in `migrations/`, tracked in `schema_migrations`, idempotent, and retried
on CockroachDB serialization conflicts. To inspect migration completion:

```sh
kubectl -n grocery-router logs deployment/grocery-router -c migrate
```

For long-lived clusters, replace imperative secret bootstrap with SOPS-encrypted Git and
[Flux SOPS decryption](https://fluxcd.io/flux/guides/mozilla-sops/). Commit only ciphertext;
keep the age private key in the cluster bootstrap secret, never in this repository.

## 5. Bootstrap Flux (requires explicit authorization)

This repository is public, but official GitHub bootstrap still pushes Flux manifests and
requires a GitHub credential. Do **not** run this until the captain authorizes both a
fine-grained PAT and the bootstrap commit to `main`. Use HTTPS token auth so bootstrap
does not create a deploy key. Flux documents these existing-repository PAT permissions:
Administration read, Contents read/write, and Metadata read.

```sh
export KUBECONFIG="$PWD/.local/talos/kubeconfig"
read -rsp 'GitHub bootstrap PAT: ' GITHUB_TOKEN && export GITHUB_TOKEN && echo
flux check --pre
flux bootstrap github \
  --token-auth \
  --owner=mbcoward3 \
  --repository=Grocery-Router \
  --branch=main \
  --path=clusters/local
unset GITHUB_TOKEN
```

Bootstrap adds `clusters/local/flux-system/`; review that commit like any other change.
Afterward, all application delivery is GitOps:

```sh
flux reconcile source git flux-system
flux reconcile kustomization flux-system --with-source
flux get all -A
```

## 6. Deploy, observe, and roll back

Once the Secret exists and the desired image tag is real, Flux applies the namespace,
ConfigMap, migration init container, Deployment, and Service:

```sh
kubectl -n grocery-router rollout status deployment/grocery-router --timeout=3m
kubectl -n grocery-router get deploy,pod,service
kubectl -n grocery-router logs deployment/grocery-router --tail=100
curl --fail http://127.0.0.1:30080/health/ready
```

`/health/live` checks the process; `/health/ready` checks the configured store. Database
failure makes readiness fail and requests error—production never silently writes to a
container directory. Missing model CLI/API access is different: deterministic code still
chooses a valid meal pool and records the planner error.

Rollback is a Git revert, not a CI command with cluster credentials:

```sh
git revert <the-image-promotion-commit>
git push origin <review-branch>
# Merge the rollback PR, then optionally accelerate reconciliation:
flux reconcile kustomization flux-system --with-source
```

Migration `001_initial.sql` is additive and compatible with the previous image. Keep future
changes expand/contract and backward-compatible before rollout; never try to undo durable
shopping state as part of an image rollback.

## 7. Move to a real Talos cluster

`deploy/base` is cluster-independent. `deploy/overlays/local` adds only Docker Talos'
NodePort. For a real cluster:

1. Copy `deploy/overlays/production` to a named overlay and add only ingress, sizing, and
   topology configuration.
2. Copy `clusters/production` to `clusters/<real-name>` and point it at that overlay.
3. Bootstrap the same `grocery-router-secrets/database-url` contract (or SOPS ciphertext).
4. Bootstrap Flux to that path after separately authorizing Git credentials.
5. Promote the same immutable `sha-...` image by PR.

The PostgreSQL URL and secret encryption/bootstrap change; no CI workflow, image, storage
API, migration command, or reusable Kubernetes resource needs redesign.
