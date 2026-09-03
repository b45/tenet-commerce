# Technical Architecture & System Design
## Tenet Commerce: Multi-Tenant Enterprise POS & Halal Supply Chain

> **Implementation boundary (2026-09-03):** the Phase 1–2 Go backend is the implemented system. Diagrams and sections that describe the Next.js offline client, AI auditor, Zakat engine, production CI/CD, durable idempotency, Redlock, or a tenant-migration runner are target designs for later phases unless explicitly identified as current runtime behavior. See [Implementation Status](IMPLEMENTATION_STATUS.md).

---

## 1. Architectural Philosophy & Overview

Tenet Commerce is engineered as a **Cloud-Native Modular Monolith** with **Logical Multi-Tenant Isolation (Schema-per-Tenant)** in PostgreSQL. 

The modular monolith paradigm was selected over distributed microservices for the MVP phase to eliminate network serialization latency, distributed transaction complexity (SAGA/2PC), and distributed tracing overhead, while preserving strict domain boundaries for future service decomposition.

```
┌─────────────────────────────────────────────────────────────────────────────────────────┐
│                                    CLIENT PLATFORM                                      │
│                  Next.js 14 App Router · React 18 · TypeScript · shadcn/ui              │
│                     Service Worker (Background Sync) · IndexedDB Queue                  │
└────────────────────────────────────────────┬────────────────────────────────────────────┘
                                             │ HTTPS / REST (Idempotency-Key)
┌────────────────────────────────────────────▼────────────────────────────────────────────┐
│                                  API GATEWAY INTERCEPTORS                               │
│        JWT Validation · RBAC Guard · Tenant Context Resolver · Idempotency Gatekeeper   │
├─────────────────────────────────────────────────────────────────────────────────────────┤
│                                  MODULAR MONOLITH CORE (Go)                             │
│                                                                                         │
│   ┌─────────────────────┐  ┌─────────────────────┐  ┌───────────────────────────────┐   │
│   │     POS & SALES     │  │  HALAL SUPPLY CHAIN │  │     SHARIA GENERAL LEDGER     │   │
│   │                     │  │                     │  │                               │   │
│   │ • Barcode / Catalog │  │ • Supplier Registry │  │ • Double-Entry Journal Engine │   │
│   │ • Cart Processing   │  │ • Cert Hard-Block   │  │ • Chart of Accounts (COA)     │   │
│   │ • Transaction Exec  │  │ • PO & Goods Receipt│  │ • Real-time Zakat Tijarah     │   │
│   └──────────┬──────────┘  └──────────┬──────────┘  └───────────────▲───────────────┘   │
│              │                        │                             │                   │
│              └────────────────────────┴─────────────────────────────┘                   │
│                                Synchronous Internal Domain Events                       │
├─────────────────────────────────────────────────────────────────────────────────────────┤
│                                  SHARED INFRASTRUCTURE LAYER                            │
│           PostgreSQL Connection Pool · Tenant Search Path Router · Redis Driver         │
└────────────────────────────┬───────────────────────────────────────┬────────────────────┘
                             │                                       │
            ┌────────────────▼────────────────┐    ┌─────────────────▼────────────────┐
            │         PostgreSQL 16           │    │              Redis 7             │
            │   • public schema (Tenants)     │    │  • Idempotency Keys (TTL: 24h)   │
            │   • tenant_{uuid} schemas       │    │  • Redlock Distributed Locks     │
            └────────────────▲────────────────┘    └──────────────────────────────────┘
                             │
            ┌────────────────┴────────────────────────────────────────────────────────┐
            │                     CONTINUOUS SHARIA AUDITOR (Python 3.12)             │
            │         Scheduled Cron Worker · Statistical Anomaly Detection           │
            │              Benford's Law · Off-Hours Anomaly · Report Generator       │
            └─────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Multi-Tenancy Architecture (Schema-per-Tenant)

### 2.1 Isolation Model Analysis

| Multi-Tenancy Pattern | Data Isolation | Blast Radius | Schema Migration Complexity | Chosen |
|---|---|---|---|---|
| **Shared Database, Shared Schema** (Tenant ID Column) | Low (Risk of WHERE clause leakage) | High | Minimal | ❌ |
| **Shared Database, Schema-per-Tenant** | **High (Logical Schema Separation)** | **Low** | **Moderate (Automated Runner)** | ✅ |
| **Database-per-Tenant** | Highest (Physical Separation) | Minimal | High (Resource intensive for MVP) | ❌ |

### 2.2 Schema-per-Tenant Routing Mechanics

1. **Global Registry:** The `public` schema maintains the immutable master registry of all provisioned tenants and system configurations:
   ```sql
   CREATE TABLE public.tenants (
       id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
       slug VARCHAR(63) NOT NULL UNIQUE,
       name VARCHAR(255) NOT NULL,
       schema_name VARCHAR(63) NOT NULL UNIQUE,
       status VARCHAR(31) NOT NULL DEFAULT 'ACTIVE',
       created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
   );
   ```

2. **Per-Request Tenant Context Resolution:**
   - The API Gateway extracts the `tenant_id` claim from the validated JWT token.
   - The connection pool middleware allocates a connection and dynamically sets PostgreSQL's `search_path`:
   ```go
   func TenantMiddleware(db *pgxpool.Pool) gin.HandlerFunc {
       return func(c *gin.Context) {
           tenantSchema := c.GetString("tenant_schema") // e.g. "tenant_acme_retail"
           
           // Acquire connection from pool
           conn, err := db.Acquire(c.Request.Context())
           if err != nil {
               c.AbortWithStatusJSON(500, gin.H{"error": "Database connection error"})
               return
           }
           defer conn.Release()

           // Dynamically set search_path
           query := fmt.Sprintf("SET search_path TO %s, public;", pgx.Identifier{tenantSchema}.Sanitize())
           if _, err := conn.Exec(c.Request.Context(), query); err != nil {
               c.AbortWithStatusJSON(500, gin.H{"error": "Failed to set tenant context"})
               return
           }

           c.Set("db_conn", conn)
           c.Next()
       }
   }
   ```

3. **Automated Migration Runner:**
   Database migrations are executed in parallel across all active tenant schemas using a transactional DDL migration tool (e.g., `golang-migrate` or `pressly/goose`):
   ```sql
   -- Loop through all active schemas during migration cycle
   DO $$
   DECLARE
       r RECORD;
   BEGIN
       FOR r IN SELECT schema_name FROM public.tenants WHERE status = 'ACTIVE'
       LOOP
           EXECUTE 'SET search_path TO ' || quote_ident(r.schema_name);
           -- Execute DDL delta
       END LOOP;
   END $$;
   ```

---

## 3. Transaction Engine, Idempotency & Distributed Locking

### 3.1 Idempotency Key Lifecycle

To prevent duplicate charges, inventory overselling, or ghost transactions caused by network dropouts and client-side retries, all mutating operations (`POST /api/v1/transactions`, `POST /api/v1/purchase-orders`) require an `Idempotency-Key` HTTP header.

```
Client                             API Gateway (Go)                         Redis 7
  │                                       │                                    │
  │─── POST /transactions ───────────────>│                                    │
  │    Header: Idempotency-Key: <UUID>    │                                    │
  │                                       │─── SETNX idem:<tenant>:<UUID> ────>│
  │                                       │    Val: "IN_FLIGHT", TTL: 86400s   │
  │                                       │<── 1 (Key Acquired) ───────────────│
  │                                       │                                    │
  │                                       │─── [Execute DB Transaction] ───────│
  │                                       │    • Decrement Inventory           │
  │                                       │    • Write POS Transaction         │
  │                                       │    • Write Double-Entry Ledger     │
  │                                       │                                    │
  │                                       │─── SET idem:<tenant>:<UUID> ──────>│
  │                                       │    Val: <JSON Response Payload>    │
  │                                       │<── OK ─────────────────────────────│
  │                                       │                                    │
  │<── HTTP 201 Created (Receipt Payload)─│                                    │
  │                                       │                                    │
  │─── (Network Retry: Same UUID) ───────>│                                    │
  │                                       │─── SETNX idem:<tenant>:<UUID> ────>│
  │                                       │<── 0 (Key Already Exists) ─────────│
  │                                       │─── GET idem:<tenant>:<UUID> ──────>│
  │                                       │<── Return Cached JSON Payload ─────│
  │<── HTTP 200/201 (Original Response)───│                                    │
