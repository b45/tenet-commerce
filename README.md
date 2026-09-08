# Tenet Commerce

Multi-tenant retail backend reference implementation for point-of-sale, supply-chain compliance, and double-entry ledger workflows.

> **Current status (2026-09-08):** The Go API, Next.js online POS, tenant-scoped catalog/cart drafts, runtime readiness, production frontend container, and benchmark evidence are implemented. Offline transaction replay, AI auditing, Zakat calculation, and production orchestration remain planned. See [Implementation Status](docs/IMPLEMENTATION_STATUS.md) and [Architecture Evidence](docs/ARCHITECTURE_EVIDENCE.md) for the verified boundary.

## What is implemented

- Go API with JWT authentication, role/permission guards, structured request logging, and schema-per-tenant routing.
- POS catalog and category management, checkout, order history, void, daily summary, QRIS configuration, stock adjustment, and low-stock query.
- Supplier, purchase-order, and goods-receipt creation with configurable compliance-certificate checks.
- Double-entry journal generation, chart of accounts, manual journal entry, trial balance, and manager dashboard aggregation.
- Redis fast-path plus PostgreSQL durable HTTP idempotency records for POS mutations; checkout uses PostgreSQL row locks for inventory mutation.

The important caveats are visible: HTTP response persistence occurs after the business handler, offline paid replay is not enabled, and benchmark figures are isolated laptop measurements. The complete boundary is documented in [Architecture Evidence](docs/ARCHITECTURE_EVIDENCE.md).

## Planned, not currently implemented

- Phase 3: Offline transaction queue, replay, conflict recovery, and operational dashboards. The online POS is implemented; paid offline replay is not.
- Phase 4: AI anomaly analysis, persisted audit reports, Zakat Tijarah engine, production deployment, and end-to-end release orchestration.

The existing Next.js project and Python scheduler are scaffolds for those phases, not finished features.

## Architecture

```text
Client / API consumer
        |
        v
Gin API: JWT -> RBAC -> tenant resolver -> domain handlers
        |                         |
        |                         +-- trusted tenant schema from public registry
        v
PostgreSQL 16 (public registry + tenant schemas) <----> Redis 7 (POS idempotency cache)
        |
        +-- POS / supply chain / ledger / manager modules
```

The backend is a modular monolith written with Gin, pgx/v5, and go-redis. PostgreSQL is the transactional system of record. The tenant request context applies a trusted schema search path to a dedicated pooled connection and resets session state before release; the move to transaction-scoped context is part of the remaining hardening gate.

## Technology

| Component | Current technology |
|---|---|
| Backend | Go 1.26.5, Gin, pgx/v5, go-redis |
| Database | PostgreSQL 16, schema-per-tenant logical isolation |
| Cache | Redis 7 for the current POS idempotency middleware |
| Frontend UI | Next.js 15, React 19, TypeScript, Tailwind |
| AI-worker scaffold | Python, APScheduler, Polars, SciPy, Pydantic |
| CI | GitHub Actions: Go build/vet/test and frontend lint/build |

## Quick start

### Prerequisites

- Docker and Docker Compose
- Go 1.26.5
- Node.js 20 (only when running the frontend scaffold)

### 1. Clone and start the verified demo stack

```bash
git clone https://github.com/b45/tenet-commerce.git
cd tenet-commerce
docker compose up -d postgres redis
docker compose up -d api frontend

curl -i http://localhost:8081/health
curl -i http://localhost:8081/ready
open http://localhost:3000
```

Compose starts PostgreSQL, Redis, the API on `localhost:8081`, and the Next.js frontend on `localhost:3000`. PostgreSQL is exposed on `localhost:5433`; Redis is exposed on `localhost:6379`.

### 2. Run the API manually (alternative to Compose)

The backend reads environment variables from the shell; it does not load `.env` automatically.

```bash
set -a
source .env.example
set +a
cd backend
go run ./cmd/api
```

The default API address is `http://localhost:8081`; confirm it with `curl http://localhost:8081/health`.

### 3. Run the frontend manually (alternative to Compose)

```bash
cd frontend
npm ci
npm run dev
```

### 4. Deterministic demo fixtures & database reset

The database comes pre-seeded with reproducible demo fixtures for golden journey testing and demonstrations:
- **Tenant A (`al-barakah-mart`)**: Demo product `SKU-DEMO-01` (unit price IDR 15,000, cost price IDR 10,000, threshold 10, initial stock 0), supplier `SUP-DEMO-VALID-01` with BPJPH Halal certificate `CERT-DEMO-HALAL-2099`, and supplier `SUP-DEMO-EXP-01` with expired certificate.
- **Tenant B (`darussalam-store`)**: Isolated supplier `SUP-DS-DEMO-01` ensuring zero cross-tenant leakage.

To reset the local demo database to a pristine initial state:
```bash
make db-reset
```
The reset script (`scripts/reset_dev_db.sh`) requires explicit opt-in confirmation (`CONFIRM_DEMO_RESET=true`) and strictly guards against resetting non-demo/arbitrary database instances.

### 5. Maintenance mode test

```bash
APP_MAINTENANCE_MODE=true \
APP_MAINTENANCE_MESSAGE='Scheduled maintenance test.' \
docker compose up -d --force-recreate api
curl -i http://localhost:8081/ready
```

The readiness endpoint returns `503`; read-only requests remain available and a dashboard mutation displays the localized maintenance banner. Restore the default with `APP_MAINTENANCE_MODE=false docker compose up -d --force-recreate api`.


## Verification

```bash
cd backend
go build ./...
go vet ./...
go test -race -short ./... # requires a running Docker daemon for Testcontainers

cd ../frontend
npm run lint
npm run build

cd ..
TENET_TEST_POSTGRES_TMPFS=1 ./scripts/benchmark_phase2.sh /tmp/tenet-benchmark-evidence
```

The `backend/integration` suite starts PostgreSQL 16 and Redis 7 through Testcontainers. If Docker is unavailable, setup fails rather than silently skipping the suite. Migrating the remaining host-dependent legacy tests into this suite is tracked by the Phase 2 hardening gate.

## Documentation

- [Implementation Status](docs/IMPLEMENTATION_STATUS.md) — current capability boundary and Phase 3 gate.
- [API Specification](docs/API_SPECIFICATION.md) — REST contract for registered routes; planned endpoints are explicitly labeled.
- [Architecture](docs/ARCHITECTURE.md) — architecture and planned evolution.
- [Architecture Evidence](docs/ARCHITECTURE_EVIDENCE.md) — source references, decisions, and limitations.
- [Benchmark Report](docs/BENCHMARK_REPORT.md) — reproducible isolated workload results.
- [Roadmap](docs/ROADMAP.md) — phase plan and hardening sequence.
- [Phase 3 Frontend Design](docs/FRONTEND_PHASE3_DESIGN.md) — proposed UI/UX, free tooling, design portability, runtime boundaries and delivery gates.
- [Frontend Guidelines](docs/FRONTEND_GUIDELINES.md) — practical design foundations, semantic colours, layouts, reusable patterns, accessibility and feature workflow (Indonesian).
- [POS Design Package](docs/design/README.md) — proposed cash POS flow, API mapping, eleven portable laptop/tablet/phone SVG wireframes and screen acceptance criteria.
- [Database Schema](docs/DATABASE_SCHEMA.md)
- [Sharia Compliance](docs/SHARIA_COMPLIANCE.md)
- [Contributing](docs/CONTRIBUTING.md)

## License

Licensed under the [Apache License 2.0](LICENSE).
