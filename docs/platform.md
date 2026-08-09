# Compose operator guide (current default)

Docker Compose is the current local and single-VPS deployment path. Kubernetes, Talos, and
Flux remain available as an optional future path in
[`platform-kubernetes.md`](platform-kubernetes.md); none is required for routine use.
GitHub Actions only tests, audits, builds, and publishes GHCR images. A VPS operator pulls
and restarts an explicitly selected immutable image.

## Local Compose

Prerequisites are Python 3.12 for repository tests and a supported Docker Engine or Docker
Desktop with the Compose v2 plugin. From the repository root:

```sh
python3 -m unittest discover -s tests
python3 -m gr.audit
docker compose up --build -d
docker compose ps
curl --fail http://127.0.0.1:8765/health/ready
```

Open <http://127.0.0.1:8765>. The app is non-root, has no Linux capabilities, has a
read-only root filesystem and bounded CPU, memory, and PIDs. Mutable development plans,
ticks, and events live in the `grocery-router-state` Docker volume. Direct
`python3 -m gr.web` runs retain the repository-backed `weeks/` and `decisions.jsonl`
workflow. To inspect or stop Compose:

```sh
docker compose logs -f --tail=100 app
docker compose down                 # preserves grocery-router-state
```

The local port is loopback-only. Override it with `GROCERY_ROUTER_PORT=9000 docker compose
up --build -d`; do not publish this development service to the internet.

## VPS architecture and authentication boundary

```text
internet :80/:443 -> Caddy (automatic TLS + authentication seam)
                         |
                         +-> private Compose network -> app :8765 -> CockroachDB TLS
```

Only Caddy publishes host ports. The app and one-shot migration service are non-root,
read-only, capability-free, resource-bounded containers on a private bridge. Caddy runs as
the unprivileged VPS operator UID on internal ports 8080/8443, while Docker maps public
80/443; it also has no capabilities. Caddy data stays in owner-only host directories.

**The committed authentication seam denies every request with 503.** This is intentional:
the app must never be reachable from the internet unauthenticated. A parallel review will
choose the smallest suitable authentication mechanism. Only then should an operator set
`GROCERY_ROUTER_AUTH_CADDYFILE` to an owner-readable Caddy snippet that authenticates every
request. Do not use an empty snippet or one that only proxies. This guide does not choose
Basic Auth, Tailscale, Cloudflare Access, or application OIDC ahead of that review. TLS can
be provisioned and the stack/database validated while the deny-all guard remains active,
but internet use remains closed until an approved authentication snippet is installed.

## Hetzner VPS prerequisites and hardening boundary

Use a currently supported x86-64 Debian or Ubuntu release and install current Docker Engine
plus the Compose v2 plugin from Docker's official repository. Apply security updates and
reboot when the kernel requires it. The operator assumptions are deliberately narrow:

- one unprivileged SSH user with key-only login, no direct root login, and tightly limited
  sudo; membership in the `docker` group is root-equivalent and must be treated that way;
- repository and `.local/` files owned by that operator, with `.env.vps`, the database URL
  file, and private auth material mode `0600`, and Caddy state directories mode `0700`;
