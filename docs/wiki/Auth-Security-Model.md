# Auth Security Model

CapitalFlow auth uses explicit tokens, server-side refresh sessions, audit logging, and defensive account controls.

## Token Model

* Access tokens are JWTs.
* Access tokens are sent as `Authorization: Bearer <token>`.
* Access tokens include the user ID and refresh session ID.
* JWT middleware validates the token and checks that the referenced refresh session is still active.

## Refresh Token Model

* Refresh tokens are opaque random tokens.
* Only refresh token hashes are stored in PostgreSQL.
* Refresh creates a new refresh token and revokes the old token.
* Reuse of a revoked refresh token revokes the full session family for that user.

## Cookie Model

The server sets `__Secure-capitalflow_refresh` for browser refresh-token rotation.

Cookie attributes:

* `Secure`
* `HttpOnly`
* `SameSite=Strict`
* `Path=/auth`

Refresh and logout use this cookie only. Auth JSON responses do not include refresh tokens.

## CSRF Model

API mutations use `Authorization: Bearer <access_token>` and do not rely on ambient cookie authentication.

Refresh/logout use the secure refresh cookie. Because that cookie is scoped to `/auth` and uses `SameSite=Strict`, cross-site browser submission risk is reduced.

Production also checks `Host`, `Origin`, and `Referer` for auth-sensitive endpoints against `PUBLIC_ORIGIN`.
Wildcard CORS origins are not allowed with credentials.

## Password Policy

Passwords must pass:

* minimum length: 12 characters
* `zxcvbn` score: at least 3
* user email, local-part, and domain are passed as user inputs to `zxcvbn`

## Lockout

Failed login attempts are tracked per user.

After 5 failed attempts, login is locked progressively:

* 5 minutes
* 15 minutes
* 1 hour
* 6 hours
* 24 hours

Successful login clears failed attempt state.

## Session Management

Users can list refresh sessions and revoke a specific session. Password change revokes all active refresh sessions.

## Passkeys

Passkey login uses WebAuthn with server-side one-use challenges.

* Registration requires an active access-token session.
* Adding any passkey requires password confirmation until recent session metadata is available.
* Login creates the same refresh session type as password login.
* `WEBAUTHN_RP_ID` and `WEBAUTHN_ORIGINS` must match the browser origin served by the reverse proxy.
* Local development can use `WEBAUTHN_RP_ID=localhost` with `http://localhost:5173` origins.
* If the frontend is opened through `127.0.0.1`, set a matching `WEBAUTHN_RP_ID` and `WEBAUTHN_ORIGINS`; WebAuthn RP IDs must match the browser host.
* Production should set `PUBLIC_ORIGIN=https://your-domain`, `WEBAUTHN_RP_ID=your-domain`, and `WEBAUTHN_ORIGINS=https://your-domain`.
* Production passkey origins must be HTTPS.
* Public passkey login options use a dedicated rate limit through `PASSKEY_OPTIONS_RATE_LIMIT_REQUESTS` and `PASSKEY_OPTIONS_RATE_LIMIT_WINDOW`.
* Expired and used WebAuthn challenges are cleaned up opportunistically by the application.

## Audit Trail

All auth-sensitive flows write events to `auth_audit_events`, including setup, login, refresh, logout, password changes, passkey ceremonies, session listing, session revocation, and refresh token reuse detection.

## Observability

Auth events are counted in `capitalflow_auth_events_total` and exposed through `GET /metrics`.

See [Auth Observability](Auth-Observability).

## Self-host Baseline

For reverse proxy and CSRF details, see:

* `docs/security/reverse-proxy.md`
* `docs/security/csrf.md`
