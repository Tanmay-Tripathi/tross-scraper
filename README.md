# tross-scraper

A **LinkedIn Profile API** — give it a LinkedIn profile URL, get the profile back as structured JSON.

Go (Gin) service backed by PostgreSQL and Redis, with a small React SPA in front of it. Deployable to any Docker host; a Render blueprint is included for one-command public HTTPS.

> **Status:** the platform is in place — layered service skeleton, config, migrations, caching, telemetry, health probes, Docker, CI and deployment. The scraping feature itself is not implemented yet.

---

## Table of contents

- [Architecture](#architecture)
- [Requirements](#requirements)
- [Local setup](#local-setup)
- [Configuration](#configuration)
- [API](#api)
- [Frontend](#frontend)
- [Deployment](#deployment)
- [Project layout](#project-layout)
- [Development workflow](#development-workflow)
- [Known limitations](#known-limitations)

---

## Architecture

Strict layering, enforced by convention and documented in [CLAUDE.md](CLAUDE.md):

```
route → controller → service → repository (+ clients) → Postgres / Redis / SQS / external APIs
```

- **Controllers** own HTTP: binding, validation, mapping, and the response envelope.
- **Services** own business logic. They never see a Gin type or an HTTP status code.
- **Repositories** own persistence. They take and return domain models.
- **Clients** own outbound integrations, behind interfaces so they can be faked in tests.

Cross-cutting infrastructure lives in `pkg/` — logging, database, cache, HTTP client, SQS/SNS, telemetry, validation, mapping and the response helpers. There are **no private dependencies**; everything is either in this repo or a public module.

**Stack:** Go 1.24 · Gin · GORM · PostgreSQL 16 · Redis 7 · golang-migrate · zerolog · OpenTelemetry · Prometheus · AWS SDK v2 (SQS/SNS) · Vite 7 · React 19 · TypeScript · Tailwind CSS 3 · shadcn/ui · Biome

---

## Requirements

- Go 1.24+
- Docker and Docker Compose (for Postgres and Redis)
- Node 22+ (frontend only)
- `fswatch` (optional, for `make run-live`)

---

## Local setup

```bash
git clone https://github.com/Tanmay-Tripathi/tross-scraper.git
cd tross-scraper

# 1. Configuration
cp config/local.example.yml config/local.yml

# 2. Dependencies
make up          # Postgres on :5432, Redis on :6379

# 3. Run
make run         # API on http://localhost:4201
```

Verify:

```bash
curl http://localhost:4201/public/v1/health
curl http://localhost:4201/private/v1/health/ready
```

### Everything in Docker

```bash
make stack       # API :4201, frontend :4204, Postgres, Redis
```

---

## Configuration

Config is a YAML file, selected with `-config` (default `./config/local.yml`).

| File                       | Purpose                                            | Committed |
|----------------------------|----------------------------------------------------|-----------|
| `config/local.example.yml` | Template for local development                     | yes       |
| `config/local.yml`         | Your local config                                  | **no**    |
| `config/production.yml`    | Deployed config — every secret is a `${VAR}` ref   | yes       |

Values support `${VAR}` and `${VAR:-default}`, expanded from the process environment at startup. **That is how credentials stay out of this repository**: the committed file names the variable, the deployment supplies the value.

Configuration is validated at startup — a missing database DSN or an unknown environment fails the process immediately rather than at the first request.

### Environment variables used by `config/production.yml`

| Variable            | Required | Default       | Purpose                                  |
|---------------------|----------|---------------|------------------------------------------|
| `DATABASE_URL`      | yes      | —             | Postgres connection string               |
| `REDIS_HOST`        | yes      | —             | Redis hostname                           |
| `REDIS_PORT`        | no       | `6379`        | Redis port                               |
| `REDIS_USERNAME`    | no       | empty         | Redis ACL user                           |
| `REDIS_PASSWORD`    | no       | empty         | Redis password                           |
| `REDIS_TLS_ENABLED` | no       | `false`       | TLS to Redis                             |
| `PORT`              | no       | `4201`        | HTTP listen port                         |
| `APP_ENV`           | no       | `prd`         | `local` / `stg` / `uat` / `prd`          |
| `LOG_LEVEL`         | no       | `info`        | `debug` / `info` / `warn` / `error`      |
| `BASE_URL`          | no       | empty         | Public base URL of the service           |
| `OTLP_EXPORTER_URL` | no       | empty         | OTLP/HTTP endpoint; empty disables traces|
| `CORS_ALLOWED_ORIGINS` | no    | empty         | Comma-separated browser origins allowed to call the API |
| `SQS_ENABLED`       | no       | `false`       | Enable AWS messaging                     |
| `AWS_REGION`        | no       | `us-east-1`   | AWS region for SQS/SNS                   |

---

## API

Base URL: `http://localhost:4201`

### Route groups

| Prefix        | Access          | Notes                                    |
|---------------|-----------------|------------------------------------------|
| `/public/v1`  | unauthenticated | Safe to expose to the internet           |
| `/v1`         | authenticated   | End-user endpoints                       |
| `/private/v1` | internal only   | Probes and internal calls; do not route publicly |

### Response envelope

Every endpoint returns the same shape.

**Success — HTTP 200**

```json
{
  "code": "00000",
  "message": "success",
  "result": { "...": "..." },
  "pagination": { "page": 1, "page_size": 20, "total_items": 42, "total_pages": 3 }
}
```

`pagination` is present only on list endpoints.

**Error — HTTP status from the error's mapping**

```json
{
  "code": "HLT01",
  "message": "one or more downstream dependencies are unavailable"
}
```

Error codes are short, stable strings catalogued in `internal/exceptions`. They are never renamed once released, so clients can safely switch on them.

| Code    | Status | Meaning                                     |
|---------|--------|---------------------------------------------|
| `00000` | 200    | Success                                     |
| `INV01` | 400    | Invalid params                              |
| `IR01`  | 400    | Invalid request                             |
| `EB01`  | 400    | Empty request body                          |
| `UA01`  | 401    | Unauthorized                                |
| `TMR01` | 429    | Too many requests                           |
| `SWW01` | 500    | Something went wrong                        |
| `SU01`  | 503    | Service temporarily unavailable             |
| `HLT01` | 503    | A downstream dependency is unavailable      |

### Endpoints

#### `GET /public/v1/health`

Liveness. Answers as long as the process is up; it does not touch Postgres or Redis, so it is cheap enough for an uptime monitor.

```bash
curl http://localhost:4201/public/v1/health
```

```json
{
  "code": "00000",
  "message": "success",
  "result": {
    "service": "tross-scraper",
    "version": "0.1.0(dev)",
    "environment": "local"
  }
}
```

#### `GET /private/v1/health/live`

Same as above, on the private group, for the container orchestrator's liveness probe.

#### `GET /private/v1/health/ready`

Readiness. Pings Postgres and Redis and returns **503** with code `HLT01` if either is down, so a broken instance is pulled out of rotation.

```json
{
  "code": "00000",
  "message": "success",
  "result": {
    "status": "healthy",
    "service": "tross-scraper",
    "version": "0.1.0(dev)",
    "environment": "local",
    "dependencies": {
      "database": { "status": "up" },
      "cache": { "status": "up" }
    }
  }
}
```

#### `GET /metrics`

Prometheus exposition: request counts, latency histogram and in-flight gauge, labelled by method, matched route template and status.

### CORS

Browser clients are rejected unless their origin is listed in `Cors.allowed_origins` (env: `CORS_ALLOWED_ORIGINS`, comma-separated). The middleware echoes the caller's origin rather than `*`, so credentialed requests work; an unlisted origin gets no CORS headers, and its preflight gets a 403. Leaving the list empty disables CORS entirely, which is the right default for a service with no browser client.

### Idempotency

Expensive or mutating endpoints can be wrapped in `middlewares.Idempotency`. Those routes require an `x-idempotency-key` header; a repeat of the same key replays the stored successful response for six hours instead of re-running the handler. Non-2xx responses are not cached, so a failure stays retryable with the same key.

### Bruno collection

Runnable examples live in [`api_collection/`](api_collection/) — open the folder in [Bruno](https://www.usebruno.com/) and select the `local` environment.

---

## Frontend

A standalone Vite + React SPA in [`frontend/`](frontend/) that talks to the API. It currently renders a landing page and a live API status panel.

```bash
cp frontend/.env.example frontend/.env
make fe-install
make fe-dev              # http://localhost:4204
```

`VITE_API_HOST` points the SPA at the API. Vite **inlines env vars at build time**, so changing it requires a rebuild, not just a restart.

---

## Deployment

The service is a single static binary in a distroless-style Alpine image running as a non-root user. Migrations run automatically at startup.

### Render (blueprint included)

[`render.yaml`](render.yaml) provisions the API, a Postgres instance, a Redis (Key Value) instance and the frontend static site — all on HTTPS.

1. Push this repository to GitHub.
2. In Render: **New → Blueprint**, and select the repository.
3. After the first deploy, cross-reference the two service URLs:
   - set the frontend's `VITE_API_HOST` to the API's URL and **rebuild** (Vite inlines it at build time, so a restart is not enough);
   - set the API's `CORS_ALLOWED_ORIGINS` to the frontend's URL.

The API's health check path is `/public/v1/health`.

### Any Docker host

```bash
docker build -t tross-scraper .
docker run -p 4201:4201 \
  -e DATABASE_URL="postgres://user:pass@host:5432/db?sslmode=require" \
  -e REDIS_HOST=redis-host \
  -e REDIS_PORT=6379 \
  -e REDIS_PASSWORD=secret \
  -e REDIS_TLS_ENABLED=true \
  -e CORS_ALLOWED_ORIGINS=https://your-frontend.example.com \
  tross-scraper
```

---

## Project layout

```
.
├── cmd/
│   ├── server/            # entry point
│   └── app/               # composition root, routes, middlewares
├── internal/
│   ├── clients/           # outbound integrations (SQS/SNS)
│   ├── config/            # config struct, loading, validation
│   ├── controllers/       # HTTP handlers
│   ├── exceptions/        # ApplicationError + error catalogue
│   ├── models/            # domain models
│   ├── repositories/      # Postgres and cache access
│   ├── response/          # outbound DTOs
│   └── services/          # business logic
├── pkg/
│   ├── db/                # Postgres store + Redis cache
│   ├── global/            # environments, headers, context keys
│   ├── log/               # structured logger
│   ├── mapper/            # DTO ↔ model conversion
│   ├── network/           # shared outbound HTTP client
│   ├── queue/awssqs/      # SQS/SNS wrapper + worker receiver
│   ├── telemetry/         # tracing, metrics, request IDs
│   ├── utils/             # response envelope helpers
│   └── validation/        # request validation helpers
├── migrations/postgres/   # golang-migrate files, applied at startup
├── config/                # YAML configuration
├── api_collection/        # Bruno API collection
├── frontend/              # Vite + React SPA
├── docker-compose.yml     # local stack
├── render.yaml            # Render blueprint
└── CLAUDE.md              # architecture and conventions guide
```

---

## Development workflow

```bash
make lint          # go fmt + go mod tidy + go vet
make test          # go test ./...
make fe-lint       # biome check --write
```

CI ([`.github/workflows/ci.yml`](.github/workflows/ci.yml)) runs gofmt, vet, build and tests for the backend; Biome and the production build for the frontend; and builds both Docker images.

Adding a feature? Follow the checklist in [CLAUDE.md §10](CLAUDE.md).

---

## Known limitations

- **The scraping feature is not implemented yet.** The repository currently ships the platform: layering, config, migrations, cache, telemetry, health probes, containers, CI and deployment. The LinkedIn client, its models and its endpoints come next.
- **Automatic migrations.** Migrations run on every boot from the container's working directory. This is convenient for a single instance; a multi-replica rollout should move them to a release step so two replicas cannot race.
- **Free-tier deployment.** Render's free plan sleeps idle services, so the first request after a quiet period is slow, and the free Postgres and Key Value instances have small quotas.
- **SQS/SNS is disabled by default.** It needs AWS credentials and creates topics and queues on demand. Enable it with `SQS_ENABLED=true`, or point `SQS.endpoint` at LocalStack for local work.
- **Tracing is off unless configured.** Set `OTLP_EXPORTER_URL` to a collector to turn it on. Prometheus metrics are always available at `/metrics`.
- **No authentication yet.** The `/v1` protected group exists but has no auth middleware behind it. Anything sensitive must not be mounted there until one is added.
- **No rate limiting yet.** `TMR01` is catalogued but nothing enforces a limit; add one before exposing scraping endpoints publicly.
- **`/metrics` is unauthenticated.** Fine behind a private network; put it behind an ingress rule before exposing the service publicly.
- **Read replicas are configured but not required.** `slave_database_dsn` falls back to the master, so reads and writes share one instance unless you set it.

---

## License

MIT — see [LICENSE](LICENSE).
