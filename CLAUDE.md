# CLAUDE.md

## Purpose

This file tells Claude Code, GitHub Copilot, and any other AI tool **exactly how to work in this repository so changes look like they were written by a human on the team**.

If you are about to add a feature, a new endpoint, or refactor logic here, read this file top-to-bottom once and then treat it as the source of truth for architecture and conventions.

---

## 1. System Overview

**tross-scraper** is a LinkedIn Profile API: it accepts a LinkedIn profile URL and returns the profile as structured JSON.

A single Go 1.24 service, API only — there is no frontend and no browser
anywhere in the stack. LinkedIn is reached by direct HTTP calls to its private
JSON API; see §9.

- HTTP API built with **Gin**.
- **Redis** for caching and the daily scrape budget. There is no relational
  database — the service holds no persistent state (see §8).
- Observability via **OpenTelemetry** traces, **Prometheus** metrics and **zerolog** structured logs.
- Module path: `github.com/Tanmay-Tripathi/tross-scraper`.

There are **no private or internal dependencies**. Everything the service needs is either in `pkg/` or a public module.

---

## 2. How to Run and Format the Service

```bash
cp config/local.example.yml config/local.yml   # first run only
make up            # start Redis in Docker
make run           # run the API server on :4201
make run-live      # run with live reload (requires fswatch)
make build         # build the binary
make test          # go test ./...
make fmt           # go fmt ./... + go mod tidy
make vet           # go vet ./...
make lint          # fmt + vet

go run ./cmd/spike <profile-url>...   # probe LinkedIn, save fixtures, report coverage
make stack         # docker compose: API + Redis
```

**After making changes, always run `make lint` before considering the work complete.**

---

## 3. Architecture and Layering

### Startup and wiring

- `cmd/server/main.go` — entry point: parses flags, loads config, builds the logger, hands off to the app.
- `cmd/app/app.go` — the composition root. Builds every dependency in order, starts the HTTP server, and shuts everything down gracefully on SIGINT/SIGTERM.
- `cmd/app/routes.go` and `cmd/app/routes_<feature>.go` — HTTP routes, grouped into public / private tiers.
- `cmd/app/middlewares/` — HTTP middlewares.

### Core layers under `internal/`

| Directory               | Responsibility                                                            |
|-------------------------|---------------------------------------------------------------------------|
| `internal/controllers`  | HTTP handlers: parse, validate, map, call a service, write the response.  |
| `internal/services`     | Business logic and orchestration. Calls repositories and clients.         |
| `internal/clients`      | Outbound integrations, behind interfaces. Currently the LinkedIn client.  |
| `internal/repositories` | All cache access. Postgres is wired but unused — see §8.                 |
| `internal/models`       | Domain models. The only types services and repositories exchange.         |
| `internal/requests`     | Inbound request DTOs.                                                     |
| `internal/response`     | Outbound response DTOs.                                                   |
| `internal/exceptions`   | `ApplicationError` and the error-code catalogue.                          |
| `internal/config`       | Config struct, loading, `${VAR}` expansion, defaults and validation.      |

Middlewares live in `cmd/app/middlewares/`: `Cors` is applied globally in `newRouter`; `Idempotency` is applied per-route on expensive or mutating endpoints.

### Shared packages under `pkg/`

| Directory          | Responsibility                                                          |
|--------------------|-------------------------------------------------------------------------|
| `pkg/db`           | The Redis cache, and an unwired Postgres store kept as a seam (§8).     |
| `pkg/log`          | Context-aware structured logger over zerolog.                           |
| `pkg/network`      | The shared outbound HTTP caller, with an optional cookie jar.           |
| `pkg/linkedin/voyager` | LinkedIn's private API: client, URN resolver, payload structs, assembler. |
| `pkg/telemetry`    | OTel tracing, Prometheus metrics, request/correlation IDs.              |
| `pkg/mapper`       | DTO ↔ model conversion.                                                 |
| `pkg/validation`   | Request payload validation and normalisation helpers.                   |
| `pkg/utils`        | Response envelope helpers and other cross-cutting utilities.            |
| `pkg/global`       | Environments, header names, context keys.                               |

### Direction of dependencies — always

```
route → controller → service → repository / client → Redis / LinkedIn
```

