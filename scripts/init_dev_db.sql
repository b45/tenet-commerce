-- ==============================================================================
-- Tenet Commerce: Local Development Database Seed Script
-- Matches specifications in docs/DATABASE_SCHEMA.md
-- ==============================================================================

-- 1. Enable UUID extension in public schema
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 2. Create Global Tenants Registry in public schema
CREATE TABLE IF NOT EXISTS public.tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug VARCHAR(63) NOT NULL UNIQUE,
    company_name VARCHAR(255) NOT NULL,
    schema_name VARCHAR(63) NOT NULL UNIQUE,
    status VARCHAR(31) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'SUSPENDED', 'TERMINATED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. Seed Sample Tenants
INSERT INTO public.tenants (id, slug, company_name, schema_name, status)
VALUES 
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'al-barakah-mart', 'PT Al Barakah Mart Syariah', 'tenant_al_barakah_mart', 'ACTIVE'),
    ('b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 'darussalam-store', 'CV Darussalam Ritel', 'tenant_darussalam_store', 'ACTIVE'),
    ('c2eebc99-9c0b-4ef8-bb6d-6bb9bd380a33', 'suspended-mart', 'Toko Suspended Non-Aktif', 'tenant_suspended_mart', 'SUSPENDED')
ON CONFLICT (slug) DO NOTHING;

-- 3.1 Create Global Users & Authentication Table in public schema
CREATE TABLE IF NOT EXISTS public.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE RESTRICT,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    role VARCHAR(63) NOT NULL CHECK (role IN ('CASHIER', 'MANAGER', 'COMPLIANCE_OFFICER', 'FINANCIAL_ADMIN', 'SUPER_ADMIN')),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_tenant_email ON public.users(tenant_id, email);

-- Seed Sample Users for Tenant A (al-barakah-mart)
-- Password for all test accounts: "Password123!"
-- Bcrypt hash: $2a$10$ccdY3IxyNJFUGpEzYG4F3OwGsEXNZa4NX1F4/G.FP.QCty.grj29y
INSERT INTO public.users (id, tenant_id, email, password_hash, full_name, role, is_active)
VALUES 
    ('11111111-1111-1111-1111-111111111111', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'cashier1@albarakah.com', '$2a$10$ccdY3IxyNJFUGpEzYG4F3OwGsEXNZa4NX1F4/G.FP.QCty.grj29y', 'Ahmad Fauzi (Cashier)', 'CASHIER', TRUE),
    ('22222222-2222-2222-2222-222222222222', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'manager1@albarakah.com', '$2a$10$ccdY3IxyNJFUGpEzYG4F3OwGsEXNZa4NX1F4/G.FP.QCty.grj29y', 'Siti Rahma (Store Manager)', 'MANAGER', TRUE),
    ('33333333-3333-3333-3333-333333333333', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'compliance1@albarakah.com', '$2a$10$ccdY3IxyNJFUGpEzYG4F3OwGsEXNZa4NX1F4/G.FP.QCty.grj29y', 'Ust. Zulkifli (Halal Officer)', 'COMPLIANCE_OFFICER', TRUE),
    ('44444444-4444-4444-4444-444444444444', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'finance1@albarakah.com', '$2a$10$ccdY3IxyNJFUGpEzYG4F3OwGsEXNZa4NX1F4/G.FP.QCty.grj29y', 'H. Mansur (Financial Admin)', 'FINANCIAL_ADMIN', TRUE),
    ('55555555-5555-5555-5555-555555555551', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'superadmin@albarakah.com', '$2a$10$ccdY3IxyNJFUGpEzYG4F3OwGsEXNZa4NX1F4/G.FP.QCty.grj29y', 'Super Administrator (Al-Barakah)', 'SUPER_ADMIN', TRUE),
    ('55555555-5555-5555-5555-555555555555', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'superadmin@tenet.internal', '$2a$10$ccdY3IxyNJFUGpEzYG4F3OwGsEXNZa4NX1F4/G.FP.QCty.grj29y', 'Super Administrator', 'SUPER_ADMIN', TRUE),
    -- Seed Sample User for Tenant B (darussalam-store)
    ('66666666-6666-6666-6666-666666666666', 'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 'manager1@darussalam.com', '$2a$10$ccdY3IxyNJFUGpEzYG4F3OwGsEXNZa4NX1F4/G.FP.QCty.grj29y', 'Budi Santoso (Store Manager)', 'MANAGER', TRUE),
    ('77777777-7777-7777-7777-777777777777', 'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', 'superadmin@darussalam.com', '$2a$10$ccdY3IxyNJFUGpEzYG4F3OwGsEXNZa4NX1F4/G.FP.QCty.grj29y', 'Super Administrator (Darussalam)', 'SUPER_ADMIN', TRUE)
ON CONFLICT (email) DO NOTHING;

-- 4. Clean & Re-create Isolated Tenant Schemas for Dev
CREATE SCHEMA IF NOT EXISTS tenant_al_barakah_mart;
CREATE SCHEMA IF NOT EXISTS tenant_darussalam_store;

-- ==============================================================================
-- 5. SCHEMA SETUP: tenant_al_barakah_mart
-- ==============================================================================

-- 5.0 Tenant Configuration
CREATE TABLE IF NOT EXISTS tenant_al_barakah_mart.tenant_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_key VARCHAR(127) NOT NULL UNIQUE,
    config_value JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO tenant_al_barakah_mart.tenant_config (config_key, config_value)
VALUES ('compliance', '{"strict_compliance_mode": true, "required_compliance": ["HALAL_MUI"]}')
ON CONFLICT (config_key) DO UPDATE SET config_value = EXCLUDED.config_value;


-- 5.1 Categories Table
CREATE TABLE IF NOT EXISTS tenant_al_barakah_mart.categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(127) NOT NULL,
    code VARCHAR(31) NOT NULL UNIQUE,
    parent_id UUID REFERENCES tenant_al_barakah_mart.categories(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 5.2 Products Table
CREATE TABLE IF NOT EXISTS tenant_al_barakah_mart.products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id UUID REFERENCES tenant_al_barakah_mart.categories(id) ON DELETE SET NULL,
    sku VARCHAR(63) NOT NULL UNIQUE,
    barcode VARCHAR(127) UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    unit_price NUMERIC(15, 2) NOT NULL CHECK (unit_price >= 0),
    cost_price NUMERIC(15, 2) NOT NULL DEFAULT 0 CHECK (cost_price >= 0),
    compliance_tags JSONB,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Ensure missing columns exist if table was partially created
ALTER TABLE tenant_al_barakah_mart.products ADD COLUMN IF NOT EXISTS category_id UUID REFERENCES tenant_al_barakah_mart.categories(id) ON DELETE SET NULL;
ALTER TABLE tenant_al_barakah_mart.products ADD COLUMN IF NOT EXISTS barcode VARCHAR(127) UNIQUE;
ALTER TABLE tenant_al_barakah_mart.products ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE tenant_al_barakah_mart.products ADD COLUMN IF NOT EXISTS cost_price NUMERIC(15, 2) NOT NULL DEFAULT 0;
ALTER TABLE tenant_al_barakah_mart.products ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE tenant_al_barakah_mart.products ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- 5.3 Inventory Table
CREATE TABLE IF NOT EXISTS tenant_al_barakah_mart.inventory (
    product_id UUID PRIMARY KEY REFERENCES tenant_al_barakah_mart.products(id) ON DELETE CASCADE,
    stock_quantity INTEGER NOT NULL DEFAULT 0 CHECK (stock_quantity >= 0),
    reorder_threshold INTEGER NOT NULL DEFAULT 10,
    warehouse_location VARCHAR(127) DEFAULT 'MAIN_STORE',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 5.4 Transactions Table
CREATE TABLE IF NOT EXISTS tenant_al_barakah_mart.transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_number VARCHAR(63) NOT NULL UNIQUE,
    idempotency_key VARCHAR(127) NOT NULL UNIQUE,
    cashier_id UUID NOT NULL,
    subtotal_amount NUMERIC(15, 2) NOT NULL CHECK (subtotal_amount >= 0),
    tax_amount NUMERIC(15, 2) NOT NULL DEFAULT 0 CHECK (tax_amount >= 0),
    discount_amount NUMERIC(15, 2) NOT NULL DEFAULT 0 CHECK (discount_amount >= 0),
    total_amount NUMERIC(15, 2) NOT NULL CHECK (total_amount >= 0),
    payment_method VARCHAR(31) NOT NULL CHECK (payment_method IN ('CASH', 'SIMULATED_CARD', 'QRIS')),
    status VARCHAR(31) NOT NULL DEFAULT 'COMPLETED' CHECK (status IN ('PENDING', 'COMPLETED', 'VOIDED')),
    customer_name VARCHAR(127),
    notes TEXT,
    cash_tendered NUMERIC(15, 2) DEFAULT 0,
    change_amount NUMERIC(15, 2) DEFAULT 0,
    payment_reference VARCHAR(127),
    void_reason VARCHAR(255),
    voided_at TIMESTAMPTZ,
    voided_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_abm_transactions_created ON tenant_al_barakah_mart.transactions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_abm_transactions_idempotency ON tenant_al_barakah_mart.transactions(idempotency_key);
CREATE INDEX IF NOT EXISTS idx_abm_transactions_status ON tenant_al_barakah_mart.transactions(status);
CREATE INDEX IF NOT EXISTS idx_abm_transactions_payment ON tenant_al_barakah_mart.transactions(payment_method);

CREATE TABLE IF NOT EXISTS tenant_al_barakah_mart.store_settings (
    key VARCHAR(63) PRIMARY KEY,
    value JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO tenant_al_barakah_mart.store_settings (key, value)
VALUES (
    'qris',
    '{"merchant_name": "Al Barakah Bakery & Mart", "nmid": "ID1020030040050", "qr_string": "00020101021126580014ID.LINKAJA.WWW011893600914300000222202151234567890123450303UMI51440014ID.CO.QRIS.WWW0215ID10200300400500303UMI5204549953033605802ID5924AL BARAKAH BAKERY & MART6010JAKARTA SE61051234062070703A0163041D2B", "qr_image_url": "https://api.qrserver.com/v1/create-qr-code/?size=300x300&data=00020101021126580014ID.LINKAJA.WWW011893600914300000222202151234567890123450303UMI51440014ID.CO.QRIS.WWW0215ID10200300400500303UMI5204549953033605802ID5924"}'::jsonb
) ON CONFLICT (key) DO NOTHING;

CREATE TABLE IF NOT EXISTS tenant_al_barakah_mart.inventory_adjustments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES tenant_al_barakah_mart.products(id) ON DELETE CASCADE,
    adjustment_type VARCHAR(31) NOT NULL CHECK (adjustment_type IN ('ADD', 'SUBTRACT', 'SET')),
    quantity_delta INTEGER NOT NULL,
    previous_quantity INTEGER NOT NULL,
    new_quantity INTEGER NOT NULL,
    reason VARCHAR(63) NOT NULL CHECK (reason IN ('DAMAGE', 'EXPIRED', 'AUDIT_CORRECTION', 'RESTOCK', 'OTHER')),
    notes TEXT,
    adjusted_by UUID NOT NULL,
    ledger_entry_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_abm_inv_adj_product ON tenant_al_barakah_mart.inventory_adjustments(product_id);
CREATE INDEX IF NOT EXISTS idx_abm_inv_adj_created ON tenant_al_barakah_mart.inventory_adjustments(created_at DESC);

-- 5.5 Transaction Items Table
CREATE TABLE IF NOT EXISTS tenant_al_barakah_mart.transaction_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID NOT NULL REFERENCES tenant_al_barakah_mart.transactions(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES tenant_al_barakah_mart.products(id) ON DELETE RESTRICT,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price NUMERIC(15, 2) NOT NULL CHECK (unit_price >= 0),
    cost_price NUMERIC(15, 2) NOT NULL DEFAULT 0 CHECK (cost_price >= 0),
    subtotal NUMERIC(15, 2) NOT NULL CHECK (subtotal >= 0)
);


-- Compliance-Aware Supply Chain Management
CREATE TABLE IF NOT EXISTS tenant_al_barakah_mart.suppliers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(63) NOT NULL UNIQUE,
    company_name VARCHAR(255) NOT NULL,
    contact_person VARCHAR(127),
    contact_email VARCHAR(255),
    contact_phone VARCHAR(63),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tenant_al_barakah_mart.compliance_certificates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    supplier_id UUID NOT NULL REFERENCES tenant_al_barakah_mart.suppliers(id) ON DELETE RESTRICT,
    cert_type VARCHAR(63) NOT NULL, 
    certificate_number VARCHAR(127) NOT NULL UNIQUE,
    issuing_authority VARCHAR(127) NOT NULL, 
    scope TEXT NOT NULL,
    valid_from DATE NOT NULL,
    expiry_date DATE NOT NULL,
    document_url VARCHAR(511),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_compliance_cert_expiry_tenant_al_barakah_mart ON tenant_al_barakah_mart.compliance_certificates(expiry_date);

CREATE TABLE IF NOT EXISTS tenant_al_barakah_mart.purchase_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    po_number VARCHAR(63) NOT NULL UNIQUE,
    supplier_id UUID NOT NULL REFERENCES tenant_al_barakah_mart.suppliers(id) ON DELETE RESTRICT,
    compliance_cert_id UUID REFERENCES tenant_al_barakah_mart.compliance_certificates(id) ON DELETE RESTRICT,
    total_amount NUMERIC(15, 2) NOT NULL CHECK (total_amount >= 0),
    status VARCHAR(31) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'ISSUED', 'PARTIALLY_RECEIVED', 'RECEIVED', 'CANCELLED')),
    issued_date DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tenant_al_barakah_mart.purchase_order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    purchase_order_id UUID NOT NULL REFERENCES tenant_al_barakah_mart.purchase_orders(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES tenant_al_barakah_mart.products(id) ON DELETE RESTRICT,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_cost NUMERIC(15, 2) NOT NULL CHECK (unit_cost >= 0),
    subtotal NUMERIC(15, 2) NOT NULL CHECK (subtotal >= 0)
);

CREATE TABLE IF NOT EXISTS tenant_al_barakah_mart.goods_receipts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    gr_number VARCHAR(63) NOT NULL UNIQUE,
    idempotency_key VARCHAR(255) NOT NULL UNIQUE,
    purchase_order_id UUID NOT NULL REFERENCES tenant_al_barakah_mart.purchase_orders(id) ON DELETE RESTRICT,
    received_by UUID NOT NULL,
    received_date DATE NOT NULL DEFAULT CURRENT_DATE,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tenant_al_barakah_mart.goods_receipt_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    goods_receipt_id UUID NOT NULL REFERENCES tenant_al_barakah_mart.goods_receipts(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES tenant_al_barakah_mart.products(id) ON DELETE RESTRICT,
    received_quantity INTEGER NOT NULL CHECK (received_quantity >= 0)
);

-- 5.5 Ledger Engine

-- Sharia Double-Entry General Ledger
CREATE TABLE IF NOT EXISTS tenant_al_barakah_mart.ledger_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(31) NOT NULL UNIQUE,
    name VARCHAR(127) NOT NULL,
    account_type VARCHAR(31) NOT NULL CHECK (account_type IN ('ASSET', 'LIABILITY', 'EQUITY', 'REVENUE', 'EXPENSE')),
    is_zakat_eligible BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tenant_al_barakah_mart.ledger_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_number VARCHAR(63) NOT NULL UNIQUE,
    entry_date DATE NOT NULL DEFAULT CURRENT_DATE,
    source_document_type VARCHAR(63) NOT NULL CHECK (source_document_type IN ('POS_SALE', 'POS_VOID', 'GOODS_RECEIPT', 'MANUAL_ADJUSTMENT', 'ZAKAT_DISBURSEMENT')),
    source_document_id UUID,
    memo TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tenant_al_barakah_mart.ledger_entry_lines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ledger_entry_id UUID NOT NULL REFERENCES tenant_al_barakah_mart.ledger_entries(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES tenant_al_barakah_mart.ledger_accounts(id) ON DELETE RESTRICT,
    debit_amount NUMERIC(15, 2) NOT NULL DEFAULT 0 CHECK (debit_amount >= 0),
    credit_amount NUMERIC(15, 2) NOT NULL DEFAULT 0 CHECK (credit_amount >= 0),
    CONSTRAINT chk_debit_or_credit CHECK (
        (debit_amount > 0 AND credit_amount = 0) OR 
        (credit_amount > 0 AND debit_amount = 0)
    )
);

CREATE INDEX IF NOT EXISTS idx_ledger_lines_account_tenant_al_barakah_mart ON tenant_al_barakah_mart.ledger_entry_lines(account_id);
CREATE INDEX IF NOT EXISTS idx_ledger_entries_date_tenant_al_barakah_mart ON tenant_al_barakah_mart.ledger_entries(entry_date);

-- Ledger Balance Invariant Trigger
CREATE OR REPLACE FUNCTION tenant_al_barakah_mart.verify_ledger_entry_balance()
RETURNS TRIGGER AS $$
DECLARE
    total_debit NUMERIC(15, 2);
    total_credit NUMERIC(15, 2);
BEGIN
    SELECT COALESCE(SUM(debit_amount), 0), COALESCE(SUM(credit_amount), 0)
    INTO total_debit, total_credit
    FROM tenant_al_barakah_mart.ledger_entry_lines
    WHERE ledger_entry_id = NEW.ledger_entry_id;

    IF total_debit <> total_credit THEN
        RAISE EXCEPTION 'Sharia Ledger Invariant Violation: Total Debits (%) must equal Total Credits (%) for Entry %', 
            total_debit, total_credit, NEW.ledger_entry_id;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_verify_ledger_balance ON tenant_al_barakah_mart.ledger_entry_lines;
CREATE CONSTRAINT TRIGGER trg_verify_ledger_balance
AFTER INSERT OR UPDATE ON tenant_al_barakah_mart.ledger_entry_lines
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION tenant_al_barakah_mart.verify_ledger_entry_balance();

-- Seed Chart of Accounts
INSERT INTO tenant_al_barakah_mart.ledger_accounts (code, name, account_type, is_zakat_eligible) VALUES
    ('1010', 'Cash on Hand', 'ASSET', TRUE),
    ('1020', 'Bank Operating Account', 'ASSET', TRUE),
    ('1030', 'Merchandise Inventory', 'ASSET', TRUE),
    ('1040', 'Trade Accounts Receivable', 'ASSET', TRUE),
    ('2010', 'Trade Accounts Payable', 'LIABILITY', TRUE),
    ('3010', 'Owner''s Equity', 'EQUITY', FALSE),
    ('4010', 'Sales Revenue', 'REVENUE', FALSE),
    ('5010', 'Cost of Goods Sold', 'EXPENSE', FALSE),
    ('5020', 'Inventory Shrinkage & Loss', 'EXPENSE', FALSE)
ON CONFLICT (code) DO NOTHING;

-- 5.6 Seed Categories for tenant_al_barakah_mart
INSERT INTO tenant_al_barakah_mart.categories (id, name, code)
VALUES
    ('c0000000-0000-0000-0000-000000000001', 'Fresh Meat & Poultry', 'CAT-FRESH-MEAT'),
    ('c0000000-0000-0000-0000-000000000002', 'Beverages & Syrups', 'CAT-BEVERAGES'),
    ('c0000000-0000-0000-0000-000000000003', 'Pantry & Groceries', 'CAT-PANTRY'),
    ('c0000000-0000-0000-0000-000000000010', 'Kue Tart & Custom Cake', 'CAT-CAKE'),
    ('c0000000-0000-0000-0000-000000000020', 'Roti & Bolu', 'CAT-BREAD'),
    ('c0000000-0000-0000-0000-000000000030', 'Pastry & Croissant', 'CAT-PASTRY'),
    ('c0000000-0000-0000-0000-000000000040', 'Jajanan Pasar Halal', 'CAT-SNACK')
ON CONFLICT (code) DO NOTHING;

-- 5.7 Seed Products for tenant_al_barakah_mart
INSERT INTO tenant_al_barakah_mart.products (id, category_id, sku, barcode, name, description, unit_price, cost_price, compliance_tags, is_active)
VALUES
    ('10000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000001', 'SKU-BEEF-01', '8991001000011', 'Daging Sapi Halal Al-Barakah 500g', 'Daging sapi segar bersertifikat Halal MUI', 75000.00, 60000.00, '["HALAL_MUI"]', TRUE),
    ('10000000-0000-0000-0000-000000000002', 'c0000000-0000-0000-0000-000000000001', 'SKU-CHICKEN-01', '8991001000028', 'Ayam Potong Segar 1kg', 'Ayam potong higienis dan halal', 38000.00, 30000.00, '["HALAL_MUI"]', TRUE),
    ('10000000-0000-0000-0000-000000000003', 'c0000000-0000-0000-0000-000000000002', 'SKU-HONEY-01', '8991001000035', 'Madu Murni Al-Barakah 350ml', 'Madu hutan murni tanpa bahan pengawet', 65000.00, 48000.00, '["HALAL_MUI"]', TRUE),
    ('10000000-0000-0000-0000-000000000004', 'c0000000-0000-0000-0000-000000000003', 'SKU-OIL-01', '8991001000042', 'Minyak Goreng Kelapa Sawit 2L', 'Minyak goreng jernih berkualitas tinggi', 34000.00, 29000.00, '["HALAL_MUI"]', TRUE),
    ('10000000-0000-0000-0000-000000000005', 'c0000000-0000-0000-0000-000000000003', 'SKU-RICE-01', '8991001000059', 'Beras Ramos Organik 5kg', 'Beras putih pulen organik', 72000.00, 62000.00, '["ORGANIC"]', TRUE),
    ('10000000-0000-0000-0000-000000000011', 'c0000000-0000-0000-0000-000000000010', 'SKU-CAKE-BF20', '8992001000010', 'Black Forest Cake 20cm', 'Kue Black Forest premium dengan dark cherry dan serutan cokelat halal', 185000.00, 120000.00, '["HALAL_MUI"]', TRUE),
    ('10000000-0000-0000-0000-000000000012', 'c0000000-0000-0000-0000-000000000010', 'SKU-CAKE-RV18', '8992001000020', 'Red Velvet Cake 18cm', 'Kue Red Velvet lembut dengan cream cheese frosting gurih manis', 160000.00, 105000.00, '["HALAL_MUI"]', TRUE),
    ('10000000-0000-0000-0000-000000000013', 'c0000000-0000-0000-0000-000000000020', 'SKU-BREAD-BG01', '8992001000030', 'Bolu Gulung Pandan Keju', 'Bolu gulung aroma pandan asli suji dengan taburan parutan keju melimpah', 45000.00, 28000.00, '["HALAL_MUI"]', TRUE),
    ('10000000-0000-0000-0000-000000000014', 'c0000000-0000-0000-0000-000000000020', 'SKU-BREAD-RS01', '8992001000040', 'Roti Sisir Butter Premium', 'Roti sisir mentega jadul lembut, manis gurih nagih', 18000.00, 11000.00, '["HALAL_MUI"]', TRUE),
    ('10000000-0000-0000-0000-000000000015', 'c0000000-0000-0000-0000-000000000030', 'SKU-PASTRY-CA01', '8992001000050', 'Croissant Almond Halal', 'Croissant renyah berlapis dengan isian almond paste dan topping almond panggang', 25000.00, 15000.00, '["HALAL_MUI"]', TRUE),
    ('10000000-0000-0000-0000-000000000016', 'c0000000-0000-0000-0000-000000000040', 'SKU-SNACK-LL01', '8992001000060', 'Lapis Legit Prunes Slice', 'Lapis legit rempah klasik dengan potongan buah prunes pilihan', 28000.00, 18000.00, '["HALAL_MUI"]', TRUE)
ON CONFLICT (sku) DO UPDATE SET 
    name = EXCLUDED.name,
    unit_price = EXCLUDED.unit_price,
    cost_price = EXCLUDED.cost_price,
    barcode = EXCLUDED.barcode,
    compliance_tags = EXCLUDED.compliance_tags;

-- 5.8 Seed Inventory Stock for tenant_al_barakah_mart
INSERT INTO tenant_al_barakah_mart.inventory (product_id, stock_quantity, reorder_threshold, warehouse_location)
SELECT id, 50, 10, 'MAIN_STORE' FROM tenant_al_barakah_mart.products
ON CONFLICT (product_id) DO UPDATE SET stock_quantity = EXCLUDED.stock_quantity;

-- ==============================================================================
-- 6. SCHEMA SETUP: tenant_darussalam_store
-- ==============================================================================

-- 6.0 Tenant Configuration
CREATE TABLE IF NOT EXISTS tenant_darussalam_store.tenant_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_key VARCHAR(127) NOT NULL UNIQUE,
    config_value JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO tenant_darussalam_store.tenant_config (config_key, config_value)
VALUES ('compliance', '{"strict_compliance_mode": false, "required_compliance": []}')
ON CONFLICT (config_key) DO UPDATE SET config_value = EXCLUDED.config_value;


-- 6.1 Categories Table
CREATE TABLE IF NOT EXISTS tenant_darussalam_store.categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(127) NOT NULL,
    code VARCHAR(31) NOT NULL UNIQUE,
    parent_id UUID REFERENCES tenant_darussalam_store.categories(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 6.2 Products Table
CREATE TABLE IF NOT EXISTS tenant_darussalam_store.products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id UUID REFERENCES tenant_darussalam_store.categories(id) ON DELETE SET NULL,
    sku VARCHAR(63) NOT NULL UNIQUE,
    barcode VARCHAR(127) UNIQUE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    unit_price NUMERIC(15, 2) NOT NULL CHECK (unit_price >= 0),
    cost_price NUMERIC(15, 2) NOT NULL DEFAULT 0 CHECK (cost_price >= 0),
    compliance_tags JSONB,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE tenant_darussalam_store.products ADD COLUMN IF NOT EXISTS category_id UUID REFERENCES tenant_darussalam_store.categories(id) ON DELETE SET NULL;
ALTER TABLE tenant_darussalam_store.products ADD COLUMN IF NOT EXISTS barcode VARCHAR(127) UNIQUE;
ALTER TABLE tenant_darussalam_store.products ADD COLUMN IF NOT EXISTS description TEXT;
ALTER TABLE tenant_darussalam_store.products ADD COLUMN IF NOT EXISTS cost_price NUMERIC(15, 2) NOT NULL DEFAULT 0;
ALTER TABLE tenant_darussalam_store.products ADD COLUMN IF NOT EXISTS is_active BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE tenant_darussalam_store.products ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- 6.3 Inventory Table
CREATE TABLE IF NOT EXISTS tenant_darussalam_store.inventory (
    product_id UUID PRIMARY KEY REFERENCES tenant_darussalam_store.products(id) ON DELETE CASCADE,
    stock_quantity INTEGER NOT NULL DEFAULT 0 CHECK (stock_quantity >= 0),
    reorder_threshold INTEGER NOT NULL DEFAULT 10,
    warehouse_location VARCHAR(127) DEFAULT 'MAIN_STORE',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 6.4 Transactions Table
CREATE TABLE IF NOT EXISTS tenant_darussalam_store.transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_number VARCHAR(63) NOT NULL UNIQUE,
    idempotency_key VARCHAR(127) NOT NULL UNIQUE,
    cashier_id UUID NOT NULL,
    subtotal_amount NUMERIC(15, 2) NOT NULL CHECK (subtotal_amount >= 0),
    tax_amount NUMERIC(15, 2) NOT NULL DEFAULT 0 CHECK (tax_amount >= 0),
    discount_amount NUMERIC(15, 2) NOT NULL DEFAULT 0 CHECK (discount_amount >= 0),
    total_amount NUMERIC(15, 2) NOT NULL CHECK (total_amount >= 0),
    payment_method VARCHAR(31) NOT NULL CHECK (payment_method IN ('CASH', 'SIMULATED_CARD', 'QRIS')),
    status VARCHAR(31) NOT NULL DEFAULT 'COMPLETED' CHECK (status IN ('PENDING', 'COMPLETED', 'VOIDED')),
    customer_name VARCHAR(127),
    notes TEXT,
    cash_tendered NUMERIC(15, 2) DEFAULT 0,
    change_amount NUMERIC(15, 2) DEFAULT 0,
    payment_reference VARCHAR(127),
    void_reason VARCHAR(255),
    voided_at TIMESTAMPTZ,
    voided_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ds_transactions_created ON tenant_darussalam_store.transactions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ds_transactions_idempotency ON tenant_darussalam_store.transactions(idempotency_key);
CREATE INDEX IF NOT EXISTS idx_ds_transactions_status ON tenant_darussalam_store.transactions(status);
CREATE INDEX IF NOT EXISTS idx_ds_transactions_payment ON tenant_darussalam_store.transactions(payment_method);

CREATE TABLE IF NOT EXISTS tenant_darussalam_store.store_settings (
    key VARCHAR(63) PRIMARY KEY,
    value JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO tenant_darussalam_store.store_settings (key, value)
VALUES (
    'qris',
    '{"merchant_name": "Darussalam Bakery & Store", "nmid": "ID1020030040050", "qr_string": "00020101021126580014ID.LINKAJA.WWW011893600914300000222202151234567890123450303UMI51440014ID.CO.QRIS.WWW0215ID10200300400500303UMI5204549953033605802ID5924DARUSSALAM BAKERY & STORE6010JAKARTA SE61051234062070703A0163041D2B", "qr_image_url": "https://api.qrserver.com/v1/create-qr-code/?size=300x300&data=00020101021126580014ID.LINKAJA.WWW011893600914300000222202151234567890123450303UMI51440014ID.CO.QRIS.WWW0215ID10200300400500303UMI5204549953033605802ID5924"}'::jsonb
) ON CONFLICT (key) DO NOTHING;

CREATE TABLE IF NOT EXISTS tenant_darussalam_store.inventory_adjustments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES tenant_darussalam_store.products(id) ON DELETE CASCADE,
    adjustment_type VARCHAR(31) NOT NULL CHECK (adjustment_type IN ('ADD', 'SUBTRACT', 'SET')),
    quantity_delta INTEGER NOT NULL,
    previous_quantity INTEGER NOT NULL,
    new_quantity INTEGER NOT NULL,
    reason VARCHAR(63) NOT NULL CHECK (reason IN ('DAMAGE', 'EXPIRED', 'AUDIT_CORRECTION', 'RESTOCK', 'OTHER')),
    notes TEXT,
    adjusted_by UUID NOT NULL,
    ledger_entry_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ds_inv_adj_product ON tenant_darussalam_store.inventory_adjustments(product_id);
CREATE INDEX IF NOT EXISTS idx_ds_inv_adj_created ON tenant_darussalam_store.inventory_adjustments(created_at DESC);

-- 6.5 Transaction Items Table
CREATE TABLE IF NOT EXISTS tenant_darussalam_store.transaction_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID NOT NULL REFERENCES tenant_darussalam_store.transactions(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES tenant_darussalam_store.products(id) ON DELETE RESTRICT,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price NUMERIC(15, 2) NOT NULL CHECK (unit_price >= 0),
    cost_price NUMERIC(15, 2) NOT NULL DEFAULT 0 CHECK (cost_price >= 0),
    subtotal NUMERIC(15, 2) NOT NULL CHECK (subtotal >= 0)
);


-- Compliance-Aware Supply Chain Management
CREATE TABLE IF NOT EXISTS tenant_darussalam_store.suppliers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(63) NOT NULL UNIQUE,
    company_name VARCHAR(255) NOT NULL,
    contact_person VARCHAR(127),
    contact_email VARCHAR(255),
    contact_phone VARCHAR(63),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tenant_darussalam_store.compliance_certificates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    supplier_id UUID NOT NULL REFERENCES tenant_darussalam_store.suppliers(id) ON DELETE RESTRICT,
    cert_type VARCHAR(63) NOT NULL, 
    certificate_number VARCHAR(127) NOT NULL UNIQUE,
    issuing_authority VARCHAR(127) NOT NULL, 
    scope TEXT NOT NULL,
    valid_from DATE NOT NULL,
    expiry_date DATE NOT NULL,
    document_url VARCHAR(511),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_compliance_cert_expiry_tenant_darussalam_store ON tenant_darussalam_store.compliance_certificates(expiry_date);

CREATE TABLE IF NOT EXISTS tenant_darussalam_store.purchase_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    po_number VARCHAR(63) NOT NULL UNIQUE,
    supplier_id UUID NOT NULL REFERENCES tenant_darussalam_store.suppliers(id) ON DELETE RESTRICT,
    compliance_cert_id UUID REFERENCES tenant_darussalam_store.compliance_certificates(id) ON DELETE RESTRICT,
    total_amount NUMERIC(15, 2) NOT NULL CHECK (total_amount >= 0),
    status VARCHAR(31) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'ISSUED', 'PARTIALLY_RECEIVED', 'RECEIVED', 'CANCELLED')),
    issued_date DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tenant_darussalam_store.purchase_order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    purchase_order_id UUID NOT NULL REFERENCES tenant_darussalam_store.purchase_orders(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES tenant_darussalam_store.products(id) ON DELETE RESTRICT,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_cost NUMERIC(15, 2) NOT NULL CHECK (unit_cost >= 0),
    subtotal NUMERIC(15, 2) NOT NULL CHECK (subtotal >= 0)
);

CREATE TABLE IF NOT EXISTS tenant_darussalam_store.goods_receipts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    gr_number VARCHAR(63) NOT NULL UNIQUE,
    idempotency_key VARCHAR(255) NOT NULL UNIQUE,
    purchase_order_id UUID NOT NULL REFERENCES tenant_darussalam_store.purchase_orders(id) ON DELETE RESTRICT,
    received_by UUID NOT NULL,
    received_date DATE NOT NULL DEFAULT CURRENT_DATE,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tenant_darussalam_store.goods_receipt_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    goods_receipt_id UUID NOT NULL REFERENCES tenant_darussalam_store.goods_receipts(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES tenant_darussalam_store.products(id) ON DELETE RESTRICT,
    received_quantity INTEGER NOT NULL CHECK (received_quantity >= 0)
);

-- 6.6 Seed Products for tenant_darussalam_store
INSERT INTO tenant_darussalam_store.products (id, sku, barcode, name, unit_price, cost_price, compliance_tags, is_active)
VALUES 
    ('20000000-0000-0000-0000-000000000001', 'SKU-DATES-01', '8992002000018', 'Kurma Ajwa Madinah Darussalam 1kg', 190000.00, 150000.00, '[]', TRUE),
    ('20000000-0000-0000-0000-000000000002', 'SKU-ZAMZAM-01', '8992002000025', 'Air Zamzam Murni 5L', 350000.00, 280000.00, '[]', TRUE)
ON CONFLICT (sku) DO UPDATE SET 
    name = EXCLUDED.name,
    unit_price = EXCLUDED.unit_price,
    cost_price = EXCLUDED.cost_price,
    barcode = EXCLUDED.barcode,
    compliance_tags = EXCLUDED.compliance_tags;

-- 6.7 Seed Inventory Stock for tenant_darussalam_store
INSERT INTO tenant_darussalam_store.inventory (product_id, stock_quantity, reorder_threshold, warehouse_location)
SELECT id, 30, 5, 'MAIN_STORE' FROM tenant_darussalam_store.products
ON CONFLICT (product_id) DO UPDATE SET stock_quantity = EXCLUDED.stock_quantity;

-- 6.8 Ledger Engine

-- Sharia Double-Entry General Ledger
CREATE TABLE IF NOT EXISTS tenant_darussalam_store.ledger_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(31) NOT NULL UNIQUE,
    name VARCHAR(127) NOT NULL,
    account_type VARCHAR(31) NOT NULL CHECK (account_type IN ('ASSET', 'LIABILITY', 'EQUITY', 'REVENUE', 'EXPENSE')),
    is_zakat_eligible BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tenant_darussalam_store.ledger_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_number VARCHAR(63) NOT NULL UNIQUE,
    entry_date DATE NOT NULL DEFAULT CURRENT_DATE,
    source_document_type VARCHAR(63) NOT NULL CHECK (source_document_type IN ('POS_SALE', 'POS_VOID', 'GOODS_RECEIPT', 'MANUAL_ADJUSTMENT', 'ZAKAT_DISBURSEMENT')),
    source_document_id UUID,
    memo TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tenant_darussalam_store.ledger_entry_lines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ledger_entry_id UUID NOT NULL REFERENCES tenant_darussalam_store.ledger_entries(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES tenant_darussalam_store.ledger_accounts(id) ON DELETE RESTRICT,
    debit_amount NUMERIC(15, 2) NOT NULL DEFAULT 0 CHECK (debit_amount >= 0),
    credit_amount NUMERIC(15, 2) NOT NULL DEFAULT 0 CHECK (credit_amount >= 0),
    CONSTRAINT chk_debit_or_credit CHECK (
        (debit_amount > 0 AND credit_amount = 0) OR 
        (credit_amount > 0 AND debit_amount = 0)
    )
);

CREATE INDEX IF NOT EXISTS idx_ledger_lines_account_tenant_darussalam_store ON tenant_darussalam_store.ledger_entry_lines(account_id);
CREATE INDEX IF NOT EXISTS idx_ledger_entries_date_tenant_darussalam_store ON tenant_darussalam_store.ledger_entries(entry_date);

-- Ledger Balance Invariant Trigger
CREATE OR REPLACE FUNCTION tenant_darussalam_store.verify_ledger_entry_balance()
RETURNS TRIGGER AS $$
DECLARE
    total_debit NUMERIC(15, 2);
    total_credit NUMERIC(15, 2);
BEGIN
    SELECT COALESCE(SUM(debit_amount), 0), COALESCE(SUM(credit_amount), 0)
    INTO total_debit, total_credit
    FROM tenant_darussalam_store.ledger_entry_lines
    WHERE ledger_entry_id = NEW.ledger_entry_id;

    IF total_debit <> total_credit THEN
        RAISE EXCEPTION 'Sharia Ledger Invariant Violation: Total Debits (%) must equal Total Credits (%) for Entry %', 
            total_debit, total_credit, NEW.ledger_entry_id;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_verify_ledger_balance ON tenant_darussalam_store.ledger_entry_lines;
CREATE CONSTRAINT TRIGGER trg_verify_ledger_balance
AFTER INSERT OR UPDATE ON tenant_darussalam_store.ledger_entry_lines
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION tenant_darussalam_store.verify_ledger_entry_balance();

-- Seed Chart of Accounts
INSERT INTO tenant_darussalam_store.ledger_accounts (code, name, account_type, is_zakat_eligible) VALUES
    ('1010', 'Cash on Hand', 'ASSET', TRUE),
    ('1020', 'Bank Operating Account', 'ASSET', TRUE),
    ('1030', 'Merchandise Inventory', 'ASSET', TRUE),
    ('1040', 'Trade Accounts Receivable', 'ASSET', TRUE),
    ('2010', 'Trade Accounts Payable', 'LIABILITY', TRUE),
    ('3010', 'Owner''s Equity', 'EQUITY', FALSE),
    ('4010', 'Sales Revenue', 'REVENUE', FALSE),
    ('5010', 'Cost of Goods Sold', 'EXPENSE', FALSE),
    ('5020', 'Inventory Shrinkage & Loss', 'EXPENSE', FALSE)
ON CONFLICT (code) DO NOTHING;
