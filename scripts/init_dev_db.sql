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
    ('55555555-5555-5555-5555-555555555555', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'superadmin@tenet.internal', '$2a$10$ccdY3IxyNJFUGpEzYG4F3OwGsEXNZa4NX1F4/G.FP.QCty.grj29y', 'Super Administrator', 'SUPER_ADMIN', TRUE)
ON CONFLICT (email) DO NOTHING;

-- 4. Clean & Re-create Isolated Tenant Schemas for Dev
CREATE SCHEMA IF NOT EXISTS tenant_al_barakah_mart;
CREATE SCHEMA IF NOT EXISTS tenant_darussalam_store;

-- ==============================================================================
-- 5. SCHEMA SETUP: tenant_al_barakah_mart
-- ==============================================================================

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
    is_halal_certified BOOLEAN NOT NULL DEFAULT TRUE,
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
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_abm_transactions_created ON tenant_al_barakah_mart.transactions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_abm_transactions_idempotency ON tenant_al_barakah_mart.transactions(idempotency_key);

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

-- 5.6 Seed Categories for tenant_al_barakah_mart
INSERT INTO tenant_al_barakah_mart.categories (id, name, code)
VALUES
    ('c0000000-0000-0000-0000-000000000001', 'Fresh Meat & Poultry', 'CAT-FRESH-MEAT'),
    ('c0000000-0000-0000-0000-000000000002', 'Beverages & Syrups', 'CAT-BEVERAGES'),
    ('c0000000-0000-0000-0000-000000000003', 'Pantry & Groceries', 'CAT-PANTRY')
ON CONFLICT (code) DO NOTHING;

-- 5.7 Seed Products for tenant_al_barakah_mart
INSERT INTO tenant_al_barakah_mart.products (id, category_id, sku, barcode, name, description, unit_price, cost_price, is_halal_certified, is_active)
VALUES
    ('10000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000001', 'SKU-BEEF-01', '8991001000011', 'Daging Sapi Halal Al-Barakah 500g', 'Daging sapi segar bersertifikat Halal MUI', 75000.00, 60000.00, TRUE, TRUE),
    ('10000000-0000-0000-0000-000000000002', 'c0000000-0000-0000-0000-000000000001', 'SKU-CHICKEN-01', '8991001000028', 'Ayam Potong Segar 1kg', 'Ayam potong higienis dan halal', 38000.00, 30000.00, TRUE, TRUE),
    ('10000000-0000-0000-0000-000000000003', 'c0000000-0000-0000-0000-000000000002', 'SKU-HONEY-01', '8991001000035', 'Madu Murni Al-Barakah 350ml', 'Madu hutan murni tanpa bahan pengawet', 65000.00, 48000.00, TRUE, TRUE),
    ('10000000-0000-0000-0000-000000000004', 'c0000000-0000-0000-0000-000000000003', 'SKU-OIL-01', '8991001000042', 'Minyak Goreng Kelapa Sawit 2L', 'Minyak goreng jernih berkualitas tinggi', 34000.00, 29000.00, TRUE, TRUE),
    ('10000000-0000-0000-0000-000000000005', 'c0000000-0000-0000-0000-000000000003', 'SKU-RICE-01', '8991001000059', 'Beras Ramos Organik 5kg', 'Beras putih pulen organik', 72000.00, 62000.00, TRUE, TRUE)
ON CONFLICT (sku) DO UPDATE SET 
    name = EXCLUDED.name,
    unit_price = EXCLUDED.unit_price,
    cost_price = EXCLUDED.cost_price,
    barcode = EXCLUDED.barcode,
    is_halal_certified = EXCLUDED.is_halal_certified;

-- 5.8 Seed Inventory Stock for tenant_al_barakah_mart
INSERT INTO tenant_al_barakah_mart.inventory (product_id, stock_quantity, reorder_threshold, warehouse_location)
SELECT id, 50, 10, 'MAIN_STORE' FROM tenant_al_barakah_mart.products
ON CONFLICT (product_id) DO UPDATE SET stock_quantity = EXCLUDED.stock_quantity;

-- ==============================================================================
-- 6. SCHEMA SETUP: tenant_darussalam_store
-- ==============================================================================

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
    is_halal_certified BOOLEAN NOT NULL DEFAULT TRUE,
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
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ds_transactions_created ON tenant_darussalam_store.transactions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ds_transactions_idempotency ON tenant_darussalam_store.transactions(idempotency_key);

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

-- 6.6 Seed Products for tenant_darussalam_store
INSERT INTO tenant_darussalam_store.products (id, sku, barcode, name, unit_price, cost_price, is_halal_certified, is_active)
VALUES 
    ('20000000-0000-0000-0000-000000000001', 'SKU-DATES-01', '8992002000018', 'Kurma Ajwa Madinah Darussalam 1kg', 190000.00, 150000.00, TRUE, TRUE),
    ('20000000-0000-0000-0000-000000000002', 'SKU-ZAMZAM-01', '8992002000025', 'Air Zamzam Murni 5L', 350000.00, 280000.00, TRUE, TRUE)
ON CONFLICT (sku) DO UPDATE SET 
    name = EXCLUDED.name,
    unit_price = EXCLUDED.unit_price,
    cost_price = EXCLUDED.cost_price,
    barcode = EXCLUDED.barcode,
    is_halal_certified = EXCLUDED.is_halal_certified;

-- 6.7 Seed Inventory Stock for tenant_darussalam_store
INSERT INTO tenant_darussalam_store.inventory (product_id, stock_quantity, reorder_threshold, warehouse_location)
SELECT id, 30, 5, 'MAIN_STORE' FROM tenant_darussalam_store.products
ON CONFLICT (product_id) DO UPDATE SET stock_quantity = EXCLUDED.stock_quantity;
