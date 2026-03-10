# URLShortener Monorepo

Repository layout:

- `services/shortener` contains the current Go URL shortener service
- `services/auth` is reserved for the future authorization service
- `web/frontend` is reserved for the future web application
- `docker-compose.yml` contains local infrastructure for development

Commands from the repository root:

```bash
go test ./services/shortener/...
go run ./services/shortener/cmd/shortener
```
