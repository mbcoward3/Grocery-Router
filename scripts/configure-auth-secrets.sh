#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: scripts/configure-auth-secrets.sh <prod|dev>

Interactively creates or updates the Grocery Router Google auth Secret without
putting credentials in shell history. An existing session secret is preserved.
Set ROTATE_SESSION_SECRET=1 to deliberately replace it and invalidate protected
OAuth transaction state.
EOF
}

case "${1:-}" in
  prod)
    namespace='grocery-router-prod'
    origin='https://groceries.matthewcoward.com'
    ;;
  dev)
    namespace='grocery-router-dev'
    origin='https://groceries-dev.matthewcoward.com'
    ;;
  *)
    usage >&2
    exit 2
    ;;
esac

for command in kubectl openssl base64; do
  command -v "$command" >/dev/null || {
    echo "required command not found: $command" >&2
    exit 1
  }
done

context=$(kubectl config current-context)
printf 'Kubernetes context: %s\nNamespace: %s\nOrigin: %s\n' "$context" "$namespace" "$origin"
read -r -p 'Continue? [y/N] ' confirmed
if [[ ! "$confirmed" =~ ^[Yy]$ ]]; then
  echo 'cancelled'
  exit 1
fi

read -r -p 'Google client ID: ' google_client_id
read -r -s -p 'Google client secret: ' google_client_secret
printf '\n'

if [[ -z "$google_client_id" || -z "$google_client_secret" ]]; then
  echo 'client ID and secret are required' >&2
  exit 1
fi

secret_name='grocery-router-auth'
if [[ "${ROTATE_SESSION_SECRET:-0}" == '1' ]]; then
  session_secret=$(openssl rand -hex 32)
elif kubectl --namespace "$namespace" get secret "$secret_name" >/dev/null 2>&1; then
  session_secret=$(
    kubectl --namespace "$namespace" get secret "$secret_name" \
      -o jsonpath='{.data.GROCERY_ROUTER_AUTH_SESSION_SECRET}' | base64 --decode
  )
  if [[ -z "$session_secret" ]]; then
    echo "existing $secret_name has no session secret; refusing to replace it implicitly" >&2
    exit 1
  fi
else
  session_secret=$(openssl rand -hex 32)
fi

bootstrap_users='[{"email":"mbcoward3@gmail.com","role":"owner"},{"email":"brooklynjocoward@gmail.com","role":"owner"}]'

kubectl --namespace "$namespace" create secret generic "$secret_name" \
  --from-literal=GROCERY_ROUTER_AUTH_ORIGIN="$origin" \
  --from-literal=GROCERY_ROUTER_AUTH_GOOGLE_CLIENT_ID="$google_client_id" \
  --from-literal=GROCERY_ROUTER_AUTH_GOOGLE_CLIENT_SECRET="$google_client_secret" \
  --from-literal=GROCERY_ROUTER_AUTH_SESSION_SECRET="$session_secret" \
  --from-literal=GROCERY_ROUTER_AUTH_BOOTSTRAP_HOUSEHOLD='Coward' \
  --from-literal=GROCERY_ROUTER_AUTH_BOOTSTRAP_USERS="$bootstrap_users" \
  --dry-run=client -o yaml | kubectl apply -f -

unset google_client_id google_client_secret session_secret
echo "configured $secret_name in $namespace"
