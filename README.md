# go-laravelcloud

A small, **dependency-free** Go REST API that showcases running Go on **Laravel Cloud**.

- Standard library only — `go.mod` has zero `require` directives.
- Go 1.22+ `net/http` routing (method + path patterns), no framework.
- OpenAPI 3.1 spec and Swagger UI **embedded into the binary** (`go:embed`) — the whole thing ships as one static file.
- Graceful shutdown, structured JSON errors, request IDs, CORS, panic recovery.

## Endpoints

Interactive docs: **`/docs`** · Raw spec: **`/openapi.json`**

| Method | Path          | Description                       |
|--------|---------------|-----------------------------------|
| GET    | /users        | List users (`?q`, `?limit`, `?offset`) |
| GET    | /users/{id}   | Get a user                        |
| POST   | /users        | Create a user                     |
| PUT    | /users/{id}   | Update a user                     |
| DELETE | /users/{id}   | Delete a user                     |
| GET    | /hello, /health, /version, /time, /random, /quote | Utility/demo endpoints |
| POST   | /echo         | Echo the request                  |
| GET    | /panic, /slow | Middleware demos                  |

## Run locally

```bash
make run                 # go run .  (listens on :9114)
make test                # go test ./...
open http://localhost:9114/docs
```

Example:

```bash
curl -s localhost:9114/users | jq
curl -s -X POST localhost:9114/users -d '{"name":"Ada","email":"ada@example.com"}' | jq
```

## Deploy to Laravel Cloud

This app deploys as a Docker image and reads its listen port from `PORT`
(falling back to `ADDR`, then `:9114`).

1. Push this repo to GitHub and create a new application in Laravel Cloud,
   pointing at your repository/branch.
2. The included `Dockerfile` builds a distroless static binary — no runtime
   language configuration needed.
3. Laravel Cloud injects a `PORT` for the web process; the app binds to it
   automatically. Confirm the exposed port in the environment's network
   settings matches.
4. Deploy. Hit `/docs` on your assigned domain to see the live interactive API.

> Persistence is in-memory behind a `store.Store` interface. To back it with
> Laravel Cloud managed Postgres, add a `PostgresStore` implementing the same
> interface (using the injected `DB_*` env vars) and construct it in `main.go` —
> no handler changes required.

## Build the container

```bash
make docker
docker run --rm -e PORT=8080 -p 8080:8080 go-laravelcloud:dev
```
