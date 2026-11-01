# Grocery Router Helm chart

The chart owns only Deployment, Service, and optional HTTPRoute. Homelab owns namespaces, database/operator/storage, authentication Secrets, network policies, and shared Gateway infrastructure.

CI embeds the signed image's immutable digest before packaging. An unpackaged development chart intentionally fails without `image.digest`:

```sh
helm lint charts/grocery-router --strict --set image.digest=sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
python3 tools/test_chart.py
```

The production selector remains `app.kubernetes.io/name=grocery-router` and `app.kubernetes.io/instance=prod` for controlled adoption. Use a separate release, namespace, `instance`, auth Secret, and database Secret for stable development. Reviewable catalog releases must not be enabled in production.

Published chart: `oci://ghcr.io/mbcoward3/charts/grocery-router`. Flux follows signed `latest`; pin a historical chart digest for a lasting rollback rather than running an imperative Helm rollback against an active latest-following controller.
