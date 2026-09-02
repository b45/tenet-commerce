-- ============================================================================
-- 04_bakery_inventory_crud.sql: Bakery Inventory Management & Adjustments
-- Creates inventory_adjustments audit table and seeds bakery catalog
-- ============================================================================

CREATE OR REPLACE FUNCTION public.apply_bakery_inventory_schema(schema_name TEXT)
RETURNS VOID AS $$
BEGIN
    -- 1. Create inventory_adjustments audit table
    EXECUTE format('
        CREATE TABLE IF NOT EXISTS %I.inventory_adjustments (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            product_id UUID NOT NULL REFERENCES %I.products(id) ON DELETE CASCADE,
            adjustment_type VARCHAR(31) NOT NULL CHECK (adjustment_type IN (''ADD'', ''SUBTRACT'', ''SET'')),
            quantity_delta INTEGER NOT NULL,
            previous_quantity INTEGER NOT NULL,
            new_quantity INTEGER NOT NULL,
            reason VARCHAR(63) NOT NULL CHECK (reason IN (''DAMAGE'', ''EXPIRED'', ''AUDIT_CORRECTION'', ''RESTOCK'', ''OTHER'')),
            notes TEXT,
            adjusted_by UUID NOT NULL,
            ledger_entry_id UUID,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        );

        CREATE INDEX IF NOT EXISTS idx_%s_inv_adj_product ON %I.inventory_adjustments(product_id);
        CREATE INDEX IF NOT EXISTS idx_%s_inv_adj_created ON %I.inventory_adjustments(created_at DESC);
    ', schema_name, schema_name, replace(schema_name, 'tenant_', ''), schema_name, replace(schema_name, 'tenant_', ''), schema_name);

    -- 2. Seed Bakery Categories
    EXECUTE format('
        INSERT INTO %I.categories (id, name, code)
        VALUES 
            (''c0000000-0000-0000-0000-000000000010'', ''Kue Tart & Custom Cake'', ''CAT-CAKE''),
            (''c0000000-0000-0000-0000-000000000020'', ''Roti & Bolu'', ''CAT-BREAD''),
            (''c0000000-0000-0000-0000-000000000030'', ''Pastry & Croissant'', ''CAT-PASTRY''),
            (''c0000000-0000-0000-0000-000000000040'', ''Jajanan Pasar Halal'', ''CAT-SNACK'')
        ON CONFLICT (code) DO NOTHING;
    ', schema_name);

    -- 3. Seed Realistic Bakery Products
    EXECUTE format('
        INSERT INTO %I.products (id, category_id, sku, barcode, name, description, unit_price, cost_price, compliance_tags, is_active)
        VALUES
            (''10000000-0000-0000-0000-000000000011'', ''c0000000-0000-0000-0000-000000000010'', ''SKU-CAKE-BF20'', ''8992001000010'', ''Black Forest Cake 20cm'', ''Kue Black Forest premium dengan dark cherry dan serutan cokelat halal'', 185000.00, 120000.00, ''["HALAL_MUI"]'', TRUE),
            (''10000000-0000-0000-0000-000000000012'', ''c0000000-0000-0000-0000-000000000010'', ''SKU-CAKE-RV18'', ''8992001000020'', ''Red Velvet Cake 18cm'', ''Kue Red Velvet lembut dengan cream cheese frosting gurih manis'', 160000.00, 105000.00, ''["HALAL_MUI"]'', TRUE),
            (''10000000-0000-0000-0000-000000000013'', ''c0000000-0000-0000-0000-000000000020'', ''SKU-BREAD-BG01'', ''8992001000030'', ''Bolu Gulung Pandan Keju'', ''Bolu gulung aroma pandan asli suji dengan taburan parutan keju melimpah'', 45000.00, 28000.00, ''["HALAL_MUI"]'', TRUE),
            (''10000000-0000-0000-0000-000000000014'', ''c0000000-0000-0000-0000-000000000020'', ''SKU-BREAD-RS01'', ''8992001000040'', ''Roti Sisir Butter Premium'', ''Roti sisir mentega jadul lembut, manis gurih nagih'', 18000.00, 11000.00, ''["HALAL_MUI"]'', TRUE),
            (''10000000-0000-0000-0000-000000000015'', ''c0000000-0000-0000-0000-000000000030'', ''SKU-PASTRY-CA01'', ''8992001000050'', ''Croissant Almond Halal'', ''Croissant renyah berlapis dengan isian almond paste dan topping almond panggang'', 25000.00, 15000.00, ''["HALAL_MUI"]'', TRUE),
            (''10000000-0000-0000-0000-000000000016'', ''c0000000-0000-0000-0000-000000000040'', ''SKU-SNACK-LL01'', ''8992001000060'', ''Lapis Legit Prunes Slice'', ''Lapis legit rempah klasik dengan potongan buah prunes pilihan'', 28000.00, 18000.00, ''["HALAL_MUI"]'', TRUE)
        ON CONFLICT (sku) DO NOTHING;
    ', schema_name);

    -- 4. Seed Inventory Quantities & Reorder Thresholds
    EXECUTE format('
        INSERT INTO %I.inventory (product_id, stock_quantity, reorder_threshold, warehouse_location)
        VALUES
            (''10000000-0000-0000-0000-000000000011'', 10, 3, ''DISPLAY_CHILLER''),
            (''10000000-0000-0000-0000-000000000012'', 8, 2, ''DISPLAY_CHILLER''),
            (''10000000-0000-0000-0000-000000000013'', 25, 5, ''BAKERY_RACK_A''),
            (''10000000-0000-0000-0000-000000000014'', 40, 10, ''BAKERY_RACK_A''),
            (''10000000-0000-0000-0000-000000000015'', 15, 5, ''PASTRY_SHOWCASE''),
            (''10000000-0000-0000-0000-000000000016'', 20, 5, ''SNACK_SECTION'')
        ON CONFLICT (product_id) DO NOTHING;
    ', schema_name);
END;
$$ LANGUAGE plpgsql;

-- Apply to active tenant schemas
SELECT public.apply_bakery_inventory_schema('tenant_al_barakah_mart');
SELECT public.apply_bakery_inventory_schema('tenant_darussalam_store');

-- Cleanup migration function
DROP FUNCTION public.apply_bakery_inventory_schema(TEXT);
