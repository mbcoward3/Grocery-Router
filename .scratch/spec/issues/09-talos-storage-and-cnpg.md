# Storage class and CNPG on single-node Talos

Type: research
Status: open
Blocked by: —

## Question

What exactly has to exist on a single-node Talos cluster before CloudNativePG can run, and
what does the deployment look like end to end?

Decision 5 picked Postgres via CNPG with `instances: 1`. Talos ships **no storage class**,
so that has to come first — and it is needed for any stateful workload, not just this one.

Find and record:

1. **The storage class.** Options are local-path-provisioner, Longhorn, Rook, and
   democratic-csi. On one node, which is least work and least likely to lose data? Name the
   version and the install method.
2. **The CNPG operator and `Cluster` custom resource** for one instance. The actual
   manifest, not a description of one.
3. **Backups.** One node is one disk, so a lost disk is a lost household. What object
   storage target does CNPG support, and what is the smallest working configuration?
4. **What a Postgres minor upgrade does** to a single-instance cluster, and how long the
   database is down.
5. **The migration Job**, run before the app rolls.
6. **The probes the app must expose.** `/healthz` and `/readyz`, with readiness checking
   the database — otherwise a rolling deploy serves errors from a pod that has not
   connected.
7. **Graceful shutdown** on `SIGTERM`, so a deploy does not drop an in-flight session.
8. **The GitHub Actions pipeline**: build the TypeScript bundle, embed it, build the Go
   binary, build the image, push it, and roll the deployment. How does CI reach a home
   cluster — a pull-based agent, or a tunnel?

Resolve with a `/research` subagent against primary sources. Capture the findings as a
document and link it here.
