# Database bootstrap source

`init_dev_db.sql` is the canonical baseline schema and development fixture for this repository. It is the only SQL script run by Docker Compose, `make db-reset`, and the Testcontainers integration suite.

## Deterministic Demo Fixtures (TPC-030)

The database baseline provisions reproducible, deterministic fixtures for golden journey demonstrations and offline-first E2E verification:
- **Tenant A (`al-barakah-mart` / `tenant_al_barakah_mart`)**:
  - Demo Product: `SKU-DEMO-01` (ID: `10000000-0000-0000-0000-000000000099`, unit price: IDR 15,000, cost price: IDR 10,000, threshold: 10, initial opening stock: 0).
  - Demo Valid Supplier: `SUP-DEMO-VALID-01` (ID: `a1000000-0000-0000-0000-000000000001`, BPJPH certificate `CERT-DEMO-HALAL-2099` valid until 2099-12-31).
  - Demo Expired Supplier: `SUP-DEMO-EXP-01` (ID: `a1000000-0000-0000-0000-000000000002`, expired certificate `CERT-DEMO-HALAL-EXPIRED`).
- **Tenant B (`darussalam-store` / `tenant_darussalam_store`)**:
  - Strictly isolated supplier: `SUP-DS-DEMO-01` (ID: `b1000000-0000-0000-0000-000000000001`, certificate `CERT-DS-DEMO-2099`).
  - Zero leakage across tenant boundaries.

## Safe Database Reset Guard

The reset command (`make db-reset` or `scripts/reset_dev_db.sh`) requires explicit opt-in confirmation (`CONFIRM_DEMO_RESET=true` or interactive prompt) and validates that the target instance is strictly a local/demo instance (`tenet_commerce` / `tenet_postgres`), preventing accidental erasure of production or non-demo environments.

The numbered SQL files remain as historical implementation artifacts. They must not be added to a bootstrap command: their effects have already been consolidated into the canonical baseline. New schema changes must be introduced as versioned migrations as part of the Phase 2 hardening work; do not copy schema fragments into test fixtures.