Never call a repository or a client directly from a route or a middleware. Any change that violates this is a bug.

---

## 4. HTTP Request–Response Flow (Golden Path)

1. **Route** (`cmd/app/routes_<feature>.go`)
   - Define the Gin route and method. It calls **one controller method** and nothing else.
   - Apply auth/permission/idempotency middleware here, not inside the handler.

2. **Controller** (`internal/controllers`)
   - Bind the request and validate it with `pkg/validation`.
   - Map request DTOs to models with `pkg/mapper`.
   - Call the service.
   - Map the returned model to a response DTO with `pkg/mapper`.
   - Write the response with `utils.SendApiResponseV2` — **never** `ctx.JSON` directly.

3. **Service** (`internal/services`)
   - Implement the use case. Call repositories.
   - Work with domain models and `*exceptions.ApplicationError` only. **No Gin types, no HTTP status codes, no raw JSON.**

4. **Repository** (`internal/repositories`)
   - Talk to the cache. Take and return domain models.
   - Use context-aware calls; keep each method to one logical query.

---

## 5. Response Envelopes and Error Handling

### 5.1 Success envelope

Success constants live in `internal/exceptions`:

- `ApiSuccessCode = "00000"`
- `ApiSuccessMessage = "success"`

**Preferred shape for new endpoints** — `utils.SendApiResponseV2`:

```go
utils.SendApiResponseV2(ctx, result, pagination, nil)
```

Emits `{ code, message, result, pagination }` (`utils.ApiResponseV2[T]`). Build `pagination` with `utils.NewPagination(page, pageSize, total)`, or pass `nil`.

`SendApiResponseV2` is the only response writer. Calling `ctx.JSON` directly bypasses the envelope and the error mapping.

### 5.2 ApplicationError

`internal/exceptions.ApplicationError` carries:

- `ErrorCode` — short, stable string enum. **Never rename a released code.**
- `ErrorMessage` — the user-facing message.
- `HttpCode` — the HTTP status.

Validation and service layers return `*exceptions.ApplicationError`. Controllers **never** hand-craft error JSON; they pass the error to the response helper:

```go
utils.SendApiResponseV2(ctx, result, nil, appErr)
```

When `appErr != nil` the helper writes `{ code, message }` at `appErr.HttpCode`, normalising a blank code, message or status first. When `appErr == nil` it writes the 200 success envelope.

### 5.3 Adding new error codes

- Add them in `internal/exceptions/errors_<feature>.go` and register them from that file's `init()` via `register(...)`.
- `register` panics on a duplicate code, so collisions surface at startup rather than in a production response.
- Reuse an existing code before inventing one. Pick a short, descriptive code and the right status (400 validation, 401 auth, 404 missing, 429 rate limit, 5xx internal).
- Use `exceptions.Wrap(logger.Errorf, code, format, args...)` to log the internal cause while returning only the catalogued message to the client.

---

## 6. Validation and Mapping

### 6.1 Request validation (`pkg/validation`)

- One validation function per incoming request type.
- Do all normalisation here: `strings.TrimSpace`, lowercase enums, and write the normalised value back.
- Parse identifiers here and **return** the typed value alongside the error: `(typedValue, *exceptions.ApplicationError)`.
- For simple checks, return just `*exceptions.ApplicationError`.
- **Never** touch Gin types or HTTP status codes in a validator.

Controllers call these helpers; they must not inline validation rules.

### 6.2 Mapping (`pkg/mapper`)

- Payload → model and model → response both live here, one file per domain (`health_mapper.go`, …).
- Services depend on models, never on request DTOs.
- Mappers are the boundary that stops a new database column from silently leaking into the API.

When you add an entity, add both directions.

---

## 7. Logging Rules

Use `pkg/log` everywhere.

### 7.1 Context-aware logging

- Derive a logger early: `logger := c.Access.Logger.With(ctx)`.
- `ctx` must be a `context.Context`, **never** a `*gin.Context`. The logger warns when you get this wrong.
- `telemetry.TraceIDMiddleware` puts the request and correlation IDs on the context; `With(ctx)` picks them up automatically.

### 7.2 Levels

