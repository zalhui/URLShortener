# URLShortener

URLShortener is a small production-oriented monorepo that combines three parts into one working system:

- a Go authentication service
- a Go URL shortener service
- a minimal web frontend served through `nginx`

The application is designed around a simple rule set:

- any user can open a short URL
- only authenticated users can create short URLs
- only the user who created a short URL can delete it

The repository is set up for local full-stack development with Docker Compose and for service-level development with plain Go commands.

## Architecture

The project uses a single-origin runtime model:

- `frontend` serves the web UI under `/app/`
- `frontend` proxies `/auth/*` to the auth service
- `frontend` proxies `/api/*` and `/{shortID}` to the shortener service

This design avoids browser CORS complexity and keeps cookie handling straightforward.

### Services

#### `services/auth`

Authentication service responsible for:

- user registration
- login
- access token issuance
- refresh token rotation
- logout
- current user lookup

Technical characteristics:

- PostgreSQL persistence
- bcrypt password hashing
- JWT access tokens
- hashed refresh tokens stored in the database
- HttpOnly refresh cookie

#### `services/shortener`

Shortener service responsible for:

- creating short URLs
- resolving short URLs
- listing URLs that belong to the current user
- deleting URLs owned by the current user

Technical characteristics:

- PostgreSQL or file-based storage
- JWT verification for protected endpoints
- public redirects without authentication
- user-scoped URL deduplication

#### `web/frontend`

Static frontend and reverse proxy responsible for:

- sign in and registration UI
- session restoration through refresh cookie
- short URL management dashboard
- exposing the system on one public origin

## Repository Layout

```text
.
├── docker-compose.yml
├── services
│   ├── auth
│   └── shortener
└── web
    └── frontend
```

Important paths:

- `services/auth/cmd/auth` - auth service entrypoint
- `services/shortener/cmd/shortener` - shortener service entrypoint
- `web/frontend` - static frontend assets and `nginx` config

## User Flow

The runtime flow is:

1. A user opens `http://localhost:3000/app/`.
2. The frontend calls `/auth/register` or `/auth/login`.
3. The auth service returns:
   - an access token in the JSON response
   - a refresh token in an HttpOnly cookie
4. The frontend keeps the access token in memory.
5. The frontend calls protected shortener endpoints with `Authorization: Bearer <token>`.
6. If the access token expires, the frontend calls `/auth/refresh` and retries the request.
7. Public short URLs continue to work directly through `GET /{shortID}` without authentication.

## Security Model

The current implementation includes the following baseline protections:

- passwords are hashed with bcrypt
- access tokens are signed JWTs
- refresh tokens are random, rotated, and stored only as hashes
- refresh cookies are `HttpOnly`
- protected shortener operations require a valid access token
- deletion is enforced by ownership checks on the server side
- short URL creation is scoped to the authenticated user

Current defaults:

- access token TTL: `15m`
- refresh token TTL: `7d`
- refresh cookie path: `/auth`
- refresh cookie mode: `SameSite=Strict`

This is a solid development baseline, but it is not a full enterprise auth platform yet. Features such as email verification, password reset, rate limiting, account lockout, and asymmetric JWT signing are not implemented.

## API Overview

### Auth API

Base service port in Docker Compose: `localhost:8081`

Endpoints:

