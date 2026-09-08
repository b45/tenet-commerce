# ADR 004: Registry-controlled tenant lifecycle

Status: current implementation decision, audited at `203257a`.

Tenant schemas come from the trusted registry, not raw request identifiers. [Middleware](../../backend/internal/tenant/middleware.go) scopes a pooled connection and resets it before release. [ScopedDB](../../backend/internal/tenant/context.go) offers transaction-local search paths; direct connection consumers still depend on middleware scoping.

Schema-per-tenant keeps domain queries local but shares database resources and administrative privileges. Shared tables would require tenant predicates everywhere; database-per-tenant would add deployment and migration overhead. Neither alternative is implemented here. Schema separation is not physical isolation.

[Migration runner](../../backend/pkg/database/migration/runner.go) validates versions/checksums and serializes the all-tenant runner with an advisory lock. Migration is an explicit operation, not request middleware. Failed migrations must be repaired through a reviewed migration; editing an applied checksum or assuming all tenants migrated together is unsafe. Public tenant provisioning is not implied by the existence of the runner.

Evidence: [tenant migration](../../backend/integration/tenant_migration_test.go) and [isolation](../../backend/integration/tenant_isolation_test.go) suites.
