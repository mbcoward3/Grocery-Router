# Grocery Router — Google Authentication and Shared Household

Status: **approved post-v1 phase; implementation authorized**  
Production origin: `https://groceries.matthewcoward.com`  
Development origin: `https://groceries-dev.matthewcoward.com`  
Original v1 authority: [`V1_SPEC.md`](V1_SPEC.md)

## 1. Product outcome

Protect Grocery Router with individual accounts while allowing the family to share one household
workspace. All household members see and mutate the same recipes, current week, grocery items, and
checklist. Anyone outside the household cannot read or change that data.

The initial release is closed: only two configured Google identities may enter. Existing recipes,
weeks, and grocery state become the `Coward` household. Both initial members are owners. There is no
public signup, invitation UI, or additional-household creation in this slice.

The identity model must permit Apple, native passwordless email, or another OIDC provider later
without replacing application users, sessions, household memberships, or product ownership.

## 2. Approved initial configuration

- Identity provider: direct Google OpenID Connect; no Clerk, Auth0, or Firebase Authentication.
- Production origin: `https://groceries.matthewcoward.com`.
- Stable development origin: `https://groceries-dev.matthewcoward.com`.
- Household: `Coward`.
- Initial owners:
  - `mbcoward3@gmail.com`
  - `brooklynjocoward@gmail.com`
- Google scopes: `openid`, `email`, and `profile` only.
- Sessions: opaque application sessions in a secure first-party cookie.
- Admission: configured verified-email allowlist for the initial release.

Production and development use separate Google OAuth clients, session secrets, databases, and
allowlists. Dynamic PR callbacks are not registered with Google.

## 3. Goals

- Smooth Google sign-in and durable sessions across browser restarts, including iPhone Safari.
- Clean application sign-out without signing the browser out of Google globally.
- Provider-independent local users that may hold multiple external identities.
- One shared household with role-based membership.
- Immediate access revocation even while an external-provider session remains valid.
- Strict tenant isolation in every authenticated query and mutation.
- Transactional migration of all existing product data into the seeded household.
- Fail-closed production and development startup configuration.
- Real OIDC behavior covered with an in-process fake provider in automated tests.

## 4. Non-goals for the initial release

- Passwords or password-reset credentials stored by Grocery Router.
- Public signup, invitations, transactional email, or self-service household creation.
- Apple sign-in, email magic links, passkeys, MFA, or enterprise SSO.
- Per-recipe permissions inside a household.
- A household switcher, despite supporting multiple memberships in the data model.
- Realtime or offline conflict resolution.
- Authentication on dynamic PR hosts through the production or development Google client.

These omissions must not produce placeholder controls in the shipped UI.

## 5. Identity and account linking

### Application user

`app_users` is the durable Grocery Router account. Provider email and profile attributes are not
its authorization key.

### External identity

`user_identities` links an application user to a verified external `(issuer, subject)`. A user may
have more than one identity. Google is the only enabled provider initially.

Provider adapters normalize a successful authentication into:

```go
type ExternalIdentity struct {
    Issuer        string
    Subject       string
    VerifiedEmail string
    DisplayName   string
    AvatarURL     string
}
```

Provider-specific tokens and behavior do not escape the adapter. Grocery Router stores no Google
access or refresh token.

### Email collision and linking

A verified provider email that matches an existing verified account must not create a duplicate,
but matching email alone must never silently link identities. The user must prove control of the
existing account through an authenticated linking or recovery flow. Linking inserts the new
external identity transactionally. Unverified native registrations must not permanently reserve an
email or own meaningful data.

Apple private-relay addresses may not match an existing email; explicit authenticated linking is
therefore the durable rule.

## 6. Authentication flow

1. A signed-out protected request records only a validated internal return path and redirects to
   `/sign-in`.
2. `GET /api/v2/auth/google/start` creates cryptographically random state, nonce, and PKCE values,
   stores their protected short-lived transaction state, and redirects to Google.
3. Google returns only to the exact configured callback origin.
4. The callback validates state, issuer, audience, signature, nonce, expiry, and PKCE before using
   any claim.
5. The adapter requires `email_verified` and returns the normalized external identity.
6. The application resolves `(issuer, subject)`. On first admission it also requires an exact
   normalized match in the configured allowlist and creates the local identity and seeded-household
   membership transactionally.
