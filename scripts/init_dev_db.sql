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

-- 4. Create Isolated Tenant Schemas
CREATE SCHEMA IF NOT EXISTS tenant_al_barakah_mart;
CREATE SCHEMA IF NOT EXISTS tenant_darussalam_store;

-- 5. Create Sample Products Table inside tenant_al_barakah_mart
CREATE TABLE IF NOT EXISTS tenant_al_barakah_mart.products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sku VARCHAR(63) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    unit_price NUMERIC(15, 2) NOT NULL,
    is_halal_certified BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed product for Tenant A
INSERT INTO tenant_al_barakah_mart.products (sku, name, unit_price, is_halal_certified)
VALUES ('SKU-BEEF-01', 'Daging Sapi Halal Al-Barakah 500g', 75000.00, true)
ON CONFLICT (sku) DO NOTHING;

-- 6. Create Sample Products Table inside tenant_darussalam_store
CREATE TABLE IF NOT EXISTS tenant_darussalam_store.products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sku VARCHAR(63) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    unit_price NUMERIC(15, 2) NOT NULL,
    is_halal_certified BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed product for Tenant B (Completely different product to test isolation)
INSERT INTO tenant_darussalam_store.products (sku, name, unit_price, is_halal_certified)
VALUES ('SKU-DATES-01', 'Kurma Ajwa Madinah Darussalam 1kg', 190000.00, true)
ON CONFLICT (sku) DO NOTHING;
