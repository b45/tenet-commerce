-- ==============================================================================
-- Tenet Commerce: SaaS Feature Access & Entitlement Schema
-- Extends public schema with subscription plans, feature grants, and audit logs.
-- ==============================================================================

-- 1. Subscription Plans Catalog (Public Schema)
CREATE TABLE IF NOT EXISTS public.plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(63) NOT NULL UNIQUE,          -- 'starter', 'growth', 'enterprise'
    name VARCHAR(255) NOT NULL,
    version INT NOT NULL DEFAULT 1,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Plan Features & Quota Grants (Public Schema)
CREATE TABLE IF NOT EXISTS public.plan_features (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id UUID NOT NULL REFERENCES public.plans(id) ON DELETE CASCADE,
    feature_key VARCHAR(127) NOT NULL,
    grant_type VARCHAR(31) NOT NULL DEFAULT 'BOOLEAN' CHECK (grant_type IN ('BOOLEAN', 'QUOTA')),
    quota_limit INT NULL,                      -- NULL represents unlimited capacity
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_plan_feature UNIQUE (plan_id, feature_key)
);

CREATE INDEX IF NOT EXISTS idx_plan_features_plan_key ON public.plan_features(plan_id, feature_key);

-- 3. Tenant Subscriptions (Public Schema)
CREATE TABLE IF NOT EXISTS public.tenant_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL UNIQUE REFERENCES public.tenants(id) ON DELETE RESTRICT,
    plan_id UUID NOT NULL REFERENCES public.plans(id) ON DELETE RESTRICT,
    status VARCHAR(31) NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'TRIALING', 'PAST_DUE', 'CANCELED')),
    current_period_start TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    current_period_end TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '365 days'),
    canceled_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tenant_subscriptions_tenant ON public.tenant_subscriptions(tenant_id);

-- 4. Subscription Audit Log (Public Schema)
CREATE TABLE IF NOT EXISTS public.subscription_audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    actor_id UUID NULL REFERENCES public.users(id) ON DELETE SET NULL,
    action VARCHAR(63) NOT NULL,               -- 'PLAN_ASSIGNED', 'UPGRADED', 'DOWNGRADED', 'STATUS_CHANGED'
    before_state JSONB NULL,
    after_state JSONB NOT NULL,
    reason TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ==============================================================================
-- Seed Baseline Plans, Features, and Tenant Subscriptions
-- ==============================================================================

-- 5. Seed Standard Plans
INSERT INTO public.plans (id, code, name, version, is_active)
VALUES
    ('10000000-0000-0000-0000-000000000001', 'starter', 'Starter Retail Tier', 1, TRUE),
    ('10000000-0000-0000-0000-000000000002', 'growth', 'Growth Business Tier', 1, TRUE),
    ('10000000-0000-0000-0000-000000000003', 'enterprise', 'Enterprise Sharia Tier', 1, TRUE)
ON CONFLICT (code) DO NOTHING;

-- 6. Seed Plan Features
-- Starter Tier Features
INSERT INTO public.plan_features (plan_id, feature_key, grant_type, quota_limit, is_enabled)
VALUES
    ('10000000-0000-0000-0000-000000000001', 'pos.checkout', 'BOOLEAN', NULL, TRUE),
    ('10000000-0000-0000-0000-000000000001', 'pos.daily_summary', 'BOOLEAN', NULL, FALSE),
    ('10000000-0000-0000-0000-000000000001', 'inventory.basic', 'BOOLEAN', NULL, TRUE),
    ('10000000-0000-0000-0000-000000000001', 'inventory.stock_opname', 'BOOLEAN', NULL, FALSE),
    ('10000000-0000-0000-0000-000000000001', 'catalog.max_products', 'QUOTA', 100, TRUE)
ON CONFLICT (plan_id, feature_key) DO NOTHING;