7. Grocery Router creates a random opaque session, stores only its SHA-256 digest, and sets the raw
   token in a cookie.
8. The browser returns to the validated internal path.

The initial allowlist is an admission and active-access policy. Removing an entry and restarting the
application revokes that user's product access; persisted provider identity remains audit evidence.

## 7. Sessions and browser behavior

Use a host-only cookie named `__Host-grocery_session` with `Secure`, `HttpOnly`, `SameSite=Lax`, and
`Path=/`. Localhost development may use a distinct non-secure development cookie. Never copy a
provider or application token to `localStorage`.

Sessions have an idle expiry and a longer absolute expiry. Ordinary use may renew the idle window
without extending the absolute limit. Rotate sessions after sign-in, account linking, or sensitive
role changes. Logout deletes the server-side session and expires the cookie.

Every authenticated request loads the current local user and membership. A deleted membership or
disabled user loses access on the next request even if the cookie and Google session remain valid.
Cookie-authenticated mutations require trusted-origin validation and CSRF protection.

A missing or expired application session may initiate a top-level Google flow. Browser and Google
policy may still show account selection or consent; the application must not depend on third-party
iframe silent login. Explicit logout lands on a signed-out screen instead of immediately logging the
user back in.

## 8. Household roles

Use `owner`, `admin`, and `member` roles. Members may read and edit the shared product state. Admins
may eventually manage invitations and ordinary members. Owners may change roles, transfer
ownership, and perform destructive household administration.

A household must always have an owner. The initial two users are both owners to avoid single-account
lockout. Last-owner removal, leave, or demotion is prohibited transactionally.

## 9. Logical data model

Use UUIDs for identity and tenant identifiers.

### `app_users`

- UUID ID;
- current verified primary email for display/contact;
- display name and optional avatar URL;
- created, updated, last-seen, and optional disabled timestamps.

### `user_identities`

- UUID ID and application-user ID;
- provider label for display and adapter selection;
- issuer and subject, unique together;
- provider email and whether it was verified at the last successful authentication;
- created and last-seen timestamps.

Do not auto-link identities based on email.

### `households`

- UUID ID;
- non-empty name;
- created and updated timestamps.

### `household_memberships`

- household ID and user ID, unique together;
- role (`owner`, `admin`, or `member`);
- created timestamp.

Removing access deletes the membership; do not keep a parallel active flag.

### `auth_sessions`

- UUID ID and application-user ID;
- SHA-256 digest of at least 256 bits of random token material, unique;
- created, last-seen, idle-expiry, and absolute-expiry timestamps;
- optional revoked timestamp;
- small non-authoritative client metadata when useful for session management.

Never store the raw session token.

### Household-owned product data

`store_sections`, `grocery_items`, `recipes`, and `weeks` are household aggregate roots. Their keys,
names, and week dates are unique within a household rather than globally. Cross-aggregate foreign
keys must prove both records belong to the same household. Child rows inherit tenancy from their
parent. No week recipe, shopping line, or contribution may connect different households.

Units remain immutable global reference data.

## 10. API and authorization boundary

Authenticated operations use `/api/v2`. Relevant public endpoints are limited to:

```text
GET  /healthz
GET  /sign-in
GET  /api/v2/auth/google/start
GET  /api/v2/auth/google/callback
POST /api/v2/auth/logout
```

Session and product examples:

```text
GET /api/v2/session
GET /api/v2/households/{householdID}/recipes
GET /api/v2/households/{householdID}/week/current
GET /api/v2/households/{householdID}/week/current/groceries
```

Middleware resolves the session, enabled user, explicit path household, and current membership, then
attaches immutable values to the Go context. Stores require household ID explicitly. An authenticated
product lookup by a bare numeric ID is invalid.

Use `401` for missing/expired authentication, `403` for insufficient household access, `404` rather
than disclosing another tenant's resource, and `409` for state conflicts.

The legacy unauthenticated product API must be removed or made unreachable at cutover. A partially
protected deployment is not acceptable.

## 11. Security requirements

- Validate OIDC tokens in Go; frontend route guards are usability only.
- Use authorization code flow with PKCE, state, and nonce.
- Cache discovery/JWKS with bounded refresh and support key rotation; fail closed.
- Never log authorization codes, provider tokens, session cookies, raw session tokens, or callback
  query strings.
