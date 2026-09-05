# Phase 3 Frontend Design and Delivery Plan

> Status: proposed architecture and delivery plan; frontend features are not implemented.
> Updated: 2026-09-05. Audience: maintainers, designers and reviewers.
> Scope: portable design assets, free tooling, browser POS and operational interfaces.

Implementation and design work follows the [Frontend Guidelines](FRONTEND_GUIDELINES.md), with a reusable [screen specification template](design/SCREEN_SPEC_TEMPLATE.md). The guidelines explain the visual/interaction rules; this document defines delivery scope and dependencies.

First bounded design package: [cash POS contract, flow, three SVG wireframes and screen specifications](design/README.md), revision `pos-lowfi-0.1`. These are reviewable proposals, not implemented features or a closed production-readiness gate.

## 1. Objective and boundaries

Build a frontend that preserves tenant isolation, transaction integrity, Halal procurement rules and balanced accounting. Design and implementation must remain usable without a particular design SaaS, AI company, paid plugin or deployment provider.

Portability means an independently recoverable project and a documented migration path. It does not imply lossless interchange between every design editor or zero engineering cost when replacing React/Next.js. Free software licensing also does not remove hardware, hosting, maintenance or optional AI inference costs.

The Phase 3 baseline uses one design editor, text-based specifications, local build/test commands and an ordinary deployable web application. Purchasing a subscription is not an acceptance dependency. AI assists development; the cashier application must operate without an AI account or model API.

This plan supersedes the original roadmap's Phase 3 calendar assumptions. Completion of this document does not certify backend readiness. The older status documents disagree about the hardening gate; reconcile their claims against code and execution evidence in work package P3-00 before enabling transactional features.

## 2. Current implementation and readiness questions

Repository paths below are evidence for current behavior; recommendations are separate from active API contracts.

| Area | Observed implementation | Required Phase 3 decision/evidence |
|---|---|---|
| Frontend | [Package](../frontend/package.json) and [home page](../frontend/src/app/page.tsx) are a Next.js 14.2.35/React 18 scaffold | Evaluate supported Next.js/React/Node versions before adding features; record migration and lint-command changes |
| Permissions | [Role definitions](../backend/pkg/auth/jwt.go), endpoint guards in domain handlers | Navigation follows actual permission strings and role restrictions; UI hiding is not authorization |
| Authentication | [Auth handler](../backend/internal/auth/handler.go) provides login/refresh/me and JSON token pairs | Browser session ownership, CSRF protection, logout/revocation and expired-session recovery |
| Catalog | [POS repository](../backend/internal/pos/repository.go) lists all active products | Search/pagination, barcode lookup and versioned catalog refresh; inactive-product management also needs an agreed read contract |
| Money | [Money utility](../backend/pkg/money/money.go) uses exact integer amounts; [POS service](../backend/internal/pos/service.go) uses it, while API DTOs still expose float values | Confirm rounding and bounds end-to-end; do not confuse internal exact arithmetic with a fully migrated wire contract |
| Idempotency | [Middleware](../backend/pkg/idempotency/middleware.go) includes Redis and PostgreSQL records on selected mutations | Prove recovery if business commit succeeds but response recording fails; prove key lifetime, lease recovery and all mutation coverage |
| Supply chain | [Handler](../backend/internal/supplychain/handler.go) includes read/lifecycle and PO/GR endpoints | Validate partial receipt, cancellation/revocation audit fields and compliance errors against actual responses |
| Ledger | [Handler](../backend/internal/ledger/handler.go) exposes accounts, entry list/create/reverse and trial balance | Do not invent a journal-detail endpoint; use available list data or explicitly propose a backend change |
| Payments | CASH, QRIS and SIMULATED_CARD exist in POS requests | Document QRIS confirmation semantics; no claim of gateway settlement or external refund automation |

