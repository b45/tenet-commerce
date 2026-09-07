# Implementation Status

> **Status date:** 2026-09-08
> **Purpose:** Distinguish code that is registered and exercised today from design work scheduled for later phases. The route manifest in `backend/cmd/api/router.go` is the runtime source of truth.

> **Planning & Hardening Sign-Off (2026-09-07):** Phase 2 Hardening Gate (G2) has officially been **VERIFIED and SIGNED OFF**. All 10 design invariants (I-01 through I-10) are proven with passing evidence across hermetic integration tests, durable idempotency, tenant isolation, append-only stock movements, exact-money ledger balance, compliance checks, and API contract parity.

## Current scope

The repository contains a fully hardened Go backend for Phase 1–2 domains and initial Next.js 15 UI client features for Phase 3.

| Area | Status | Evidence / boundary |
|---|---|---|
| Authentication and RBAC | Implemented & Hardened | Login, refresh, identity endpoints (`/api/v1/auth/me`), permission introspection (`/api/v1/me/capabilities`), JWT and RBAC permission middleware (`RequireRole`, `RequirePermission`). |
| Tenant routing | Implemented & Hardened | Schema-per-tenant isolation (`search_path = tenant_{uuid}`) through request-scoped PostgreSQL connection; physical connection state reset (`RESET ALL`) before pool release. Safe zero-downtime tenant migrations runner. |
| POS | Implemented & Hardened | Product/category CRUD, checkout, order history, void, QRIS configuration, stock adjustment, low-stock query, stock movement card report (`/api/v1/pos/inventory/card`), and stock overview (`/api/v1/pos/inventory/overview`). |
| Idempotency | Implemented & Hardened | Durable Redis database-backed idempotency middleware mounted on all state-mutating endpoints (`POST /checkout`, `POST /orders/:id/void`, `POST /inventory/adjust`, `POST /suppliers`, `POST /purchase-orders`, `POST /goods-receipts`, etc.). |
| Supply chain | Implemented & Hardened | Supplier registry, certificate tracking, purchase-order issuance and cancellation serialization, goods-receipt with inline QC audit, atomic stock movements, and document-level product traceability. |
| Ledger | Implemented & Hardened | Chart of accounts, entries, manual entries, reversal entries, and trial balance. Immutable append-only ledger entries and exact integer money calculation invariants enforced. |
| POS Web Client | Implemented — Phase 3 (P3-03, P3-04) | Next.js 15 POS interface with catalog, category filters, barcode search, cart management, cash tender modal, 80mm thermal receipt printing, daily sales summary modal, order history, and tenant-scoped IndexedDB catalog/drafts. Browser smoke and golden transaction journey run in CI. |
| Supply Chain Hub | Implemented — Phase 3 (P3-06) | Supplier directory, multi-standard certificate tracking, certificate registration, validity countdown, and status badges. |
| Purchase Orders | Implemented — Phase 3 | Localized supplier/product lookup, PO creation, list/detail remaining balances, and eligible cancellation at `/supply-chain/po`. |
| Runtime operability | Implemented — Phase 3 hardening | `/health` liveness, dependency-aware `/ready`, bounded startup, graceful shutdown, and non-root API Compose image. |
| Internationalization (i18n) | Implemented — Phase 3 | Zero hardcoded strings; 1:1 key parity across Indonesian (`id`), English (`en`), and Arabic (`ar`) with full RTL/LTR bidirectional support. |
| Offline Cash Settlement & Replay | Planned — Phase 3 (P3-05) | Gated offline cash outbox queue with monotonic replay and background sync reconciliation. |
| AI auditor and Zakat | Planned — Phase 4 | Python worker scheduler scaffold; anomaly detector and Zakat calculation engine planned for Phase 4. |
| Production delivery | Planned — Phase 4 | Compose setup for dev; production orchestration planned for Phase 4. |

## Phase 2 hardening gate

Phase 2 Hardening Gate (G2) has been fully satisfied:

1. Hermetic PostgreSQL and Redis integration tests run in CI without connectivity-based skips.
2. Tenant context is transaction-scoped and proven safe under pooled-connection reuse (`RESET ALL`).
3. Critical mutations have durable, request-fingerprint-aware idempotency.
4. Purchase-order, goods-receipt, compliance, inventory, and ledger invariants are tested under concurrency.
5. Money representation and posted-ledger mutability have explicit, enforced exact-money policies.
6. Public API documentation, Postman collections (`Tenet_Commerce_Full_API.postman_collection.json`), benchmarks, and runtime instructions match 100% of the registered implementation (44/44 endpoints).
