# Platform validation record

Observed on the task host on 2026-08-09:

| Check | Result |
|---|---|
| Python | 3.12.3 |
| Unit/integration-boundary tests | 98 passed (core, planner fallback, storage, migration, web health) |
| Ingredient audit | 250/254 resolved; four known correct refusals |
| Locked dependencies | psycopg/psycopg-binary 3.3.4 installed with required hashes into an isolated target |
| Kustomize | base, both overlays, and both cluster paths rendered with kubectl v1.36.3 / Kustomize v5.8.1 |
| Workflow | YAML parsed; Prettier 3.6.2 check passed |
| Flux client on host | v2.5.1 (below repository pin v2.9.4) |
| Talos client on host | v1.10.0 (below repository pin v1.13.8) |
| Docker client | 29.7.2 |
| Docker daemon as task user | unavailable: selected Desktop daemon not running; system socket permission denied |
| Existing Kubernetes endpoint | unrelated endpoint unreachable; no apply attempted |

`./scripts/talos-local-up.sh` stopped at its version safety check with:

```text
talosctl v1.13.8 is required; found v1.10.0
```

The authorized local cluster therefore was **not created**: current prerequisites do not
permit it, and no Docker container, network, cluster, kubeconfig, credential, or host
setting was created/deleted/changed. A container build also could not reach the selected
Docker daemon. The repository pins current clients, image base digest and dependencies and
records copy/pasteable installation/provisioning commands in `docs/platform.md`.

No CockroachDB account or connection credential was available or created. Database tests
exercise the production store's parameterized SQL contract, atomic tick behavior,
configuration/TLS gate, migration ledger/idempotence, serialization retry boundary, and
no-fallback failure path without making an external account.