- Hetzner Cloud Firewall (and a host firewall that accounts for Docker's forwarding rules)
  allowing TCP 80/443 globally, UDP 443 if QUIC is desired, and TCP 22 only from approved
  operator source addresses; no rule for 8765;
- outbound DNS, HTTPS, and CockroachDB connectivity (normally TCP 26257). If the Cockroach
  Cloud cluster uses an IP allowlist, add the VPS's stable egress address;
- an operator-owned domain with A and, only when IPv6 is configured, AAAA records pointing
  at the VPS before Caddy starts. This repository neither invents nor provisions a domain;
- an existing CockroachDB Serverless/Basic database and SQL user. Its PostgreSQL URL must
  use `sslmode=verify-full`; URL-encode credential special characters.

CockroachDB is the durable production state. Confirm the Cockroach Cloud backup/retention
policy meets the household's recovery objective and periodically test the provider's
restore/export procedure. The VPS has no database volume to back up. Separately protect
the database URL in an approved secret/password manager and back up Caddy's `.local` state
if avoiding certificate reissuance matters. Never put database or auth credentials in
Git, an image, a ticket, or CI.

Docker and the host OS are the operator's patching boundary. Compose does not configure
SSH, firewall policy, unattended upgrades, DNS, Hetzner resources, Cockroach accounts, or
external credentials.

## First VPS deploy

Clone a reviewed release of this repository as the unprivileged operator. Obtain the full
`tag@sha256:digest` reference from the successful `main` workflow summary; do not use
`latest`, `main`, or an unqualified mutable tag.

```sh
git clone https://github.com/mbcoward3/Grocery-Router.git
cd Grocery-Router
cp .env.vps.example .env.vps
chmod 600 .env.vps
install -d -m 700 .local/caddy/data .local/caddy/config .local/secrets
umask 077
read -rsp 'CockroachDB DATABASE_URL: ' DATABASE_URL && echo
printf '%s\n' "$DATABASE_URL" > .local/secrets/database-url
unset DATABASE_URL
chmod 600 .local/secrets/database-url
id -u; id -g                    # put these exact numbers in .env.vps
$EDITOR .env.vps                # set image@digest, domain, email, UID/GID, and secret path
```

Leave `GROCERY_ROUTER_AUTH_CADDYFILE` unset for the safe deny-all posture. After the auth
review approves a mechanism, write its Caddy snippet under `.local/secrets/`, `chmod 600`
it, and set the path in `.env.vps`. Validate required substitutions without printing the
rendered configuration, pull, migrate, and start:

```sh
test "$(stat -c %a .env.vps)" = 600
test "$(stat -c %a .local/secrets/database-url)" = 600
IMAGE_REF="$(grep '^GROCERY_ROUTER_IMAGE=' .env.vps | cut -d= -f2-)"
printf '%s\n' "$IMAGE_REF" | grep -Eq '^ghcr\.io/mbcoward3/grocery-router:sha-[0-9a-f]{40}@sha256:[0-9a-f]{64}$'
unset IMAGE_REF
docker compose --env-file .env.vps -f compose.vps.yaml config --quiet
docker compose --env-file .env.vps -f compose.vps.yaml pull
docker compose --env-file .env.vps -f compose.vps.yaml run --rm migrate
docker compose --env-file .env.vps -f compose.vps.yaml up -d app caddy
```

Compose rejects missing image, domain, email, UID/GID, or database-secret path before
startup. The application reads the URL from the read-only container secret and independently
rejects a missing/unreadable file, non-PostgreSQL URL, production file storage, or any
production URL without `sslmode=verify-full`; it never falls back to JSON files. Migration
or database failure prevents the app becoming healthy and therefore prevents Caddy from
starting.

## Verify, observe, and stop

```sh
docker compose --env-file .env.vps -f compose.vps.yaml ps
docker compose --env-file .env.vps -f compose.vps.yaml exec app \
  python -c "import urllib.request; urllib.request.urlopen('http://127.0.0.1:8765/health/ready', timeout=3).read(); print('ready')"
docker compose --env-file .env.vps -f compose.vps.yaml logs --tail=100 app caddy
curl -sS -o /dev/null -w 'HTTPS status: %{http_code}\n' "https://$(grep '^GROCERY_ROUTER_DOMAIN=' .env.vps | cut -d= -f2-)/"
```

An unauthenticated external request must never return the app (`200`). The committed guard
returns `503`; an approved authentication layer will normally return a challenge or
redirect. Verify an authenticated browser session separately after that layer is chosen.
For ongoing logs or a clean stop (without deleting Caddy state):

```sh
docker compose --env-file .env.vps -f compose.vps.yaml logs -f --tail=100 app caddy
docker compose --env-file .env.vps -f compose.vps.yaml down
```

## Update and rollback

Record the current `GROCERY_ROUTER_IMAGE`, replace it in `.env.vps` with the new successful
`main` workflow's complete `tag@digest`, then run the explicit release step:

```sh
chmod 600 .env.vps
IMAGE_REF="$(grep '^GROCERY_ROUTER_IMAGE=' .env.vps | cut -d= -f2-)"
printf '%s\n' "$IMAGE_REF" | grep -Eq '^ghcr\.io/mbcoward3/grocery-router:sha-[0-9a-f]{40}@sha256:[0-9a-f]{64}$'
unset IMAGE_REF
docker compose --env-file .env.vps -f compose.vps.yaml config --quiet
docker compose --env-file .env.vps -f compose.vps.yaml pull
docker compose --env-file .env.vps -f compose.vps.yaml run --rm migrate
docker compose --env-file .env.vps -f compose.vps.yaml up -d app caddy
docker compose --env-file .env.vps -f compose.vps.yaml ps
```

Rollback changes only `GROCERY_ROUTER_IMAGE` back to the recorded previous digest and
re-runs `pull` and `up -d`. Do not reverse durable data migrations during image rollback;
keep schema changes expand/contract and backward-compatible. There is intentionally no CI
SSH deploy, cluster credential, webhook control plane, or required auto-updater.