```

### 3.2 Dual-Layer Inventory Concurrency Protection

To guarantee zero overselling under heavy concurrent checkouts (e.g., flash sales):

1. **Layer 1: Redis Distributed Lock (Fast Rejection):**
   ```go
   // Key: lock:inventory:<tenant_id>:<sku_id>
   lockKey := fmt.Sprintf("lock:inventory:%s:%s", tenantID, item.SKU)
   mutex := redsync.New(pool).NewMutex(lockKey, redsync.WithExpiry(5*time.Second))
   if err := mutex.Lock(); err != nil {
       return ErrConcurrentItemModification
   }
   defer mutex.Unlock()
   ```

2. **Layer 2: PostgreSQL Row-Level Lock (ACID Guarantee):**
   ```sql
   -- Executed inside the database transaction
   SELECT stock_quantity 
   FROM inventory 
   WHERE product_id = $1 
   FOR UPDATE;
   ```

### 3.3 Atomic Transaction Void & Reversal Architecture

When a transaction is voided or refunded (e.g., customer cancel, incorrect cake size/flavor, defective pastry):
1. **Row-Level Lock:** `SELECT status FROM transactions WHERE id = $1 FOR UPDATE`.
2. **State Validation & Mutex:** Verifies the transaction is currently `COMPLETED` (prevents double-voids). Updates status to `VOIDED` along with `void_reason`, `voided_by`, and timestamp.
3. **Stock Re-crediting:** Atomically increments `inventory.stock_quantity` for each item in `transaction_items`.
4. **Sharia Ledger Reversal:** Posts an automated `JE-VOID-...` journal entry reversing revenue (`4010`), liquid asset accounts (`1010 Cash` or `1020 Bank`), COGS (`5010`), and merchandise inventory asset (`1030`).
5. **Idempotency Defense:** Guards the `POST /api/v1/pos/orders/:id/void` route with Redis 24h key locking.

### 3.4 Retail QRIS & Dynamic Tenant Invoicing

Tenet Commerce supports Indonesian Standard QR Code (**QRIS**) payment flows:
- Tenant-specific merchant identities, NMID, and EMVCo-compliant QR strings are managed dynamically in `store_settings` (`qris` key).
- Cashiers can fetch live QR codes on the POS terminal for customer mobile banking/e-wallet scans (BCA, Mandiri, GoPay, OVO, Dana, LinkAja).
- Eliminates manual EDV configuration changes during seasonal shifts or kiosk pop-up operations.

---

## 4. Target Design — Phase 3 Offline-First Synchronization

The Next.js 14 POS client is built with an offline-first foundation to operate without disruption during network outages.

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              BROWSER CLIENT RUNTIME                             │
│                                                                                 │
│   ┌─────────────────────┐                         ┌─────────────────────────┐   │
│   │   React POS Cart    │                         │     IndexedDB Queue     │   │
│   │                     │                         │                         │   │
│   │ • Barcode Input     │   Dispatch Transaction  │ • Txn UUID              │   │
│   │ • Subtotal / Tax    │────────────────────────>│ • Idempotency-Key       │   │
│   │ • Checkout Button   │                         │ • Items & Quantities    │   │
│   └─────────────────────┘                         │ • Status: PENDING_SYNC  │   │
│                                                   └────────────┬────────────┘   │
│                                                                │                │
│                                                     Poll State │                │
│   ┌────────────────────────────────────────────────────────────▼────────────┐   │
│   │                       SERVICE WORKER BACKGROUND SYNC                     │   │
│   │                                                                         │   │
│   │  • Detects 'online' event                                               │   │
│   │  • Reads queued transactions from IndexedDB (FIFO order)                │   │
│   │  • Dispatches POST /api/v1/transactions with preserved Idempotency-Key  │   │
│   │  • On HTTP 201: Updates IndexedDB status to 'SYNCED'                    │   │
│   │  • On HTTP 4xx/5xx: Applies Exponential Backoff (1s, 2s, 4s, 8s)        │   │
│   └────────────────────────────────────────┬────────────────────────────────┘   │
└────────────────────────────────────────────┼────────────────────────────────────┘
                                             │ Background HTTPS Sync
                                             ▼
                                  API Gateway & Backend Engine
```

