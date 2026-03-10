# Auth Service

Backend authentication service for the monorepo.

Capabilities:
- user registration and login
- bcrypt password hashing
- JWT access tokens
- rotating refresh tokens stored as SHA-256 hashes
- PostgreSQL-backed users and sessions

Run from the repository root:

```bash
go run ./services/auth/cmd/auth
```

Run tests from the repository root:

```bash
go test ./services/auth/...
```

Required environment:

- `AUTH_ACCESS_TOKEN_SECRET`
- `AUTH_DATABASE_DSN` or the `AUTH_DB_*` variables

Optional environment:

- `AUTH_SERVER_ADDRESS`
- `AUTH_ACCESS_TOKEN_TTL`
- `AUTH_REFRESH_TOKEN_TTL`
- `AUTH_ISSUER`
- `AUTH_COOKIE_NAME`
- `AUTH_COOKIE_DOMAIN`
- `AUTH_COOKIE_SECURE`
