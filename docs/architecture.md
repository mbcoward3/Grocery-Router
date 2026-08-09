# Delivery architecture

```text
push/PR -> GitHub Actions tests + image build
main    -> immutable GHCR sha tag + digest in workflow summary
reviewed image-tag PR -> main -> Flux pull reconciliation -> Kubernetes
                                             |
                                             +-> CockroachDB via DATABASE_URL Secret
```

Trust boundaries:

- The planner can select meals but has no tools and never receives ingredient lines.
- CI has source read and, only on `main`, GHCR package write. It has no kubeconfig,
  database URL, deploy key, or cluster-admin credential.
- Flux has cluster reconciliation authority and reads desired state from `main`.
- Cluster differences live in Kustomize overlays; `deploy/base` is reusable.
- Credentials never enter the image or plaintext Git.
- Production selects database storage explicitly. Missing configuration, TLS verification,
  migration, or database availability fails rather than switching to local files.

## State split

Reviewable domain inputs (`profile.md`, catalogue tables, recipes, and `items.md`) remain
markdown in Git and are loaded fresh. They are the source for every deterministic list
calculation. Generated runtime state crosses `gr/storage.py`:

- `weekly_plans`: the rendered, inspectable plan document keyed by Sunday;
- `shopping_ticks`: independent checkbox state with stable deterministic keys;
- `plan_events`: append-only planning decisions;
- `schema_migrations`: explicit migration history.

`FileStore` preserves the original local-development workflow. `DatabaseStore` uses only
portable PostgreSQL SQL/types plus CockroachDB's standard serializable retry signal. The
production configuration gate prevents `FileStore` from becoming an availability fallback.

## Reconciliation and promotion

Images are commit-addressed (`sha-<full commit>`). CI exposes the corresponding digest in
its workflow summary. A reviewed overlay change is the auditable promotion record; Flux,
not CI, deploys it. Rollback reverts that Git change. A future real Talos cluster adds an
overlay and bootstraps the same Secret contract, leaving build and storage boundaries
unchanged.
