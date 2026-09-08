# Tenant reconciliation

Use a disposable database initialized from `scripts/init_dev_db.sql`. Select an active schema from its trusted `public.tenants` registry. Run from the repository root with PostgreSQL client tools and standard connection environment variables:

```sh
psql -X -v ON_ERROR_STOP=1 -v tenant_schema=tenant_al_barakah_mart -f docs/verification/reconcile.sql
```

The script resolves exactly one active registry entry, quotes the schema identifier, and uses a repeatable-read, read-only transaction. An absent registry entry fails the command. Both result sets should be empty; returned rows are discrepancies requiring investigation, not automated repairs. SQL execution success alone does not mean the result sets are empty.

The stock query assumes opening cutover has occurred and subsequent mutations have movement records. It does not infer missing legacy history. These checks do not prove tenant isolation, crash recovery, or concurrent correctness; run the corresponding integration suites linked from [architecture evidence](../ARCHITECTURE_EVIDENCE.md).
