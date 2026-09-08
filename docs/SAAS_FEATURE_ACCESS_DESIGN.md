# SaaS Feature Access & Entitlement Design

Status: proposed architecture, saved for future implementation. Updated: 2026-09-06.

This document does not establish a commercial package, implement an endpoint, or authorize billing integration. Free, Pro, and Enterprise are illustrative package names; prices, feature allocation, and limits remain undecided.

## 1. Objective and current foundation

Support tenant-specific feature availability and subscription limits without coupling domain code to a billing or feature-flag vendor. Retain the existing Go modular monolith, PostgreSQL, and optional Redis caching.

The repository already contains tenant resolution/isolation and status checks (`backend/internal/tenant/`), role-derived JWT permissions (`backend/pkg/auth/jwt.go`), and API permission guards. These are foundations, not evidence of implemented subscription enforcement. `docs/DATABASE_SCHEMA.md` describes the existing registry and tenant configuration pattern; the entities below are proposals, not applied migrations.

Recommended subscription unit: one tenant/company, with user or branch limits where applicable. Confirm this product decision before implementation.

## 2. Separate policy dimensions

- **Entitlement:** capabilities purchased or granted to a tenant through a package.
- **Permission:** operations a particular authenticated user may perform within that tenant.
- **Tenant setting:** an owner's operational choice within the tenant's granted capabilities.
- **Feature flag:** platform rollout or emergency operational control; not proof of purchase or authorization.
- **Quota:** a typed capacity or usage limit, with explicit counting and reset rules.

A paid tenant does not grant every staff member every permission. An owner cannot unlock an unpurchased capability by changing a setting. Do not encode packages as roles such as `PRO_MANAGER`, or spread `isPremium` checks across components.

Effective access combines verified tenant context, entitlement, permission, applicable operational controls, business invariants, and operation-specific quota checks. Unknown feature keys and missing grants deny access by default. Not every operation consumes a quota; policy must distinguish reads, new writes, and recovery.

## 3. Responsibilities and boundaries

### Backend and database

The backend is authoritative on every protected API operation, including direct requests that bypass the UI. Use a shared policy evaluator for handlers, services, workers, and future synchronization paths. Middleware performs early checks; services enforce transactional invariants and capacity reservations atomically.

Suggested minimal persistence:

- `plans`: stable package identity and explicit versioning policy.
- `plan_features`: stable feature keys and validated, typed grants/limits.
- `tenant_subscriptions`: tenant, package version, lifecycle status, and effective dates.
- Validated tenant module settings using the existing tenant configuration pattern where suitable.
- Audit events for package and setting changes, including actor, tenant, reason, timestamps, and before/after values.

Later additions, only when needed: expiring tenant-specific overrides, usage counters/reservations, and billing event records. Define uniqueness, effective-period overlap rules, unlimited-limit representation, and concurrent quota enforcement before migrations.

Platform subscription metadata may follow the existing shared tenant registry pattern through a dedicated, explicitly tenant-scoped repository. This is not permission to query tenant business data outside its active schema. Tenant identity comes from verified authentication context, never an arbitrary client-supplied tenant ID.

Separate platform administrator authority from tenant-owner authority. Do not reinterpret the existing `SUPER_ADMIN` role as cross-tenant platform access. Any override must be scoped and auditable and cannot bypass domain invariants or user permissions.

### Frontend

Proposed, not registered: `GET /api/v1/me/capabilities` returns effective capabilities, relevant limits, safe denial reasons, and a policy version for the authenticated user and tenant. Final routing and DTOs require contract review.

Use a shared typed capability accessor for navigation and page/action states. Distinguish loading, insufficient subscription, insufficient permission, owner-disabled, quota-exceeded, and service-unavailable states. Do not briefly show protected data while capabilities load. Direct URLs must have meaningful restricted states; APIs still enforce independently.

Scope any capabilities cache to tenant and user, clear it on session/tenant changes, and avoid shared public caching of personalized responses. Refresh after policy changes or stale-policy denials. New UI must follow existing responsive, accessibility, ID/EN/AR, and RTL conventions.

Frontend environment variables are not access controls. Public Next.js variables are bundled at build time. Keep environment configuration for deployment concerns; do not require a new frontend build when a tenant upgrades.