---

## 5. Security Architecture & RBAC

### 5.1 JWT Token Schema
```json
{
  "iss": "tenet-commerce-auth",
  "sub": "usr_9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d",
  "tenant_id": "ten_a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
  "tenant_slug": "al-barakah-mart",
  "role": "MANAGER",
  "permissions": [
    "pos:checkout",
    "inventory:read",
    "inventory:write",
    "supply_chain:po:create",
    "ledger:view"
  ],
  "iat": 1788100000,
  "exp": 1788100900
}
```

### 5.2 Role-Based Access Control (RBAC) Matrix

| Permission String | Cashier | Store Manager | Compliance Officer | Financial Admin | Super Admin |
|---|:---:|:---:|:---:|:---:|:---:|
| `pos:checkout` | ✅ | ✅ | ❌ | ❌ | ✅ |
| `pos:read` | ✅ | ✅ | ❌ | ❌ | ✅ |
| `pos:void` | ✅ | ✅ | ❌ | ❌ | ✅ |
| `inventory:read` | ✅ | ✅ | ✅ | ✅ | ✅ |
| `inventory:write` | ❌ | ✅ | ❌ | ❌ | ✅ |
| `supply_chain:manage` | ❌ | ✅ | ✅ | ❌ | ✅ |
| `halal_cert:override` | ❌ | ❌ | ❌ (Strict Block) | ❌ | ❌ (Strict Block) |
| `ledger:read` | ❌ | ✅ | ✅ | ✅ | ✅ |
| `ledger:write` | ❌ | ❌ | ❌ | ✅ | ✅ |
| `ai_audit:view` | ❌ | ✅ | ✅ | ✅ | ✅ |
| `tenant:manage` | ❌ | ❌ | ❌ | ❌ | ✅ |

