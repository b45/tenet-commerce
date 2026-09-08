# Architecture evidence and verification boundary

Source audit: commit `203257aec9ca5c9e3211a22df548e4d266bee22a`. References below identify implementation and tests to inspect; they do not certify a deployment or assert that tests were executed for this document.

## Runtime and trust boundary

The browser calls Next.js BFF routes. The BFF forwards authenticated requests to the Go API. PostgreSQL stores business records; Redis provides response caching and coordination. The Compose topology is a local demo configuration, not a production availability guarantee.

- [Compose services](../docker-compose.yml) define frontend, API, PostgreSQL and Redis.
- [BFF proxy](../frontend/src/app/api/backend/[...path]/route.ts) reads the access cookie, forwards the bearer token and checks mutation origin.
- [Tenant middleware](../backend/internal/tenant/middleware.go) resolves the trusted registry schema, sets the connection search path and resets the connection before release.
- [ScopedDB](../backend/internal/tenant/context.go) provides transaction-local search paths. Existing services also accept the middleware-scoped pooled connection directly; do not claim every call uses ScopedDB.
- [Tenant isolation tests](../backend/integration/tenant_isolation_test.go) and [migration tests](../backend/integration/tenant_migration_test.go) are the relevant verification entry points.

## Idempotency and commit boundaries

[Idempotency middleware](../backend/pkg/idempotency/middleware.go) fingerprints method, concrete path and body. Redis is a fast path; `idempotency_requests` is a PostgreSQL tenant table. Mismatched payloads return a conflict. Completed records can replay the stored response.

The middleware updates the durable HTTP response **after** the handler returns. This update is not part of the handler's business transaction. Consequently, middleware presence alone does not prove atomic command-result persistence across a process crash.

[POS checkout](../backend/internal/pos/service.go) separately checks the transaction idempotency key and commits the sale, items, stock changes and journal within its business transaction. [Goods receipt persistence](../backend/internal/supplychain/repository.go) also provides receipt lookup by idempotency key. These recovery mechanisms must be evaluated per operation, including payload mismatch and response reconstruction; they are not a universal exactly-once guarantee. See [idempotency integration tests](../backend/integration/idempotency_test.go).

## Receipt transaction

[Supply-chain service](../backend/internal/supplychain/service.go), [repository](../backend/internal/supplychain/repository.go) and [compliance policy](../backend/internal/supplychain/compliance.go) are the implementation references. A receipt locks its purchase order, evaluates compliance and reconciles requested items against outstanding quantities. Inline QC requires delivered quantity to equal accepted plus rejected quantity, with a reason for rejections. Accepted quantities drive stock and valuation; the receipt and ledger posting share a transaction.

Relevant tests: [supply chain](../backend/integration/supplychain_test.go), [operational behavior](../backend/integration/supplychain_operational_test.go), [certificate validity](../backend/integration/compliance_validity_test.go), and [ledger invariants](../backend/integration/ledger_invariants_test.go). QC behavior is repository policy; this document does not attribute it to an unavailable external ERP specification.

## Public scope

- IndexedDB currently supports catalog and cart drafts. Paid offline transaction replay requires the policies in [Phase 3 design](FRONTEND_PHASE3_DESIGN.md); it is not enabled by the existence of browser storage.
- QRIS configuration does not establish provider settlement verification.
- AI auditing and Zakat calculation remain target scope.
- Maintenance rejects mutations and makes readiness unavailable. The banner appears after a maintenance response; it is not a proactive status subscription. Its localized text is independent of the backend custom error message.
- [Actual CI](../.github/workflows/ci.yml) is the workflow source of truth. Passing application CI does not prove that a Docker image was successfully built and exercised.

## Remaining evidence work

Topology, tenant boundary, receipt sequence and core ERD are recorded in [architecture](ARCHITECTURE.md); decision records and [reconciliation commands](verification/README.md) are linked there. This documentation candidate passed backend build/vet/race-short, frontend lint/build, relative-link checks and Mermaid CLI rendering of all three diagrams. The reconciliation script ran against a fresh canonical bootstrap in PostgreSQL 16 with tmpfs storage; both result sets were empty. This verifies the fixture and query execution, not post-load or production data. Full integration/browser suites were not rerun for this documentation-only change. Benchmark publication and clean-clone showcase are separate tasks. Do not infer their completion from merged runtime changes.
