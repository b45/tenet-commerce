-- Apply before deploying the certificate-validity service.
-- Re-runnable upgrade for every trusted tenant in the registry; existing documents
-- retain absent evaluation history rather than receiving fabricated decisions.
BEGIN;
DO $upgrade$
DECLARE
    tenant_schema TEXT;
BEGIN
    FOR tenant_schema IN SELECT schema_name FROM public.tenants WHERE status = 'ACTIVE' ORDER BY schema_name LOOP
        IF tenant_schema !~ '^tenant_[a-z0-9_]+$' THEN
            RAISE EXCEPTION 'Unsafe tenant schema in registry: %', tenant_schema;
        END IF;
        IF to_regnamespace(tenant_schema) IS NULL THEN
            RAISE EXCEPTION 'Missing tenant schema: %', tenant_schema;
        END IF;
        EXECUTE format('ALTER TABLE %I.compliance_certificates ADD COLUMN IF NOT EXISTS revoked_at TIMESTAMPTZ', tenant_schema);
        EXECUTE format('CREATE TABLE IF NOT EXISTS %I.compliance_decisions (
            document_type TEXT NOT NULL CHECK (document_type IN (''PO'', ''GR'')),
            document_id UUID NOT NULL,
            decision JSONB NOT NULL,
            PRIMARY KEY (document_type, document_id)
        )', tenant_schema);
        EXECUTE format($ddl$
            CREATE OR REPLACE FUNCTION %I.prevent_compliance_decision_mutation()
            RETURNS TRIGGER AS $body$
            BEGIN
                RAISE EXCEPTION 'Compliance decisions are immutable';
            END;
            $body$ LANGUAGE plpgsql
        $ddl$, tenant_schema);
        EXECUTE format('DROP TRIGGER IF EXISTS immutable_compliance_decision ON %I.compliance_decisions', tenant_schema);
        EXECUTE format('CREATE TRIGGER immutable_compliance_decision BEFORE UPDATE OR DELETE ON %I.compliance_decisions FOR EACH ROW EXECUTE FUNCTION %I.prevent_compliance_decision_mutation()', tenant_schema, tenant_schema);
    END LOOP;
END;
$upgrade$;
COMMIT;
