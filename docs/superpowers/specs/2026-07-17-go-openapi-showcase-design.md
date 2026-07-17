# Design: Full OpenAPI-documented Go API to showcase Go on Laravel Cloud

**Date:** 2026-07-17
**Status:** Approved (design), pending spec review

## Goal

Turn the existing minimal stdlib HTTP API into a polished, self-documenting REST
API that showcases what makes Go compelling on Laravel Cloud: a tiny static
binary, near-zero third-party dependencies, fast cold starts, and interactive
API docs served straight from the binary.

## Guiding principles

- **Stdlib-first.** Use Go 1.22+ `net/http` routing and the standard library.
  Avoid heavy web frameworks — "no framework needed" is part of the pitch.
- **Single self-contained binary.** The OpenAPI spec and Swagger UI assets are
  compiled into the binary via `go:embed`. No CDN, no external asset hosting,
  no runtime file dependencies. One binary deploys everywhere.
- **Bulletproof demo.** No external service (DB, cache) required to run or
  demonstrate the API. Architecture makes the upgrade path obvious.

## Architecture

Existing layout is kept and extended:

```
main.go                        # server bootstrap, graceful shutdown
internal/
  handlers/                    # HTTP handlers, one file per concern
    handlers.go                # Register(mux) — single source of routes
    users.go                   # users resource (list/get/create/update/delete)
    hello.go health.go random.go echo.go version.go
    docs.go                    # NEW: serves /openapi.yaml and /docs (Swagger UI)
  middleware/                  # recovery, logging, request-id, CORS (NEW)
  models/                      # domain types (User, validation)
  store/                       # NEW: Store interface + in-memory implementation
  respond/                     # JSON + structured error helpers
api/
  openapi.yaml                 # NEW: hand-written OpenAPI 3.1 spec (source of truth)
  swagger-ui/                  # NEW: vendored Swagger UI static assets (embedded)
```

### OpenAPI approach: spec-first, hand-written, embedded

- `api/openapi.yaml` is the source of truth, written by hand as OpenAPI 3.1.
- Embedded into the binary with `go:embed`.
- `GET /openapi.yaml` serves the raw spec (content-type `application/yaml`).
- `GET /docs` serves an embedded Swagger UI configured to load `/openapi.yaml`.
- A **contract test** asserts every route registered in `handlers.Register` has a
  matching `path + method` in the spec, and vice versa. This catches drift
  without the ceremony of code generation.

Rejected alternative: `oapi-codegen` (spec-first typed codegen). It gives
compile-time contract enforcement but restructures handlers and adds a build
step, working against the "clean, obvious, reliable demo" goal. The contract
test provides most of the safety at a fraction of the complexity.

### Persistence: in-memory behind a `Store` interface

```go
type Store interface {
    List(ctx context.Context, q string, limit, offset int) ([]models.User, int, error)
    Get(ctx context.Context, id int) (models.User, error)
    Create(ctx context.Context, u models.User) (models.User, error)
    Update(ctx context.Context, id int, u models.User) (models.User, error)
    Delete(ctx context.Context, id int) error
}
```

- `MemoryStore` implements it, seeded with the current sample users.
- Handlers depend on the `Store` interface, injected at construction — no
  package-level global map.
- `ErrNotFound` sentinel error lets handlers map store errors to HTTP 404.
- A documented seam (comment + README note) shows how a `PostgresStore` backed
  by Laravel Cloud managed Postgres would drop in. Not implemented.

## API surface

Utility/demo endpoints (kept as-is, documented in the spec):
`GET /`, `GET /hello`, `GET /health`, `GET /version`, `GET /time`,
`GET /random`, `GET /quote`, `POST /echo`, `GET /panic`, `GET /slow`.

Users resource (completed and hardened):

| Method | Path          | Notes                                                |
|--------|---------------|------------------------------------------------------|
| GET    | /users        | Pagination `?limit` (default 20, max 100), `?offset`; filter `?q=` over name/email. Returns `{users, count, limit, offset}`. |
| GET    | /users/{id}   | 404 with structured error if missing.                |
| POST   | /users        | Validates name + email; 201 with created user.       |
| PUT    | /users/{id}   | NEW. Full update; validates; 404 if missing.         |
| DELETE | /users/{id}   | 204 on success; 404 if missing.                      |

### Validation rules

- `name`: required, 1–100 chars after trimming whitespace.
- `email`: required, must parse via `net/mail.ParseAddress`.
- Validation failures return `400` with a structured error listing the problem.

## Error handling

Single structured error shape, used everywhere:

```json
{
  "error": "Not Found",
  "detail": "user not found",
  "status": 404,
  "request_id": "a1b2c3d4"
}
```

- `respond.Error` is extended to include the `request_id` (pulled from context).
- Panics are recovered by existing `Recovery` middleware and returned as a
  structured 500.
- The error shape is defined once as a `models.APIError` type and referenced in
  the OpenAPI spec.

## Middleware

Existing chain kept: `Recovery` → `Logging` → `RequestID`. Add:

- **CORS** middleware (permissive: `*` origin, common methods/headers) so
  browsers and the hosted Swagger UI can call the API during demos.

## Testing

- Table-driven tests for each users handler (happy path + validation + 404),
  driven against a fresh `MemoryStore` per test.
- Contract test: registered routes ⇔ OpenAPI spec paths/methods match.
- Existing `hello_test.go` style is followed. `make test` stays green.

## Deploy / docs

- README rewritten with:
  - What the project is and the Go-on-Laravel-Cloud pitch.
  - Local run (`make run`), test (`make test`), Docker build.
  - Laravel Cloud deploy walkthrough (Dockerfile-based; `PORT`/`ADDR` env).
  - `curl` examples for each endpoint and a link to `/docs`.
- Confirm the server honors the platform's port env var. Laravel Cloud / most
  PaaS inject `PORT`; `main.go` currently reads `ADDR`. Update bootstrap to
  prefer `PORT` (as `:$PORT`) then fall back to `ADDR`, then `:9114`.

## Out of scope (YAGNI for this pass)

- Auth / API keys, rate limiting, metrics endpoint, a second resource.
- Real database. Interface seam only.
- Code generation from the spec.

## Success criteria

1. `make test` passes, including the spec-drift contract test.
2. `docker build` produces a working single-binary image; `curl` against a
   running container returns expected JSON for every endpoint.
3. `/docs` renders interactive Swagger UI that can execute requests against the
   live API, with all endpoints and schemas present.
4. README lets a reader deploy to Laravel Cloud and hit the API.
