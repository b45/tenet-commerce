# ADR 006: Inventory history and inline receipt QC

Status: implemented policy, audited at `203257a`.

[Stock movements](../../scripts/init_dev_db.sql) are append-only and have unique source identities. Opening balances establish a cutover baseline; subsequent deltas reconcile to inventory. Reconstructing fictitious pre-cutover movements would invent history, so the opening entry explicitly marks the boundary. Migration reruns must not duplicate it. Corrections use new movements, not edits to posted history.

[Receipt service](../../backend/internal/supplychain/service.go) records inline QC: delivered equals accepted plus rejected; rejection requires a reason. Accepted quantities determine inventory and journal valuation. Recording delivered quantity as available stock would include rejected goods. Deferring QC would require a separate quarantine/release model, which this flow does not implement.

This is repository policy and is not a verified quotation from an external ERP specification. Certificate checks are re-evaluated at receipt time under [compliance policy](../../backend/internal/supplychain/compliance.go). Historical decisions are not rewritten when certificates later change.

Evidence: [inventory tests](../../backend/internal/inventory/inventory_test.go), [operational tests](../../backend/integration/supplychain_operational_test.go), and [compliance tests](../../backend/integration/compliance_validity_test.go). Reconciliation does not substitute for concurrency and rollback tests.
