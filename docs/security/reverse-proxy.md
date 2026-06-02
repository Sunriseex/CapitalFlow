# Reverse Proxy Security

CapitalFlow can run behind Nginx, Traefik, or another trusted reverse proxy. In production, configure the public origin and only trust forwarding headers from known proxy addresses.

## Required Production Env

```env
APP_ENV=production
PUBLIC_ORIGIN=https://capitalflow.home.arpa
COOKIE_SECURE=true
COOKIE_SAMESITE=Strict
ALLOW_DIRECT_IP_LOGIN=false
TRUSTED_PROXIES=127.0.0.1/32,172.16.0.0/12
CORS_ALLOWED_ORIGINS=https://capitalflow.home.arpa
JWT_SECRET=<at-least-32-random-bytes>
```

`PUBLIC_ORIGIN` must be a full origin with scheme and host, without a path:

```text
valid:   https://capitalflow.home.arpa
invalid: capitalflow.home.arpa
invalid: https://capitalflow.home.arpa/app
```

## Host Policy

In production, auth-sensitive endpoints only accept requests whose `Host` matches the host from `PUBLIC_ORIGIN`.

Protected paths:

* `/auth/setup`
* `/auth/login`
* `/auth/refresh`
* `/auth/logout`
* `/api/v1/auth/password`
* `/api/v1/auth/sessions`

`/health` is not blocked by the host policy, so load balancers can probe it.

Direct IP login is blocked in production when `ALLOW_DIRECT_IP_LOGIN=false`. Private DNS names such as `capitalflow.home.arpa` are allowed when they are the configured public origin.

## Forwarded Client IP

CapitalFlow uses `X-Forwarded-For` and `X-Real-IP` only when the direct `RemoteAddr` belongs to `TRUSTED_PROXIES`.

Resolution order for trusted proxies:

1. First valid IP in `X-Forwarded-For`.
2. Valid `X-Real-IP`.
3. Direct `RemoteAddr`.

When the request does not come from a trusted proxy, forwarded headers are ignored. This prevents clients from rotating spoofed forwarded IPs to bypass rate limits.

## Nginx Example

```nginx
server {
    listen 443 ssl http2;
    server_name capitalflow.home.arpa;

    location / {
        proxy_pass http://127.0.0.1:18080;
        proxy_set_header Host $http_host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## Traefik Notes

Use `PUBLIC_ORIGIN` matching the external HTTPS URL and set `TRUSTED_PROXIES` to the Docker or LAN ranges that can actually connect to the backend.