- `Debug`/`Debugf` — detailed diagnostics around complex branching. Never in a hot path.
- `Info`/`Infof` — lifecycle events, major state changes, successful completion.
- `Warn`/`Warnf` — degraded but recoverable (e.g. Redis unavailable, falling back to the uncached path).
- `Error`/`Errorf` — an operation could not complete, or data is wrong.

### 7.3 Placement and content

- Log an error **once**, at the layer that detects it. Do not re-log the same failure as it bubbles up unless you are adding context.
- Include the identifiers that make debugging possible: entity IDs, queue name, message ID, cache key.
- **Never log secrets, credentials, cookies or session tokens.** Strip query strings from any URL you log — they routinely carry tokens.

---

## 8. Repositories and Storage

**Redis is the only live datastore.** The service holds no relational state: a
profile is fetched, mapped, cached and returned. `RepositoryAccess.Db` is a nil
`*db.Store` and `PingDatabase` returns `ErrDatabaseNotConfigured`, which
readiness reports as `disabled` rather than `down`.

Do not add a Postgres dependency without a real reason to persist something. If
you do, wire the store in `cmd/app/app.go` — the seam is a single `var store
*db.Store` there, and every layer below already accepts it.

All cache access goes through `internal/repositories`. The rules below apply to
Postgres if it is ever wired back in.

- Postgres via **GORM**: `access.Db.MasterDB` for writes, `access.Db.SlaveDB` for reads. **Nil-check it first.**
- Always pass `context.Context` so timeouts and cancellation propagate.
- Keep methods narrow — one logical query, or a small coherent sequence.
- Distinguish "not found" from a real error: return `gorm.ErrRecordNotFound` for a missing row and let the service map it to the right `ApplicationError`.
- Use `Store.Begin`/`Commit`/`Rollback` for multi-step writes that must succeed or fail together.
- Watch for N+1 patterns and Cartesian products on multi-relation reads. Batch or preload.

Migrations live in `migrations/postgres/` and are run by `db.NewStore`, which nothing currently calls. `make migrate-new` still creates a pair, for when a store is wired.

---

## 9. External Clients and Configuration

External integrations live in `internal/clients`. Today that is `client_linkedin.go`.

- One file per integration, behind a `Client<Name>Methods` interface, so services depend on an abstraction and tests can fake it.
- Registered on the `Clients` aggregate in `internal/clients/main.go`, wired from `cmd/app/app.go`, reaching services via `ServiceAccess.Clients`.
- Build outbound HTTP on `pkg/network.NetworkOpsMethods`, not a bare `http.Client`, so timeouts, trace-header propagation and URL-redacting logs stay consistent.
- A client whose dependency is unreachable is logged and left `nil` — a degraded downstream must not take the API down. **Callers check for `nil`.**
- Clients return `*exceptions.ApplicationError`, so no layer above reads an upstream HTTP status.
- Services call clients; controllers and repositories never do.

**LinkedIn specifics.** Everything about LinkedIn's wire format is quarantined in
`pkg/linkedin/voyager` — if LinkedIn changes its API, that package is the only
thing that moves. Section on/off rules are enforced in exactly one function,
`pkg/mapper.ToProfileResult`; do not re-implement them elsewhere.

Configuration rules:

- Everything environment-driven goes through `internal/config` and `config/*.yml`.
- Config files support `${VAR}` and `${VAR:-default}`. **This is how secrets stay out of the repo** — the committed file names the variable, the deployment supplies the value.
- Numeric and boolean fields in a committed config must always use the `${VAR:-default}` form; a bare unset `${VAR}` leaves an empty YAML value.
- List fields use `config.StringList`, which accepts both a YAML sequence and a comma-separated scalar — the scalar form is what makes a list settable from one environment variable.
- When adding a field: add it to the struct, give it a sensible default in `applyDefaults`, validate it in `validate` if it is required, and document it in `config/local.example.yml`.
- Keep existing keys backwards compatible.
- `config/local.yml` is gitignored. **Never commit it.**

---

## 10. Adding a New Feature (Checklist)

