# CSRF Model

CapitalFlow does not use refresh cookies as API authentication for financial endpoints.

## API Authentication

Protected API routes under `/api/v1/*` use:

```http
Authorization: Bearer <access-token>
```

Browsers do not attach this header automatically, so normal cross-site form posts cannot authenticate API mutations.

## Refresh Cookie

The refresh token is stored in `__Secure-capitalflow_refresh` with:

* `HttpOnly`
* `Secure` when configured by `COOKIE_SECURE`
* `SameSite=Strict` by default
* `Path=/auth`

The `/auth/refresh` and `/auth/logout` endpoints use this cookie. The cookie is scoped to `/auth`, so it is not sent to `/api/*`.

## CORS

Credentialed CORS is allowed only for configured origins. Wildcard origins are not allowed when credentials are enabled.

Production should set:

```env
CORS_ALLOWED_ORIGINS=https://capitalflow.home.arpa
PUBLIC_ORIGIN=https://capitalflow.home.arpa
COOKIE_SAMESITE=Strict
```

Preflight requests do not require authentication.

## Origin Checks

In production, auth-sensitive endpoints compare:

* `Host` with the host from `PUBLIC_ORIGIN`;
* `Origin` and `Referer`, when present, with the full `PUBLIC_ORIGIN`.

This gives the password and refresh flows an explicit same-origin boundary before passkeys/WebAuthn are added.