- Use constant-time comparisons for token digests where application comparison is necessary.
- Restrict accepted origins and callback destinations to configured HTTPS origins.
- Set `Cache-Control: no-store` on auth and session responses.
- Use a restrictive Content Security Policy compatible with Google sign-in redirects.
- Treat names, avatars, and emails as untrusted display data.
- Rate-limit auth starts, callbacks, and future recovery/linking endpoints.
- Do not trust identity headers from the reverse proxy or request body.
- Do not provide an `AUTH_DISABLED`, query-string, or header-based production bypass.
- Production refuses to start without valid origin, provider, session, and allowlist configuration.

## 12. Environment and test isolation

Production and stable development use distinct Google clients and Kubernetes Secrets. Production
trusts only its configured Google issuer and audience. Development cannot access the production
database.

Automated tests run an in-process fake OIDC discovery/JWKS provider and exercise real middleware,
callback, session, and authorization code. Tests cover malformed, expired, wrong-issuer,
wrong-audience, wrong-nonce, and rotated-key tokens.

Dynamic PR previews initially fail closed and may show the signed-out shell. If authenticated PR
previews become necessary, use a dedicated test OIDC issuer trusted only by preview configuration;
do not add wildcard Google callbacks or a production auth bypass.

## 13. Existing-data migration

The migration is rehearsed against a production backup and proceeds transactionally:

1. Create users, identities, households, memberships, and session tables.
2. Create the configured `Coward` household exactly once.
3. Add nullable household keys to household-owned roots.
4. Backfill every existing row to the seeded household.
5. Add tenant-scoped uniqueness and cross-household constraints.
6. Make household keys non-null and remove old global uniqueness.
7. Let each configured owner claim their membership on first verified Google sign-in.
8. Verify row counts, contribution integrity, and existing acceptance behavior.
9. Enable authenticated v2 and remove public product access in one cutover.

Never add an implicit default household to ordinary application queries. A missing tenant predicate
must fail rather than fall into the seeded household.

## 14. Delivery slices

1. **Documentation and configuration:** approve this contract, validate startup settings, establish
   separate secrets, and create the stable dev origin.
2. **Identity foundation:** schema, provider interface, Google adapter, secure sessions, `/session`,
   signed-out shell, and fake-OIDC tests without exposing product routes differently.
3. **Household migration:** seed/backfill the household, tenant-scope schema and queries, claim both
   owners, and expose authenticated v2 product routes.
4. **Cutover:** validate production backup and migration, deploy fail closed, smoke-test both owners
   on desktop and iPhone, then eliminate unauthenticated product access.
5. **Later capabilities:** explicit identity linking, invitations, account management, Apple, and
   public onboarding only through separately approved slices.

Each slice must be deployable without exposing a partially protected production product.

## 15. Acceptance criteria

### Authentication and sessions

- Both configured owners can sign in and out through Google on desktop and iPhone Safari.
- An unlisted Google identity cannot create a usable local account or access product data.
- Sessions survive ordinary browser restarts, honor idle/absolute expiry, rotate, and revoke.
- Wrong state, nonce, issuer, audience, expiry, signature, and rotated-key behavior is tested.
- No database row or log contains Google access/refresh tokens or raw session tokens.

### Authorization and tenancy

- Every product operation succeeds for a current household member with the required role.
- It fails for signed-out, disabled, removed, and non-member users.
- Changing household, recipe, occurrence, line, or contribution IDs cannot cross tenants.
- Database constraints reject cross-household product references.
- Query audit finds no authenticated product lookup by unscoped bare ID.

### Migration and deployment

- Existing recipes, mappings, weeks, lines, completion state, overrides, and contribution traces are
  unchanged after household backfill.
- Each configured owner claims exactly one membership and repeated callbacks are idempotent.
- Production startup fails when auth configuration is absent or inconsistent.
- No unauthenticated product endpoint remains reachable after cutover.
- Development and production use different OAuth clients, sessions, and databases.

## 16. Deferred decisions

- Invitation and public-signup policy.
- How a newly created household obtains or onboards a recipe corpus.
- Native passwordless-email provider and email-delivery service.
- Apple Developer account, Services ID, private signing key, and key-rotation owner.
- Account-linking and recovery UI.
- Session-management UI and security-event audit history.