- `GET /healthz`
- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/refresh`
- `POST /auth/logout`
- `GET /auth/me`

Example register request:

```json
{
  "email": "user@example.com",
  "password": "VerySecurePassword123"
}
```

Example auth response:

```json
{
  "access_token": "<jwt>",
  "token_type": "Bearer",
  "expires_in": 900,
  "user": {
    "id": "user-id",
    "email": "user@example.com",
    "created_at": "2026-03-12T12:00:00Z"
  }
}
```

### Shortener API

Base service port in Docker Compose: `localhost:8080`

Public endpoints:

- `GET /ping`
- `GET /{shortID}`

Protected endpoints:

- `POST /`
- `POST /api/shorten`
- `GET /api/urls`
- `DELETE /api/urls/{shortID}`

Example JSON shorten request:

```json
{
  "url": "https://example.com"
}
```

Example shorten response:

```json
{
  "result": "http://localhost:3000/abc123"
}
```

## Storage Model

### Auth database

Main tables:

- `auth_users`
- `auth_sessions`
- `schema_migrations`

`auth_sessions` stores:

- hashed refresh token
- user agent
- IP address
- expiry
- revocation timestamp
- replacement token hash for rotation

### Shortener database

Main tables:

- `short_urls`
- `schema_migrations`

`short_urls` stores:

- `uuid`
- `user_id`
- `original_url`
- `short_url`
- `created_at`

The schema enforces uniqueness on `(user_id, original_url)` so that the same user gets deterministic deduplication for the same original URL, while different users may shorten the same URL independently.

## Running the Full Stack

### Prerequisites

- Docker
- Docker Compose

### Start everything

```bash
docker compose up --build
```

Open the application:

```text
http://localhost:3000/app/
```

Published ports:

- `3000` - frontend and public entrypoint
- `8081` - auth service
- `8080` - shortener service
- `5434` - auth PostgreSQL
- `5433` - shortener PostgreSQL

### Rebuild only the frontend

If only frontend files changed:

```bash
docker compose up -d --build --no-deps frontend
```

If your local Docker daemon refuses to recreate the frontend container, update the running one directly:

```bash
docker cp web/frontend/index.html frontend-service:/usr/share/nginx/html/app/index.html
docker cp web/frontend/styles.css frontend-service:/usr/share/nginx/html/app/styles.css
docker cp web/frontend/app.js frontend-service:/usr/share/nginx/html/app/app.js
```

## Local Development Without Full Docker Runtime

You can run the databases in Docker and the Go services directly from the host.

### Auth service

```bash
export AUTH_ACCESS_TOKEN_SECRET='change-me-to-a-long-random-secret-at-least-32-chars'
export AUTH_DATABASE_DSN='postgres://auth:auth@localhost:5434/auth?sslmode=disable'
go run ./services/auth/cmd/auth
```

### Shortener service

```bash
export AUTH_ACCESS_TOKEN_SECRET='change-me-to-a-long-random-secret-at-least-32-chars'
export DATABASE_DSN='postgres://shortener:shortener@localhost:5433/shortener?sslmode=disable'
go run ./services/shortener/cmd/shortener
```

In that mode the services are still reachable on:

- `http://localhost:8081`
- `http://localhost:8080`

## Configuration

### Auth service environment variables

Primary variables:

- `AUTH_SERVER_ADDRESS`
- `AUTH_ACCESS_TOKEN_SECRET`
- `AUTH_ACCESS_TOKEN_TTL`
- `AUTH_REFRESH_TOKEN_TTL`
- `AUTH_ISSUER`
- `AUTH_COOKIE_NAME`
- `AUTH_COOKIE_DOMAIN`
- `AUTH_COOKIE_SECURE`
- `AUTH_DATABASE_DSN`

Alternative DB variables:

- `AUTH_DB_HOST`
- `AUTH_DB_PORT`
- `AUTH_DB_USER`
- `AUTH_DB_PASSWORD`
- `AUTH_DB_NAME`
- `AUTH_DB_SSL_MODE`

### Shortener service environment variables

Primary variables:

- `SERVER_ADDRESS`
- `BASE_URL`
- `AUTH_ISSUER`
- `AUTH_ACCESS_TOKEN_SECRET`
- `FILE_STORAGE_PATH`
- `DATABASE_DSN`

Alternative DB variables:

- `DB_HOST`
- `DB_PORT`
- `DB_USER`
- `DB_PASSWORD`
- `DB_NAME`
- `DB_SSL_MODE`

## Testing

Run tests from the repository root:

```bash
GOCACHE=$(pwd)/.cache/go-build GOMODCACHE=$(pwd)/.cache/go-mod go test ./services/auth/...
GOCACHE=$(pwd)/.cache/go-build GOMODCACHE=$(pwd)/.cache/go-mod go test ./services/shortener/...
```

## Manual Verification Checklist

After the stack is running:

1. Open `http://localhost:3000/app/`.
2. Register a user.
3. Create a short URL.
4. Open the generated short URL in a new tab.
5. Refresh the page and verify the session is restored.
6. Delete the created link.
7. Verify the deleted short URL returns `404`.
8. Log out and confirm protected actions are no longer available.

## Development Notes

- The frontend is intentionally minimal and dependency-free.
- The backend services are separated by responsibility and database ownership.
- Database migrations are applied automatically by each Go service on startup.
- The repository is structured to grow into a broader monorepo with additional services if needed.

## Current Limitations

- no email verification
- no password reset flow
- no rate limiting or brute-force protection
- no asymmetric JWT signing or JWKS endpoint
- no frontend build pipeline beyond static assets
- no integration test suite against live containers

## Summary

This repository is a complete local full-stack example of:

- Go microservice-style backend separation
- JWT-based authentication
- ownership-aware URL management
- public short link redirection
- a single-origin frontend with reverse proxying

It is intentionally small, but it already models the core boundaries and operational concerns of a real multi-service web application.
