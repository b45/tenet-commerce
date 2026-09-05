# Implementation Status

> **Status date:** 2026-09-03  
> **Purpose:** distinguish code that is registered and exercised today from design work scheduled for later phases. The route manifest in `backend/cmd/api/router.go` is the runtime source of truth.

> **Planning addendum (2026-09-05):** the [Phase 3 design proposal](FRONTEND_PHASE3_DESIGN.md) records current code observations and a readiness work package. The historical matrix below is not fully synchronized with newer durable-idempotency and supply-chain lifecycle code, and its gate status conflicts with ROADMAP.md. Reconciliation is pending; this addendum does not certify backend readiness. Frontend implementation remains a scaffold while design planning proceeds.

## Current scope

The repository currently contains a Go backend for the Phase 1–2 domain slice. Phase 3 and Phase 4 are intentionally not started until the hardening gate described below is complete.

| Area | Status | Evidence / boundary |
|---|---|---|
| Authentication and RBAC | Implemented, hardening in progress | Login, refresh, and identity endpoints; JWT and permission middleware. Startup now rejects an unset/default JWT secret outside development. Refresh-session revocation and further production configuration hardening remain planned. |
| Tenant routing | Implemented, hardening in progress | Schema-per-tenant routing through a request-scoped PostgreSQL connection; session state is reset before pool release. Transaction-scoped search-path isolation and migration runner remain hardening work. |
| POS | Implemented | Product/category CRUD, checkout, order history, void, QRIS configuration, stock adjustment, and low-stock query. |
| Idempotency | Partial | Redis middleware is mounted on checkout, void, and stock adjustment only. Durable database-backed idempotency is a hardening-gate requirement. |
| Supply chain | Implemented creation slice, hardening in progress | Supplier, purchase-order, and goods-receipt creation with configurable certificate checks. Read/lifecycle APIs, partial receiving, and stronger reconciliation are not yet complete. |
| Ledger | Implemented service layer | Chart of accounts, entries, manual entries, and trial balance. Database-level append-only and exact-money hardening remain planned. |
| Manager dashboard | Implemented | Aggregated dashboard endpoint. |
| Frontend/offline POS | Planned — Phase 3 | The Next.js project is a starter scaffold; no POS client, IndexedDB queue, or service worker is implemented. |
| AI auditor and Zakat | Planned — Phase 4 | The Python worker is a scheduler scaffold; no extraction, anomaly analysis, report persistence, or Zakat API is implemented. |
| Production delivery | Planned — Phase 4 | Compose currently starts PostgreSQL and Redis for development only; it does not ship API, frontend, or production orchestration. |

## Phase 2 hardening gate

Phase 3 starts only after the following evidence is available:

1. Hermetic PostgreSQL and Redis integration tests run in CI without connectivity-based skips.
2. Tenant context is transaction-scoped and proven safe under pooled-connection reuse.
3. Critical mutations have durable, request-fingerprint-aware idempotency.
4. Purchase-order, goods-receipt, compliance, inventory, and ledger invariants are tested under concurrency.
5. Money representation and posted-ledger mutability have an explicit, enforced policy.
6. Public API documentation, Postman collections, benchmarks, and runtime instructions match the registered implementation.

The detailed execution plan is intentionally maintained in `docs/local/` and excluded from the public repository.
