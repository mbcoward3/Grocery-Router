#!/usr/bin/env bash
set -euo pipefail

repo=${1:?GitOps checkout is required}
action=${2:?PR action is required}
pr=${3:?PR number is required}
sha=${4:?head SHA is required}

[[ "$pr" =~ ^[0-9]+$ ]] || { echo 'Invalid PR number' >&2; exit 1; }
[[ "$sha" =~ ^[0-9a-f]{40}$ ]] || { echo 'Invalid commit SHA' >&2; exit 1; }

root="$repo/apps/grocery-router/previews"
dir="$root/pr-$pr"

if [[ "$action" == closed ]]; then
  rm -rf "$dir"
  "$(dirname "$0")/render-preview-kustomization.sh" "$root"
  exit 0
fi

# The container workflow runs independently. Wait until its public immutable tag exists.
tag="sha-$sha"
manifest="https://ghcr.io/v2/mbcoward3/grocery-router/manifests/$tag"
for attempt in {1..80}; do
  token=$(curl -fsSL 'https://ghcr.io/token?service=ghcr.io&scope=repository:mbcoward3/grocery-router:pull' \
    | python3 -c 'import json,sys; print(json.load(sys.stdin)["token"])')
  if curl -fsSI -H "Authorization: Bearer $token" \
      -H 'Accept: application/vnd.oci.image.index.v1+json' "$manifest" >/dev/null; then
    break
  fi
  if (( attempt == 80 )); then
    echo "Timed out waiting for $tag" >&2
    exit 1
  fi
  sleep 10
done

mkdir -p "$dir"
date -u -d '+24 hours' --iso-8601=seconds > "$dir/.expires-at"

cat > "$dir/kustomization.yaml" <<EOF
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - resources.yaml
EOF

cat > "$dir/resources.yaml" <<EOF
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: grocery-router-pr-$pr-db
  namespace: grocery-router-dev
spec:
  instances: 1
  imageName: ghcr.io/cloudnative-pg/postgresql:18.4-system-trixie
  bootstrap:
    initdb:
      database: grocery_router
      owner: grocery_router
  storage:
    storageClass: local-path
    size: 2Gi
  resources:
    requests: {cpu: 50m, memory: 192Mi}
    limits: {cpu: 500m, memory: 512Mi}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: grocery-router-pr-$pr
  namespace: grocery-router-dev
spec:
  replicas: 1
  strategy:
    type: Recreate
  selector:
    matchLabels:
      app.kubernetes.io/name: grocery-router
      app.kubernetes.io/instance: pr-$pr
  template:
    metadata:
      labels:
        app.kubernetes.io/name: grocery-router
        app.kubernetes.io/instance: pr-$pr
    spec:
      automountServiceAccountToken: false
      securityContext:
        runAsNonRoot: true
        runAsUser: 65532
        runAsGroup: 65532
        seccompProfile:
          type: RuntimeDefault
      containers:
        - name: grocery-router
          image: ghcr.io/mbcoward3/grocery-router:$tag
          imagePullPolicy: IfNotPresent
          env:
            - name: GROCERY_ROUTER_DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: grocery-router-pr-$pr-db-app
                  key: uri
          ports:
            - name: http
              containerPort: 8080
          readinessProbe:
            httpGet: {path: /healthz, port: http}
          livenessProbe:
            httpGet: {path: /healthz, port: http}
            periodSeconds: 30
          startupProbe:
            httpGet: {path: /healthz, port: http}
            periodSeconds: 5
            failureThreshold: 24
          resources:
            requests: {cpu: 25m, memory: 96Mi}
            limits: {cpu: 500m, memory: 384Mi}
          securityContext:
            allowPrivilegeEscalation: false
            capabilities: {drop: [ALL]}
            readOnlyRootFilesystem: true
          volumeMounts:
            - {name: tmp, mountPath: /tmp}
      volumes:
        - name: tmp
          emptyDir: {sizeLimit: 128Mi}
---
apiVersion: v1
kind: Service
metadata:
  name: grocery-router-pr-$pr
  namespace: grocery-router-dev
spec:
  selector:
    app.kubernetes.io/name: grocery-router
    app.kubernetes.io/instance: pr-$pr
  ports:
    - {name: http, port: 80, targetPort: http}
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: grocery-router-pr-$pr
  namespace: grocery-router-dev
spec:
  parentRefs:
    - {name: lan, namespace: gateway-system}
  hostnames:
    - pr-$pr.192-168-4-200.sslip.io
  rules:
    - backendRefs:
        - {name: grocery-router-pr-$pr, port: 80}
EOF

"$(dirname "$0")/render-preview-kustomization.sh" "$root"