---

## 6. Target Design — Phase 4 Asynchronous AI Sharia Auditor

The AI Auditor is an isolated, asynchronous worker written in **Python 3.12** that connects to tenant schemas with **read-only database privileges**.

```
┌────────────────────────────────────────────────────────────────────────┐
│               AI AUDITOR WORKER RUNTIME (Python 3.12)                  │
│                                                                        │
│  ┌───────────────────────┐         ┌────────────────────────────────┐  │
│  │ APScheduler (Cron)    │────────>│ Tenant Schema Iterator         │  │
│  │ (Runs Every Sunday)   │         │ (Reads Active Tenants)         │  │
│  └───────────────────────┘         └───────────────┬────────────────┘  │
│                                                    │                   │
│                                     Extract Batch  │                   │
│                                     Ledger Records ▼                   │
│                                    ┌────────────────────────────────┐  │
│                                    │ Anomaly Detection Pipeline     │  │
│                                    │                                │  │
│                                    │ 1. Benford's Law Digits Check  │  │
│                                    │ 2. Off-Hours Sales Clustering  │  │
│                                    │ 3. Z-Score Volume Outliers     │  │
│                                    │ 4. Unbalanced Account Auditing │  │
│                                    └───────────────┬────────────────┘  │
│                                                    │                   │
│                                     Synthesize     ▼                   │
│                                    ┌────────────────────────────────┐  │
│                                    │ Audit Report Generator         │  │
│                                    │ • Severity: INFO/WARN/CRITICAL │  │
│                                    │ • Explainable Anomaly Payload  │  │
│                                    └───────────────┬────────────────┘  │
└────────────────────────────────────────────────────┼───────────────────┘
                                                     │ Write Report Record
                                                     ▼
                                     PostgreSQL (tenant_*.ai_audit_reports)
```

---

## 7. CI/CD & DevOps: Current Pipeline and Target Design

```yaml
name: CI/CD Pipeline

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  lint-and-validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Go Linter
        uses: golangci/golangci-lint-action@v4
        with:
          version: latest
      - name: Python Linter (Ruff & Mypy)
        run: |
          pip install ruff mypy
          ruff check ai-worker/
          mypy ai-worker/
      - name: Frontend Linter & Typecheck
        run: |
          cd frontend && npm ci
          npm run lint
          npx tsc --noEmit

  integration-tests:
    needs: lint-and-validate
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_PASSWORD: test
        ports: [5432:5432]
      redis:
        image: redis:7-alpine
        ports: [6379:6379]
    steps:
      - uses: actions/checkout@v4
      - name: Run Backend Integration Tests
        run: go test -v -race ./...

  build-and-publish:
    needs: integration-tests
    if: github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    steps:
      - name: Build Multi-Stage Docker Images
        run: |
          docker build -t ghcr.io/${{ github.repository }}/backend:latest ./backend
          docker build -t ghcr.io/${{ github.repository }}/ai-worker:latest ./ai-worker
          docker build -t ghcr.io/${{ github.repository }}/frontend:latest ./frontend
```

---

## 7. Observability, Logging & Distributed Tracing

Tenet Commerce implements enterprise-grade, zero-dependency structured logging using Go 1.21+ `log/slog` designed for native compatibility with **Grafana Loki**, **Promtail**, and **CloudWatch**.

### 7.1 Tracing & Correlation Model

