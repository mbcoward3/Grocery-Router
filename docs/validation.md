# Platform validation record

Observed on the task host on 2026-08-09:

| Check | Result |
|---|---|
| Python | 3.12.3 |
| Unit/integration-boundary tests | 101 passed (core, planner fallback, storage, migration, web health) |
| Ingredient audit | 250/254 resolved; four known correct refusals |
| Compose | Docker Compose v2.27.0 validated `compose.yaml` and `compose.vps.yaml` |
| Local image | Built successfully from the digest-pinned Python base with hash-locked dependencies |
| Local runtime | Healthy as UID/GID 10001, read-only root, all capabilities dropped, 512 MiB/PID limits enforced |
| Local HTTP | `/health/ready` returned `200 ok` through loopback port 18765 |
| Failure gates | Missing VPS variables, missing production database configuration, and URL without `sslmode=verify-full` all failed before serving |
| Caddy | Pinned `caddy:2.10.0-alpine` digest pulled; Caddyfile plus deny-all auth seam validated with Caddy 2.10.0 |
| Kustomize | Base, both overlays, and both preserved cluster paths rendered successfully |

Port 8765 was already in use by another task on the shared host, so the documented
`GROCERY_ROUTER_PORT` override used 18765 for the runtime check. The first image build
exceeded a five-minute command timeout while Docker Desktop slowly extracted its base; the
same build completed successfully on retry and the container then became healthy. The test
container and network were removed afterward; its named development-state volume was
preserved as `docker compose down` promises.

No CockroachDB account/credential, public DNS name, VPS, or approved authentication option
was available or created. Consequently the production stack was not connected to the
internet or a live database. Database tests exercise the production store's SQL contract,
migrations, TLS/configuration gates, serialization retry, and no-fallback path. Caddy's
configuration was validated in its pinned image, and the committed authentication seam
remains deny-all until the separate review approves a mechanism.
