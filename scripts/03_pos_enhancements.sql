-- ============================================================================
-- 03_pos_enhancements.sql: POS Order History, Void/Refund & QRIS Enhancements
-- Adds retail/bakery columns to transactions and store_settings table
-- ============================================================================

-- Function to apply enhancements to a tenant schema
CREATE OR REPLACE FUNCTION public.apply_pos_enhancements(schema_name TEXT, tenant_name TEXT)
RETURNS VOID AS $$
BEGIN
    -- 1. Add bakery & retail operational columns to transactions
    EXECUTE format('
        ALTER TABLE %I.transactions
            ADD COLUMN IF NOT EXISTS customer_name VARCHAR(127),
            ADD COLUMN IF NOT EXISTS notes TEXT,
            ADD COLUMN IF NOT EXISTS cash_tendered NUMERIC(15, 2) DEFAULT 0,
            ADD COLUMN IF NOT EXISTS change_amount NUMERIC(15, 2) DEFAULT 0,
            ADD COLUMN IF NOT EXISTS payment_reference VARCHAR(127),
            ADD COLUMN IF NOT EXISTS void_reason VARCHAR(255),
            ADD COLUMN IF NOT EXISTS voided_at TIMESTAMPTZ,
            ADD COLUMN IF NOT EXISTS voided_by UUID;
    ', schema_name);

    -- Index on transactions search and void status
    EXECUTE format('
        CREATE INDEX IF NOT EXISTS idx_%s_transactions_status ON %I.transactions(status);
        CREATE INDEX IF NOT EXISTS idx_%s_transactions_payment ON %I.transactions(payment_method);
    ', replace(schema_name, 'tenant_', ''), schema_name, replace(schema_name, 'tenant_', ''), schema_name);

    -- 2. Create store_settings table for tenant-scoped configuration (e.g. QRIS payload)
    EXECUTE format('
        CREATE TABLE IF NOT EXISTS %I.store_settings (
            key VARCHAR(63) PRIMARY KEY,
            value JSONB NOT NULL,
            updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        );
    ', schema_name);

    -- 3. Seed default QRIS configuration if not exists
    EXECUTE format('
        INSERT INTO %I.store_settings (key, value)
        VALUES (
            ''qris'',
            json_build_object(
                ''merchant_name'', %L,
                ''nmid'', ''ID1020030040050'',
                ''qr_string'', ''00020101021126580014ID.LINKAJA.WWW011893600914300000222202151234567890123450303UMI51440014ID.CO.QRIS.WWW0215ID10200300400500303UMI5204549953033605802ID5924'' || %L || ''6010JAKARTA SE61051234062070703A0163041D2B'',
                ''qr_image_url'', ''https://api.qrserver.com/v1/create-qr-code/?size=300x300&data=00020101021126580014ID.LINKAJA.WWW011893600914300000222202151234567890123450303UMI51440014ID.CO.QRIS.WWW0215ID10200300400500303UMI5204549953033605802ID5924''
            )
        )
        ON CONFLICT (key) DO NOTHING;
    ', schema_name, tenant_name, upper(tenant_name));

    -- 4. Update ledger_entries check constraint to allow POS_VOID
    EXECUTE format('
        ALTER TABLE %I.ledger_entries DROP CONSTRAINT IF EXISTS ledger_entries_source_document_type_check;
        ALTER TABLE %I.ledger_entries ADD CONSTRAINT ledger_entries_source_document_type_check 
            CHECK (source_document_type IN (''POS_SALE'', ''POS_VOID'', ''GOODS_RECEIPT'', ''MANUAL_ADJUSTMENT'', ''ZAKAT_DISBURSEMENT''));
    ', schema_name, schema_name);
END;
$$ LANGUAGE plpgsql;

-- Apply to active tenant schemas
SELECT public.apply_pos_enhancements('tenant_al_barakah_mart', 'Al Barakah Bakery & Mart');
SELECT public.apply_pos_enhancements('tenant_darussalam_store', 'Darussalam Bakery & Store');

-- Cleanup migration function
DROP FUNCTION public.apply_pos_enhancements(TEXT, TEXT);