```
Client Request (X-Trace-ID optional)
  │
  ▼
[RealIPMiddleware] ────────► Resolves client IP (CF-Connecting-IP → X-Real-IP → X-Forwarded-For)
  │
  ▼
[TraceMiddleware] ─────────► Generates/propagates X-Trace-ID (UUIDv4) + X-Span-ID (8-byte hex)
  │                          Injects scoped logger into Go context.Context
  │
  ▼
[AccessLogMiddleware] ─────► Measures latency_ms, captures status, tenant_slug, user_id, role
  │
  ▼
[RecoveryMiddleware] ──────► Catches panics, logs full debug.Stack() in JSON, returns clean 500
```

### 7.2 Structured JSON Log Format (Loki-Ready)

```json
{
  "time": "2026-08-31T23:00:00.123Z",
  "level": "INFO",
  "msg": "HTTP Request Completed",
  "trace_id": "8f3b6c2a-9e1d-4b5c-8a7e-123456789abc",
  "span_id": "a1b2c3d4e5f67890",
  "method": "POST",
  "path": "/api/v1/auth/login",
  "status": 200,
  "latency_ms": 12.45,
  "client_ip": "103.28.12.45",
  "user_agent": "Mozilla/5.0 ...",
  "tenant_slug": "al-barakah-mart",
  "user_id": "11111111-1111-1111-1111-111111111111",
  "role": "CASHIER"
}
```

---

## 8. POS Transaction Engine: Current Behavior and Hardening Target

The Point of Sale (POS) checkout engine executes sales with strict ACID guarantees, atomic stock decrements, and double-charge prevention.

### 8.1 Dual-Layer Concurrency & Idempotency Model

```
POS Client (Checkout Submission)
  │  Headers: Authorization: Bearer <token>, Idempotency-Key: <uuid>
  │
  ▼
[IdempotencyMiddleware (Redis)]
  ├─ SETNX idempotency:{tenant_slug}:{idempotency_key} (TTL: 24h)
  ├─ If Key Exists & Completed → Returns Cached Response (X-Cache-Lookup: HIT)
  └─ If Key New → Acquires Lock & Proceeds to Execution
  │
  ▼
[POS Checkout Service]
  │  PostgreSQL Transaction (`pgx.Tx`) on Schema `tenant_{slug}`
  │
  ├─ 1. SELECT ... FROM products p JOIN inventory i ON p.id = i.product_id
  │     WHERE p.sku = ANY($1) FOR UPDATE OF p, i
  │     ↳ Acquires Row-Level Exclusive Lock on requested products & stock
  │
  ├─ 2. Stock Validation: IF stock_quantity < requested_qty → ROLLBACK + 409 Conflict
  │
  ├─ 3. Atomic Stock Decrement: UPDATE inventory SET stock_quantity = stock_quantity - $qty
  │
  ├─ 4. Insert Master Record: INSERT INTO transactions (...)
  │     Insert Line Items:   INSERT INTO transaction_items (...)
  │
  ├─ 5. COMMIT Transaction
  │
  └─ 6. Cache Response in Redis (TTL: 24h) → Return 201 Created (Receipt)
```

### 8.2 APM-Grade Transaction Latency Profiling (Grafana Loki Ready)

To enable granular performance observability without requiring an external APM agent, the POS checkout service instruments sub-millisecond execution timers across all critical database phases and emits them in the structured JSON log:

```json
{
  "time": "2026-09-01T00:05:26.952Z",
  "level": "INFO",
  "msg": "POS Checkout Transaction Completed",
  "trace_id": "84ae8d69-edf6-422f-af64-b29e92a0fcb8",
  "span_id": "b49b43020a975b09",
  "transaction_number": "TXN-20260901-b36252",
  "total_amount": 145000,
  "items_count": 1,
  "duration_total_ms": 8.761,
  "duration_lock_products_ms": 1.210,
  "duration_stock_decrement_ms": 1.041,
  "duration_insert_txn_ms": 1.606,
  "duration_insert_items_ms": 2.456,
  "duration_commit_ms": 2.155
}
```

#### Grafana LogQL Query Examples for APM Dashboards:
- **P95 Lock Contention Latency:**
  `quantile_over_time(0.95, {service="tenet-commerce-api"} | json | unwrap duration_lock_products_ms [5m])`
- **P99 Commit Latency:**
  `quantile_over_time(0.99, {service="tenet-commerce-api"} | json | unwrap duration_commit_ms [5m])`
- **Average Item Insert Latency:**
  `avg_over_time({service="tenet-commerce-api"} | json | unwrap duration_insert_items_ms [1m])`

---

*Tenet Commerce — Technical Architecture Documentation v1.0.0*
