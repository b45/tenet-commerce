# Tenet Commerce

Multi-tenant retail backend reference implementation for point-of-sale, supply-chain compliance, and double-entry ledger workflows.

> **Current status:** Phase 1–2 backend is implemented. A Phase 2 hardening gate is in progress before Phase 3 begins. See [Implementation Status](docs/IMPLEMENTATION_STATUS.md) for the verified boundary between implemented and planned work.

## What is implemented

- Go API with JWT authentication, role/permission guards, structured request logging, and schema-per-tenant routing.
- POS catalog and category management, checkout, order history, void, daily summary, QRIS configuration, stock adjustment, and low-stock query.
- Supplier, purchase-order, and goods-receipt creation with configurable compliance-certificate checks.
- Double-entry journal generation, chart of accounts, manual journal entry, trial balance, and manager dashboard aggregation.
- Redis response-idempotency middleware for POS checkout, void, and stock adjustment; checkout uses PostgreSQL row locks for inventory mutation.

The important caveats are visible: idempotency is not yet durable across Redis failure, supply-chain receiving needs stronger reconciliation, and integration tests must be made hermetic before Phase 3. The complete hardening criteria are documented in [Implementation Status](docs/IMPLEMENTATION_STATUS.md).

## Planned, not currently implemented

- Phase 3: POS web application, IndexedDB offline queue, service worker replay, and operational dashboards.
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
| Frontend scaffold | Next.js 14, React 18, TypeScript, Tailwind |
| AI-worker scaffold | Python, APScheduler, Polars, SciPy, Pydantic |
| CI | GitHub Actions: Go build/vet/test and frontend lint/build |

## Quick start

### Prerequisites

- Docker and Docker Compose
- Go 1.26.5
- Node.js 20 (only when running the frontend scaffold)

### 1. Clone and start infrastructure

```bash
git clone https://github.com/b45/tenet-commerce.git
cd tenet-commerce
docker compose up -d postgres redis
```

Compose starts only development PostgreSQL and Redis. PostgreSQL is exposed on `localhost:5433`; Redis is exposed on `localhost:6379`.

### 2. Run the API

The backend reads environment variables from the shell; it does not load `.env` automatically.

```bash
set -a
source .env.example
set +a
cd backend
go run ./cmd/api
```

The default API address is `http://localhost:8081`; confirm it with `curl http://localhost:8081/health`.

### 3. Run the frontend scaffold (optional)

```bash
cd frontend
npm ci
npm run dev
```

## Verification

```bash
cd backend
go build ./...
go vet ./...
go test -race -short ./... # requires a running Docker daemon for Testcontainers

cd ../frontend
npm run lint
npm run build
```

The `backend/integration` suite starts PostgreSQL 16 and Redis 7 through Testcontainers. If Docker is unavailable, setup fails rather than silently skipping the suite. Migrating the remaining host-dependent legacy tests into this suite is tracked by the Phase 2 hardening gate.

## Documentation

- [Implementation Status](docs/IMPLEMENTATION_STATUS.md) — current capability boundary and Phase 3 gate.
- [API Specification](docs/API_SPECIFICATION.md) — REST contract for registered routes; planned endpoints are explicitly labeled.
- [Architecture](docs/ARCHITECTURE.md) — architecture and planned evolution.
- [Roadmap](docs/ROADMAP.md) — phase plan and hardening sequence.
- [Database Schema](docs/DATABASE_SCHEMA.md)
- [Sharia Compliance](docs/SHARIA_COMPLIANCE.md)
- [Contributing](docs/CONTRIBUTING.md)

## License

Licensed under the [Apache License 2.0](LICENSE).