1. **Model** — `internal/models/<entity>.go`.
2. **Repository** — `internal/repositories/repo_<feature>.go`, registered in `main.go`.
3. **Service** — `internal/services/service_<feature>.go`, registered in `main.go`.
4. **DTOs** — `internal/requests/requests_<feature>.go` and `internal/response/response_<feature>.go`. Use `binding:"required"` tags.
5. **Errors** — `internal/exceptions/errors_<feature>.go`, registered from its `init()`.
6. **Validation and mapping** — `pkg/validation` and `pkg/mapper`.
7. **Controller** — `internal/controllers/controller_<feature>.go`, registered in `main.go`.
8. **Routes** — `cmd/app/routes_<feature>.go`, called from `addRoutes`.
10. **API examples** — add or update the Bruno requests under `api_collection/`.

### Naming conventions

| Layer      | File pattern              | Interface pattern            |
|------------|---------------------------|------------------------------|
| Controller | `controller_<feature>.go` | `Controller<Feature>Methods` |
| Service    | `service_<feature>.go`    | `Service<Feature>Methods`    |
| Repository | `repo_<feature>.go`       | `Repository<Feature>Methods` |
| Client     | `client_<feature>.go`     | `Client<Feature>Methods`     |
| Middleware | `middleware_<name>.go`    | `Middleware<Name>Methods`    |
| Errors     | `errors_<feature>.go`     | —                            |
| Mapper     | `<feature>_mapper.go`     | —                            |

Each layer follows the same DI shape:

- `types.go` — the `*Access` struct holding that layer's shared dependencies.
- `main.go` — the aggregate struct (`Controllers`, `Services`, …) and its `New*` constructor.
- Feature files — implement an interface and take the `*Access` struct in their constructor.

### Route groups (`cmd/app/routes.go`)

- `PublicApiV1` = `/public/v1` — unauthenticated.
- There is no authenticated tier. `/v1` was removed along with its one route —
  no auth middleware exists, so mounting anything there is just a second public
  path. Add the group back in the same commit that adds the middleware.
- `PrivateApiV1` = `/private/v1` — internal only, not routed from the internet.

---

## 11. Coding Style and General Principles

- **Single responsibility** — one function does one thing. Past ~30–40 lines, or once it mixes validation with DB access with mapping, split it.
- **Modular and reusable** — extract shared logic into `pkg/mapper`, `pkg/validation` or `pkg/utils` rather than duplicating across controllers and services.
- **Pass by value by default** — use pointers only to mutate in place, to express optional (nil vs value), or when a struct is measurably expensive to copy.
- **Parameter objects over long argument lists** — more than three inputs means a small options/request struct.
- **No business logic in routes** — Gin handlers stay thin.
- **Constructors for complex structs**; keep structs small and focused; prefer composition; do not export fields other packages should not mutate.
- **Error handling** — check every error, wrap with `fmt.Errorf("...: %w", err)` to preserve context, avoid `panic` for normal control flow, avoid double-logging.
- **Resource lifetime** — close resources with `defer`, propagate context cancellation into goroutines, give every goroutine a clear exit condition.
- **Concurrency** — pass context into any goroutine doing work; use bounded worker pools, never unbounded goroutine spawning.
- **Security** — validate every external input, use parameterised queries, hash password-like secrets with a real KDF, and keep credentials in the environment.
- **Performance** — profile before optimising; use pagination or streaming for anything that could return many rows; set timeouts on every external call.
- **Testing** — prefer table-driven tests; mock repositories in service tests; cover edge cases as well as the happy path.
- **Packages** — small and cohesive; no import cycles.
- **Context** — `context.Context` is the first parameter of any function doing I/O. Never store it in a struct or a global.
- **Dependencies** — check a new library for known vulnerabilities before adding it.

---

## 12. Performance and Safety

- Pass `context.Context` down to every DB, cache and HTTP call.
- Paginate or stream any endpoint that could return many rows.
- Batch to avoid N+1 queries.
- No global mutable state — inject through `cmd/app/app.go`.
- Set timeouts and limits on every external call.
- Scraping work is expensive and rate-limited upstream: cache aggressively in Redis, and put mutating endpoints behind `middlewares.Idempotency` so a client retry does not trigger a second fetch.

---

## 13. Evolving This Guide

When you introduce a new pattern — a cross-cutting middleware, a shared client helper, a different response style — update this file with:

- where the new pattern lives,
- when to use it versus the existing pattern,
- an example specific to **this** repo, not generic Go advice.

This file should always describe how the project works **today**.