-- Growth Tier Features
INSERT INTO public.plan_features (plan_id, feature_key, grant_type, quota_limit, is_enabled)
VALUES
    ('10000000-0000-0000-0000-000000000002', 'pos.checkout', 'BOOLEAN', NULL, TRUE),
    ('10000000-0000-0000-0000-000000000002', 'pos.daily_summary', 'BOOLEAN', NULL, TRUE),
    ('10000000-0000-0000-0000-000000000002', 'inventory.basic', 'BOOLEAN', NULL, TRUE),
    ('10000000-0000-0000-0000-000000000002', 'inventory.stock_opname', 'BOOLEAN', NULL, TRUE),
    ('10000000-0000-0000-0000-000000000002', 'supply_chain.basic', 'BOOLEAN', NULL, TRUE),
    ('10000000-0000-0000-0000-000000000002', 'catalog.max_products', 'QUOTA', 1000, TRUE)
ON CONFLICT (plan_id, feature_key) DO NOTHING;

-- Enterprise Tier Features
INSERT INTO public.plan_features (plan_id, feature_key, grant_type, quota_limit, is_enabled)
VALUES
    ('10000000-0000-0000-0000-000000000003', 'pos.checkout', 'BOOLEAN', NULL, TRUE),
    ('10000000-0000-0000-0000-000000000003', 'pos.daily_summary', 'BOOLEAN', NULL, TRUE),
    ('10000000-0000-0000-0000-000000000003', 'pos.offline_mode', 'BOOLEAN', NULL, TRUE),
    ('10000000-0000-0000-0000-000000000003', 'inventory.basic', 'BOOLEAN', NULL, TRUE),
    ('10000000-0000-0000-0000-000000000003', 'inventory.stock_opname', 'BOOLEAN', NULL, TRUE),
    ('10000000-0000-0000-0000-000000000003', 'inventory.multi_location', 'BOOLEAN', NULL, TRUE),
    ('10000000-0000-0000-0000-000000000003', 'supply_chain.strict_halal', 'BOOLEAN', NULL, TRUE),
    ('10000000-0000-0000-0000-000000000003', 'ledger.reporting', 'BOOLEAN', NULL, TRUE),
    ('10000000-0000-0000-0000-000000000003', 'catalog.max_products', 'QUOTA', NULL, TRUE)
ON CONFLICT (plan_id, feature_key) DO NOTHING;

-- 7. Seed Initial Subscriptions for Test Tenants
-- al-barakah-mart -> Growth Plan (Active)
INSERT INTO public.tenant_subscriptions (id, tenant_id, plan_id, status)
VALUES
    ('20000000-0000-0000-0000-000000000001', 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', '10000000-0000-0000-0000-000000000002', 'ACTIVE')
ON CONFLICT (tenant_id) DO UPDATE
SET plan_id = EXCLUDED.plan_id, status = EXCLUDED.status, updated_at = NOW();

-- darussalam-store -> Starter Plan (Active)
INSERT INTO public.tenant_subscriptions (id, tenant_id, plan_id, status)
VALUES
    ('20000000-0000-0000-0000-000000000002', 'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22', '10000000-0000-0000-0000-000000000001', 'ACTIVE')
ON CONFLICT (tenant_id) DO UPDATE
SET plan_id = EXCLUDED.plan_id, status = EXCLUDED.status, updated_at = NOW();

-- suspended-mart -> Starter Plan (Canceled)
INSERT INTO public.tenant_subscriptions (id, tenant_id, plan_id, status, canceled_at)
VALUES
    ('20000000-0000-0000-0000-000000000003', 'c2eebc99-9c0b-4ef8-bb6d-6bb9bd380a33', '10000000-0000-0000-0000-000000000001', 'CANCELED', NOW())
ON CONFLICT (tenant_id) DO UPDATE
SET plan_id = EXCLUDED.plan_id, status = EXCLUDED.status, canceled_at = EXCLUDED.canceled_at, updated_at = NOW();
