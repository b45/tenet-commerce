# ADR 005: Durable response cache and domain recovery boundary

Status: current implementation boundary; universal atomic command results remain unproven.

[Middleware](../../backend/pkg/idempotency/middleware.go) fingerprints method, concrete path and body, checks Redis, then uses the tenant `idempotency_requests` table. Redis-only deduplication would lose the durable fallback after eviction or outage. PostgreSQL therefore retains the HTTP result; Redis is an acceleration layer.

The HTTP result is written after the handler returns. A crash between business commit and result persistence is possible. [Checkout](../../backend/internal/pos/service.go) and [receipt](../../backend/internal/supplychain/service.go) use domain lookup by key, but recovery identity and payload matching must be evaluated separately for each endpoint. Do not describe the current middleware as an atomic command-result transaction or an unlimited exactly-once guarantee.

An alternative is a command record, payload fingerprint, business mutation and response written in one transaction. That would require explicit integration into every mutating domain and migration of recovery semantics; this ADR does not implement it. Preserve existing keys and audit records during such a migration. Unknown client outcomes must retain their identity rather than generate a fresh sale automatically.

Evidence: [idempotency integration suite](../../backend/integration/idempotency_test.go) and [checkout recovery tests](../../frontend/src/features/pos/checkout.test.mjs). A test's presence is not a passing execution record.
