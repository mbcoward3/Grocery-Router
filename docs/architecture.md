# Delivery architecture

## Current Compose path

```text
push/PR -> GitHub Actions tests + audit + image build
main    -> immutable GHCR sha tag + digest in workflow summary
                                      |
VPS operator selects tag@digest ------+
  -> docker compose pull -> migrate -> restart
  -> Caddy automatic TLS + required authentication seam
  -> private app container -> CockroachDB via DATABASE_URL
```

Docker Compose is the primary local and single-VPS path. Release is an explicit operator
pull/migrate/restart; CI has no SSH key, host credential, database URL, or deployment
control plane. The app has no published VPS port. Caddy is the only ingress and the
committed authentication seam denies all requests until an approved authentication option
is installed. See [`platform.md`](platform.md).

## Optional future cluster path

The existing `deploy/` Kustomize resources, `clusters/` Flux paths, and Talos helper scripts
are preserved. A future reviewed image-tag PR can let Flux reconcile the same app and
CockroachDB contract onto Kubernetes. That option does not make Talos, Kubernetes, or Flux
a prerequisite today; see [`platform-kubernetes.md`](platform-kubernetes.md).

## Trust boundaries

- The planner can select meals but has no tools and never receives ingredient lines.
- CI has source read and, only on `main`, GHCR package write. It has no database, VPS,
  kubeconfig, deploy-key, or cluster-admin credential.
- Production credentials never enter the image, container environment, or plaintext Git.
  The owner-readable CockroachDB URL file is mounted read-only as a Compose secret.
- Production selects database storage explicitly. Missing configuration, TLS verification,
  migration, or database availability fails rather than switching to local files.
- The internet cannot address the app container directly. Caddy terminates TLS and must
  authenticate before proxying; the repository's safe default proxies nothing.

## State split

Reviewable domain inputs (`profile.md`, catalogue tables, recipes, and `items.md`) remain
markdown in Git and are loaded fresh. They are the source for every deterministic list
calculation. Generated runtime state crosses `gr/storage.py`:

- `weekly_plans`: the rendered, inspectable plan document keyed by Sunday;
- `shopping_ticks`: independent checkbox state with stable deterministic keys;
- `plan_events`: append-only planning decisions;
- `schema_migrations`: explicit migration history.

`FileStore` preserves local development; Compose places that state on a named volume while
direct Python runs use repository files. `DatabaseStore` uses portable PostgreSQL SQL/types
plus CockroachDB's standard serializable retry signal. The production configuration gate
prevents `FileStore` from becoming an availability fallback.

## Promotion and rollback

Images are commit-addressed (`sha-<full commit>`) and CI exposes `tag@sha256:digest` in its
workflow summary. The VPS operator records the old reference, substitutes the reviewed new
reference, pulls, runs additive migrations, and recreates the service. Rollback restores
the prior digest; it does not reverse durable database state.

If the optional cluster path is adopted later, a reviewed overlay change becomes the
promotion record and Flux performs reconciliation. The image, storage boundary, migration
command, and CI credential boundary remain unchanged.