Next.js currently lists 14.x as unsupported. Select a supported release when P3-02 begins, then update repository stack documentation through a dedicated change. This planning task does not upgrade dependencies. [Next.js support policy](https://nextjs.org/support-policy)

The product's Halal tag is not evidence of a current, independently verified certificate: catalog code derives a flag from tags such as `HALAL_MUI`. UI copy must distinguish a catalog tag from supplier certificate validity and avoid implying regulatory certification of the application.

Registered routes remain authoritative. API changes proposed here are dependencies, not newly active endpoints; implementing them requires synchronized [API Specification](API_SPECIFICATION.md) and both applicable Postman collections.

## 3. Free tools and exit paths

| Concern | Default | Portable source/delivery | Exit path and limitation |
|---|---|---|---|
| Product/UX specifications | Markdown in local Git repository | `.md`, ordinary relative links | Read in any editor; no cloud board required |
| Task flows | Mermaid text in Markdown | Source text plus optional SVG preview | Source remains editable if a renderer is unavailable |
| Manually arranged diagrams | draw.io desktop, only when needed | `.drawio` XML and SVG preview | Continue offline; visual exports alone may omit editable graph data |
| UI design/prototype | Penpot | Native `.penpot` export including required libraries, plus per-screen SVG, tokens and behavior specifications | Restore in another Penpot instance; SVG supports visual migration, not complete component/prototype migration |
| Design tokens | DTCG-format JSON | `.tokens.json`; generated CSS variables | Small local transform, version-pinned; no paid token-sync plugin |
| UI review in popular tools | Optional Figma or Miro free access | Import SVG/PNG for review | Review results must return to the source specification; these services are not the sole archive |
| Implementation/testing | TypeScript/React, local CLI, browser tools | Source, lockfile, commands, test reports | No paid visual regression or hosted test service required |
| Delivery | Standard Node runtime or container | Build instructions, environment contract | Self-hosting is possible; managed hosting remains optional |

Penpot provides an open-source implementation and a cloud $0 plan at the time of research. Its cloud offering currently lists 10 GB storage and seven days of automatic version history; keep independent exports. Start on the free service if convenient, and retain self-hosting as the recovery path rather than taking on a design-server maintenance task immediately. [Penpot pricing](https://penpot.app/pricing), [source repository](https://github.com/penpot/penpot)

Penpot documents native file export/import and options to include shared libraries. Preserve those libraries when archiving; an exported file is not automatically a universal interchange format. [Penpot export/import](https://help.penpot.app/user-guide/export-import/export-import-files/)

draw.io stores editable diagram data in `.drawio`/XML and can include diagram data in certain exports. Keep the native file even when distributing an SVG. [draw.io file formats](https://www.drawio.com/docs/manual/editor/save-file-formats/)

Figma imports SVG as editable vector layers, but this does not recreate native Auto Layout, variants or click interactions. Miro imports SVG as a solid image. Rebuilding either native object model needs extra work; do not make a paid converter necessary. [Figma imports](https://help.figma.com/hc/en-us/articles/360040028034-Add-images-and-videos-to-designs), [Miro formats](https://help.miro.com/hc/en-us/articles/360017731613-Supported-file-formats)

### Portability acceptance test

At each approved design milestone:

1. Export the native design and its libraries, screen SVGs and token JSON to files outside the design service.
2. Record editor version, fonts/licenses, export date and stable screen IDs in the asset manifest.
3. Import the native export into a fresh compatible Penpot project; inspect components, text and prototype links.
4. Open one SVG in an independent renderer/editor; compare text, dimensions and key visual states. Record import losses explicitly.
5. Confirm the screen's behavior can be reconstructed from its Markdown specification without access to the original board or chat.

This is a planned exit test, not a test already performed by this document.

## 4. Design artifact contract

Public source artifacts describe the product. Personal planning, model usage, task handoffs and draft exploration remain local. The following directories are proposed and created only when their contents exist:

```text
docs/design/
  README.md                    asset index, approved revision and licenses
  flows/*.mmd                  one source graph per flow
  screens/*.md                 layout, fields, states and interactions
  tokens/*.tokens.json         authoritative semantic tokens
  exports/*.svg                derived visual snapshots
  source/*.penpot              milestone native archive, if reasonable in size
docs/adr/                      architecture decisions and alternatives
docs/local/                    ignored working plans and AI handoffs
```

Large native archives should have a version/checksum manifest and a maintainer-owned independent backup with a documented restore process. Do not leave their only copy behind an expiring SaaS link or require paid Git LFS storage. Ordinary compact sources/previews belong in Git; avoid duplicate archives for every intermediate revision. Internal files are ignored and need their own local/offline backup.

A screen specification contains: stable ID, intended role/job, API dependencies, fields and validation, layout at laptop/tablet/phone widths, focus order/keyboard actions, loading/empty/error/offline/permission states, destructive-action explanation and measurable acceptance examples. SVG is a visual reference; text and state specifications preserve the behavior that SVG cannot carry.

Tokens use semantic names such as `color.status.warning` and `space.controlGap`; source JSON generates CSS variables. Editors consume copies, and accepted editor changes must update the source JSON. Use a documented subset of the [Design Tokens Community Group format](https://www.designtokens.org/tr/2025.10/format/); it is a community specification, not a W3C Recommendation. Check chosen editor support before promising automatic token round trips.

Each font/icon/UI kit includes its upstream source and redistribution license. Self-host allowed font assets; do not require premium icon libraries or remotely served fonts to render the design/application.

## 5. User experience and permissions

Two shared workspaces are sufficient: a fast POS layout and an operations layout. Indonesian is the initial interface language; dates and IDR amounts use explicit formatting conventions. Tenant timezone is a contract decision, not the browser's implicit timezone.

| Role | Landing/job | Navigation supported by current permissions |
|---|---|---|
| Cashier | POS: complete a sale and recover uncertain outcomes | POS, orders/void, daily summary; inventory read for catalog |
| Manager | Dashboard: review sales, stock and certificate alerts | Dashboard, POS/orders, inventory write, supply chain, ledger read |
| Compliance officer | Suppliers: inspect validity and trace documents | Supply chain, traceability, inventory read, ledger read |
| Financial admin | Ledger: inspect and post balanced journals | Ledger read/write, inventory read |
| Super admin | Operational overview in the authenticated tenant | All registered functions permitted by backend; no invented cross-tenant switcher |

The manager dashboard has a role restriction in addition to ordinary permission checks. QRIS reads require `inventory:read`; configuration writes require `inventory:write`. Logout/tenant change must reset visible data immediately. A tenant slug typed into the UI never overrides the JWT tenant.

### Screen families and flow coverage

| ID | Task and screens | Critical acceptance behavior |
|---|---|---|
| UX-01 | Tenant login, expired session, forbidden | Generic credential error; return to permitted route; pause queued commands during reauthentication |
| UX-02 | POS catalog/search/barcode, cart | Scan cannot hijack text fields/dialogs; quantity edits preserve focus; local totals follow money policy |
| UX-03 | Cash payment, receipt, reprint | Tender/change validated; unknown outcome preserves command identity; reprint creates no sale |
| UX-04 | Local draft, offline queue, sync center | Show saved/queued/confirmed distinctly; unreconciled sale remains visible after restart |
| UX-05 | Order list/detail, full void, daily summary | Permission and reason checks; explain stock/accounting reversal; no invented shift-close or partial-refund workflow |
| UX-06 | Products/categories, low stock, adjustment | Distinguish stock adjustment from metadata edit; explain quantity and ledger impact |
| UX-07 | Supplier/certificate registry, renewal/revocation | Show authority, dates and validity source; link certificate documents; no upload widget without upload support |
| UX-08 | PO creation/detail/cancel, partial GR/detail | Recheck certificate at receipt; show ordered/received/remaining quantities and reject over-receiving |
| UX-09 | Product document traceability | Navigate supported supplier/PO/GR links; do not claim batch-level traceability if API only tracks documents |
| UX-10 | COA, journals, reversal, trial balance/dashboard | Debit/credit difference visible; posted records not editable; show report date/freshness |

Design generic form/table/confirmation patterns once. Fully design error and recovery states for POS/offline/compliance/ledger; derive routine CRUD screens from those patterns. Dashboard charts require actual time-series data; the current aggregate endpoint does not justify invented sales trends.

Initial detailed screens: POS register, cash payment/receipt, sync center, supplier compliance/PO block. Login and basic tables can start from a licensed reusable pattern. Visual direction: neutral surfaces, restrained emerald/amber/red semantic states, readable typography and tabular numerals. Test keyboard use, scanner behavior, focus restoration and 80 mm printing on actual target devices. Scope update 2026-09-05: laptop, tablet (portrait/landscape) and phone are operational design targets; phone checkout is no longer deferred as a product scope decision. Release still requires validation. See [responsive POS specification](design/screens/POS_RESPONSIVE.md) for eight additional static frames and acceptance cases. Existing delivery-hour estimates have not yet been recalibrated for phone operational QA; measure a responsive vertical slice before revising the forecast.

## 6. Runtime architecture and replaceable boundaries

Keep the existing monorepo and one frontend deployment. Framework selection is independent of deployment provider; Next.js supports ordinary Node/container self-hosting. An authenticated BFF requires a server and cannot be shipped as a static-only export. [Next.js self-hosting](https://nextjs.org/docs/app/guides/self-hosting)

Proposed modules:

```text
frontend/src/
  app/                  routing, layouts, thin BFF routes
  features/             auth, pos, inventory, supply-chain, ledger, manager
  components/           shared UI and domain components
  lib/                  HTTP adapter, permissions, validation, money
  offline/              IndexedDB adapter, outbox, sync runner
```

The Go backend owns business validation. BFF code owns session transport, forwards keys/errors/trace IDs and does not reimplement checkout or journal posting. Restrict upstream routes rather than building an arbitrary proxy. Protect cookie-authenticated mutations against CSRF, secure session cookies, prevent shared caching of tenant data, serialize refresh attempts and define server-side revocation. A cookie wrapper alone does not revoke already issued refresh tokens.

Candidate dependencies, to be checked for supported versions/license at implementation: Radix/shadcn-style primitives, TanStack Query for server state, React Hook Form and Zod for forms, Dexie for IndexedDB, Vitest/Testing Library and Playwright. Add a global state library only if a reducer is insufficient. Storybook is optional after repeated components justify it. Keep a dependency/license inventory and standard CLI test entry points.

Feature code talks to small HTTP/storage interfaces where these isolate transport and browser persistence. Avoid a general plugin architecture, custom component framework or paid telemetry dependency. Test fixtures, if required, are explicitly test-scoped and must never register fake production API routes or substitute fabricated metrics in the application.

Plan a machine-readable OpenAPI contract alongside the current API specification and Postman files. Generated TypeScript types can be regenerated without vendor services; runtime validation still handles malformed server responses. This task does not generate or claim a complete OpenAPI definition.

## 7. Offline safety and recovery design

Deliver offline catalog/cart persistence first. Enable offline cash sales only after stock, price, payment and recovery policies have acceptance tests. Disconnected terminals cannot guarantee globally current stock or zero overselling of physically handed-over goods using a stale cache. Any offline sale policy must accept that operational risk or introduce backend inventory allocation/reservations.

### Proposed state transitions

```text
DRAFT -> QUEUED -> SENDING -> CONFIRMED
                    |-- transient transport/5xx/429 --> RETRY_WAIT -> SENDING
                    |-- session expired -------------> AUTH_REQUIRED
                    |-- permanent business conflict -> NEEDS_REVIEW
                    |-- result not known ------------> OUTCOME_UNKNOWN
```

State names describe local client state, not new server transaction statuses. Final receipts use server confirmation; queued receipts explicitly state pending synchronization.

- Persist command ID/key, immutable request body, tenant/user ownership, local sale time, catalog version, amounts used for the local receipt, attempt count and last result before showing a saved receipt. Keep audit metadata outside the request body unless the API supports it.
- Retry timeouts and eligible failures with the same identity. Classify by error code: an in-flight idempotency `409` can wait/retry, while insufficient stock or payload mismatch needs review. Handle `429` using server retry guidance. Do not blindly classify all `4xx` as permanent or all failures as safe to resubmit under a new key.
- Before replacing a failed command, establish that the original did not commit. A corrected command receives a new ID only after that resolution, retaining a link to the original. Stock rejection after cash/goods exchange needs documented refund/reconciliation, not silent record deletion.
- Confirm backend behavior if commit succeeds before idempotency response storage. Test retention beyond 24 hours, interrupted leases and duplicate sends across tabs. Middleware presence alone is not evidence of exactly-once effects.
- Freeze the local priced receipt. Since the current checkout sends SKU/quantity and the server prices at replay time, offline sale enablement needs an explicit price-snapshot/quote or variance-resolution contract. Also define original cashier attribution, local sale time versus server posting date, and stale permissions.
- Synchronize oldest commands first within one tenant/terminal queue; define handling of a blocked head without silently reordering stock-dependent sales. Only one tab/worker holds the sender lease; a crashed lease can recover.
- Background Sync is an enhancement. Supply a foreground runner on app start, focus, reconnect and manual retry; check API reachability, not just `navigator.onLine`. Browser support is limited and requires a secure context. [MDN Background Synchronization](https://developer.mozilla.org/en-US/docs/Web/API/Background_Synchronization_API)
- Expiry locks the UI and pauses sending. Reauthentication must match command ownership; never replay a previous cashier's queue under an unrelated account. Namespace storage by tenant/user/terminal/schema version, but do not treat namespacing as encryption or protection from a device administrator/XSS.
- On logout clear credentials, rendered data and disposable caches. Preserve unsent commands in a locked recovery store; define access/retention and a same-owner recovery procedure before implementation. Never erase a paid-but-unsynced sale automatically. Without an approved secure persistence policy, permit offline drafts only.
- If storage is unavailable/full, fail to a clear unsaved state. Cache eviction, private browsing, OS shutdown and loss of the device prevent any honest guarantee of 100% browser durability. Test database migrations and service-worker updates with pending commands.

These rules refine the older offline diagram in ARCHITECTURE.md. The active checkout route is `/api/v1/pos/checkout`; the older `/api/v1/transactions` example is not registered. A universal 30-second resync guarantee is not an acceptance promise across queue sizes and browser conditions.

## 8. Delivery packages and gates

Packages are planning units, not submitted GitHub issues. Create issue-form drafts with repository metadata only when turning a package into a ticket.

| Package | Dependencies | Reviewable output / exit criterion | Focused hours estimate |
|---|---|---|---:|
| P3-00 Readiness | None | Route/permission/error map; support-version decision; documented backend dependency list; reconcile status evidence | 16–24 |
| P3-01 UX and portability | P3-00 for transactional assumptions; exploration can begin earlier | UX-01..10 outlines; four detailed screen groups; token sample; one export/reimport test | 24–40 |
| P3-02 Foundation | P3-00, accepted UX basics | Supported framework baseline, secure session ADR, shells, permissions, HTTP adapter, test harness | 32–48 |
| P3-03 Online POS | P3-02, money/session contracts | Scan/cart/cash sale/receipt, history/void/summary; real-backend golden path and failure path | 56–80 |
| P3-04 Offline drafts | P3-03 | Catalog storage, restore/migration, foreground sync runner, isolation; no unsupported offline-settlement claim | 56–88 |
| P3-05 Offline cash | P3-04 and resolved backend recovery/price policy | Queue, lease, retries, unknown-outcome recovery and reconciliation tests | 64–112 |
| P3-06 Operations | P3-02, relevant contracts | Inventory, certificate, PO, partial GR and traceability journeys | 64–96 |
| P3-07 Finance/dashboard | P3-02, ledger contracts | Actual aggregates, balanced journal/reversal, trial balance, permission tests | 40–64 |
| P3-08 Hardening | Delivered feature packages | Accessibility, device/print, security, performance and restore acceptance | 48–80 |
| P3-09 Handoff | P3-08 | Portable design archive, reproducible build, release limitations and maintenance guide | 16–24 |

Planning total: **416–656 focused hours**, plus 25% contingency = **520–820 hours**. At 30 focused hours/week this is approximately **18–28 weeks**; at 15 hours/week **35–55 weeks**. These are estimates, not delivery commitments. They include feature testing throughout; P3-08 covers cross-feature acceptance. Large new backend capabilities or payment-gateway integration require a separate estimate.

A portfolio milestone can ship P3-00..04 plus appropriate P3-08/09 checks with clearly disclosed offline-draft scope. Full Phase 3 remains incomplete until accepted offline cash and operational/finance journeys are verified. Do not compress the full plan into the original two-week frontend slot.

### Acceptance gates

- **Design:** task/state coverage, actual API mappings, accessible interactions, reviewed screen specifications and successful portability exercise. Real-user walkthroughs are preferable; simulated role review must be labeled as such.
- **Contract:** demonstrate tenant isolation, exact-money boundaries, idempotent ambiguous-result recovery, certificate hard-blocks and balanced ledger. Missing evidence blocks the dependent feature, not all wireframing.
- **Functionality:** Playwright exercises actual Go/PostgreSQL/Redis golden journeys, denied permissions and domain errors. No placeholder production routes, fake finance values or invisible failed queues.
- **Accessibility:** target WCAG 2.2 AA; automated scans plus keyboard/focus, zoom/reflow and screen-reader checks. Automation alone cannot establish conformance. [WCAG 2.2](https://www.w3.org/TR/WCAG22/)
- **Performance:** measure warm-cache scan-to-cart on named target hardware, initial catalog hydration at representative/50k scale, queue recovery and first interaction. Proposed warm-cache p95 target: 150 ms. Report observed results and dataset, not unsupported SLA claims.
- **Release:** required backend build/vet/race-short suite and frontend lint/build pass; browser/print tests completed; deployment runs without a specific host SDK; exports restore from independent files.

## 9. Provider-independent contribution contract

Any human or AI-assisted contribution uses repository files, stable UX/package IDs, ordinary patches and the same acceptance checks. No particular model, editor extension, MCP connector, cloud memory or chat-history export is necessary to understand or implement the design.

Public decisions live in this specification and accepted ADRs. The chosen model may change without modifying the product architecture. Provider-specific prompts, quota information and temporary handoff notes remain in ignored local files. Maintainers control commits and remote actions under the repository governance rules.

Review accepted work through behavior and test evidence rather than the vendor that generated it. Read only the relevant contract/feature files for a task; use short reproducible examples when another contributor takes over. This preserves continuity while keeping context and review effort manageable.

## 10. First implementation-ready design package

Start P3-00/P3-01 with UX-02/03/04: one desktop POS wireframe, payment/receipt states and queue recovery states. Add a small semantic-token sample and one native Penpot/SVG export test before multiplying screens. Resolve the price/session/idempotency dependencies as ADRs, then expand to compliance and finance. Remaining deliverables are explicitly planned; no native design file or working FE feature is produced by this planning document.
