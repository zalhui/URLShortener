# URLShortener Monorepo

Repository layout:

- `services/shortener` contains the current Go URL shortener service
- `services/auth` contains the authentication service
- `web/frontend` contains the static web application and reverse proxy
- `docker-compose.yml` contains the local full-stack environment

Commands from the repository root:

```bash
go test ./services/shortener/...
go run ./services/shortener/cmd/shortener
```

Full stack:

```bash
docker compose up --build
```

Then open:

```bash
http://localhost:3000/app/
```
