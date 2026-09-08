\set ON_ERROR_STOP on
-- Run with psql -v tenant_schema=<trusted registry schema> -f this-file.
BEGIN TRANSACTION ISOLATION LEVEL REPEATABLE READ READ ONLY;
SELECT schema_name AS verified_schema FROM public.tenants
WHERE schema_name = :'tenant_schema' AND status = 'ACTIVE' \gset
SET LOCAL search_path TO :"verified_schema";

-- Expected: no rows. Includes journals with fewer than two lines.
SELECT e.id, count(l.id) AS line_count,
       coalesce(sum(l.debit_amount), 0) AS debit,
       coalesce(sum(l.credit_amount), 0) AS credit
FROM ledger_entries e LEFT JOIN ledger_entry_lines l ON l.ledger_entry_id = e.id
GROUP BY e.id
HAVING count(l.id) < 2 OR coalesce(sum(l.debit_amount), 0) <= 0
    OR coalesce(sum(l.debit_amount), 0) <> coalesce(sum(l.credit_amount), 0);

-- Expected: no rows after opening cutover and all subsequent movements.
SELECT i.product_id, i.stock_quantity, coalesce(sum(m.quantity_delta), 0) AS movement_balance
FROM inventory i LEFT JOIN stock_movements m ON m.product_id = i.product_id
GROUP BY i.product_id, i.stock_quantity
HAVING i.stock_quantity < 0 OR i.stock_quantity <> coalesce(sum(m.quantity_delta), 0);
ROLLBACK;