## 4. Commerce safety and lifecycle

- Feature gates cannot disable double-entry accounting, tenant isolation, idempotency, or mandatory strict-mode compliance checks. A ledger reporting UI can be gated while internal checkout accounting remains mandatory.
- Model module dependencies explicitly. Disabling a module must not strand accepted transactions or allow callers to skip required side effects.
- Downgrade must not delete historical business data. Define read/export retention, over-limit behavior, and restrictions on new capacity before launch.
- Distinguish new actions from safe status lookup/replay/recovery of already accepted operations. Recovery remains authenticated and tenant-scoped; it must not generate a new transaction identity to evade a denial.
- Do not use a long-lived JWT `isPro` claim as the sole source of truth. Subscription changes need server-side evaluation and a defined revocation latency.
- If caching is introduced, use bounded freshness, tenant/policy versions, and invalidation on changes. Cache failure may fall back to the authoritative DB; inability to establish permission must not enable paid mutations. Define safe read/recovery behavior separately, never as an authorization bypass.
- Truly offline devices cannot learn revocation immediately. Offline entitlement duration, clock assumptions, activity limits, queued-transaction handling, and reconnect reconciliation remain design gates. Saving this plan does not resolve existing POS durability/recovery blockers.
- Billing integration must verify server-side payment events, process duplicates idempotently, and handle out-of-order updates and reconciliation. A browser success redirect is not payment evidence.

## 5. Vendor-neutral implementation

Keep stable domain feature keys independent of package names and payment product IDs. Illustrative keys: `reports.advanced`, `inventory.multi_location`, `pos.offline`, and `users.max_active`; naming a key does not mean the feature exists.

Keep plan definitions and subscription state in the application's own portable schema. Map external billing identifiers through an adapter. Start with server-managed grants; automatic billing and external feature-management services are not prerequisites.

OpenFeature can later abstract rollout providers if required. It is not a replacement for entitlements, authorization, or transactional quota enforcement. Avoid a generic scripting/rule engine and new infrastructure until concrete requirements justify them.

## 6. Incremental delivery and acceptance

1. **Policy specification:** confirm subscription unit, feature catalog, dependency rules, actor boundaries, quotas, and lifecycle decisions. Define package contents separately from implementation.
2. **One complete vertical slice:** select one existing, non-critical capability; implement persistence, evaluator, API enforcement, capabilities response, and FE restricted states. Do not register placeholder endpoints. Update REST and Postman contracts alongside actual API changes.
3. **Operational maturity:** add scoped management, audit visibility, atomic quotas, and lifecycle transitions. Add automated billing only after access controls are proven.

Required acceptance coverage includes:

- Denied API calls remain denied when bypassing FE; paid tenants still respect user permissions.
- Cross-tenant requests and capability-cache leakage are rejected.
- Tenant owners cannot self-grant paid features or platform authority.
- Missing/unknown grants, cache outages, expiry, and policy changes have explicit results.
- Concurrent requests cannot exceed capacity; retries do not double-consume usage.
- Downgrade preserves data and defined recovery paths; core accounting/compliance still execute.
- Future billing tests cover invalid signatures, duplicates, reordered events, and reconciliation.
- FE includes translated and accessible restricted/loading states across device sizes.

Use unit/integration and CLI checks, plus owner-led manual device review. Automated browser testing is outside the current agreed workflow. Run repository-required backend and frontend checks before implementation completion.

## 7. Decisions still open

- Actual package contents, prices, trial/grace periods, and entitlement effective times.
- Which existing capability is the first vertical slice.
- Quota definitions: active seats versus invitations, branch counting, metered periods/time zones, and reservation release rules.
- Downgrade read/export policy and handling of accepted/in-flight work.
- Platform administrator authentication and authorization model.
- Maximum policy staleness, offline authorization bounds, and billing provider selection, if any.

## References

- [OWASP Authorization Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html): server-side authorization, deny by default, and per-request checks.
- [Next.js environment variables](https://nextjs.org/docs/pages/guides/environment-variables): public build-time configuration behavior.
- [OpenFeature providers](https://openfeature.dev/docs/reference/concepts/provider/): optional provider abstraction for feature flags.

References consulted on 2026-09-06. Implementation details must be checked against the project's installed versions when work begins.
