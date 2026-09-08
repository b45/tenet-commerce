# Tenet Commerce portfolio showcase

Tenet Commerce is a multi-tenant retail reference system for POS, halal supply-chain controls and Sharia double-entry accounting. The current release boundary is the online POS and operational APIs; offline paid replay, AI audit, Zakat and production availability remain planned.

## 90-second path

1. Start the clean demo stack with the [README quick start](../README.md#quick-start).
2. Sign in to `http://localhost:3000` with tenant `al-barakah-mart`, email `manager1@albarakah.com`, and password `Password123!` (development fixture only), then inspect the POS, inventory, supply-chain, ledger and manager screens.
3. Follow the API and data boundaries in [Architecture Evidence](ARCHITECTURE_EVIDENCE.md).
4. Run the required checks and the isolated benchmark harness from the README.

## What the evidence shows

- Tenant routing resolves a trusted schema and resets pooled connection state.
- POS checkout locks product and inventory rows, posts stock and ledger effects atomically, and preserves idempotency identity.
- Purchase orders and goods receipts apply certificate policy and inline QC; rejected goods do not become accepted stock.
- The frontend uses a Next.js BFF with HttpOnly session cookies and localized ID/EN/AR UI.
- CI runs backend, frontend and Browser E2E checks. The benchmark report adds isolated PostgreSQL/Redis measurements with raw JSONL output.

## Honest boundaries

The Compose stack is a reproducible demo runtime. It is not a production SLA, hosted service or TLS deployment. The HTTP idempotency response record is reconciled after the business handler, so crash recovery remains endpoint-specific. Catalog and cart drafts use IndexedDB; paid offline replay needs an explicit price, stock, ownership and recovery policy. QRIS configuration is not provider settlement verification. AI and Zakat sections are planned design work.

## Reviewer links

- [Architecture](ARCHITECTURE.md)
- [Implementation Status](IMPLEMENTATION_STATUS.md)
- [Benchmark Report](BENCHMARK_REPORT.md)
- [API Specification](API_SPECIFICATION.md)
- [Phase 3 Design](FRONTEND_PHASE3_DESIGN.md)
- [Sharia Compliance](SHARIA_COMPLIANCE.md)
