# Route & RBAC Permission Mapping Matrix

> **Purpose:** Defines the end-to-end mapping between Next.js App Router client routes, backend API endpoints (`/api/v1/*`), required RBAC permissions, and authorized user roles.
> **Reference:** `backend/pkg/auth/jwt.go`, `backend/cmd/api/router.go`, `docs/FRONTEND_GUIDELINES.md`.

---

## 1. Role-Based Access Control (RBAC) Hierarchy

The backend defines 5 authoritative roles with granular permissions in `backend/pkg/auth/jwt.go`:

| Role | Domain Scope | Predefined Permissions |
|---|---|---|
| **CASHIER** | Retail POS & Terminal Sales | `pos:checkout`, `pos:read`, `pos:void`, `inventory:read` |
| **MANAGER** | Store Operations & Analytics | `pos:checkout`, `pos:read`, `pos:void`, `inventory:read`, `inventory:write`, `supply_chain:manage`, `ledger:read`, `ai_audit:view` |
| **COMPLIANCE_OFFICER** | Halal Certification & Audit | `inventory:read`, `supply_chain:manage`, `ledger:read`, `ai_audit:view` |
| **FINANCIAL_ADMIN** | General Ledger & Accounting | `inventory:read`, `ledger:read`, `ledger:write`, `ai_audit:view` |
| **SUPER_ADMIN** | Enterprise Tenant Owner | All permissions + `tenant:manage` |

---

## 2. Frontend Route & Backend Endpoint Matrix

| Next.js App Router Route | Backend Endpoints (`/api/v1`) | Required Permission | Authorized Roles | Navigation Location & Fallback |
|---|---|---|---|---|
| `/login` | `POST /auth/login` | *Public* | *All (Unauthenticated)* | Auth layout. Redirects to role home on success. |
| `/` | `GET /auth/me` | *Public / Session Check* | *All* | Root router redirect: CASHIER $\rightarrow$ `/pos`, MANAGER $\rightarrow$ `/dashboard`, etc. |
| **`/pos`** | `GET /pos/products`<br>`GET /pos/categories` | `pos:read`, `inventory:read` | CASHIER, MANAGER, SUPER_ADMIN | Primary POS Register (Catalog, Scanner, Cart). |
| **`/pos/checkout`** | `POST /pos/checkout` | `pos:checkout` | CASHIER, MANAGER, SUPER_ADMIN | POS Cash Payment modal / full view. Idempotent checkout. |
| **`/pos/orders`** | `GET /pos/orders`<br>`GET /pos/daily-summary` | `pos:read` | CASHIER, MANAGER, SUPER_ADMIN | POS Transaction History & Daily Sales Summary. |
| **`/pos/orders/[id]`** | `GET /pos/orders/:id`<br>`POST /pos/orders/:id/void` | `pos:read`<br>(`pos:void` for void action) | CASHIER, MANAGER, SUPER_ADMIN | Order Receipt detail and supervisor void action. |
| **`/inventory`** | `GET /pos/products`<br>`GET /pos/categories`<br>`GET /pos/inventory/low-stock` | `inventory:read` | CASHIER, MANAGER, COMPLIANCE_OFFICER, FINANCIAL_ADMIN, SUPER_ADMIN | Inventory overview & catalog browsing. |
| **`/inventory/adjust`** | `POST /inventory/adjust` | `inventory:write` | MANAGER, SUPER_ADMIN | Stock count correction with mandatory audit reason. |
| **`/supply-chain/suppliers`** | `GET /supply-chain/suppliers`<br>`POST /supply-chain/suppliers` | `supply_chain:manage` | MANAGER, COMPLIANCE_OFFICER, SUPER_ADMIN | Supplier directory and vendor profile management. |
| **`/supply-chain/certificates`**| `GET /supply-chain/certificates`<br>`POST /supply-chain/certificates` | `supply_chain:manage` | MANAGER, COMPLIANCE_OFFICER, SUPER_ADMIN | Halal certificate registry, expiry monitor, status badge. |
| **`/supply-chain/po`** | `GET /supply-chain/po`<br>`POST /supply-chain/po` | `supply_chain:manage` | MANAGER, COMPLIANCE_OFFICER, SUPER_ADMIN | Purchase Order list, creation, and vendor dispatch. |
| **`/supply-chain/gr`** | `GET /supply-chain/gr`<br>`POST /supply-chain/gr` | `supply_chain:manage` | MANAGER, COMPLIANCE_OFFICER, SUPER_ADMIN | Goods Receipt receiving dock, PO reconciliation, cert check. |
| **`/supply-chain/traceability/[sku]`**| `GET /supply-chain/traceability/product/:id` | `supply_chain:manage` | MANAGER, COMPLIANCE_OFFICER, SUPER_ADMIN | End-to-end halal supply chain batch traceability tree. |
| **`/ledger/accounts`** | `GET /ledger/accounts` | `ledger:read` | MANAGER, COMPLIANCE_OFFICER, FINANCIAL_ADMIN, SUPER_ADMIN | Chart of Accounts (COA) structure and balances. |
| **`/ledger/entries`** | `GET /ledger/entries`<br>`POST /ledger/entries` | `ledger:read`<br>(`ledger:write` for new entry) | FINANCIAL_ADMIN, SUPER_ADMIN (create)<br>+ MANAGER, COMPLIANCE_OFFICER (read) | General ledger journal entries with debit/credit balance check. |
| **`/ledger/trial-balance`**| `GET /ledger/trial-balance` | `ledger:read` | FINANCIAL_ADMIN, SUPER_ADMIN, MANAGER, COMPLIANCE_OFFICER | Trial balance sheet verification. |
| **`/dashboard`** | `GET /manager/dashboard` | `pos:read`, `inventory:read`, `supply_chain:manage` | MANAGER, SUPER_ADMIN | Executive summary KPI cards (sales, low stock, expiring certs). |
| `/403` | N/A | *Public* | *All* | Standard Access Denied / Permission Insufficient page. |
| `/404` | N/A | *Public* | *All* | Resource Not Found page. |

---

## 3. Client-Side Authorization & Enforcement Rules

1. **Route Guard Middleware (`frontend/src/middleware.ts`)**:
   - Inspects `tenet_access_token` session claims.
   - If unauthenticated and accessing a protected route $\rightarrow$ Redirect to `/login?next=<path>`.
   - If authenticated but missing required permission $\rightarrow$ Redirect to `/403`.

2. **Sidebar & Menu Visibility (`PermissionGate`)**:
   - Sidebar links for modules outside the user's `permissions[]` are omitted from the DOM to eliminate clutter.
   - *Rule:* UI omission is purely a convenience mechanism; actual authorization is enforced by the Next.js middleware and validated cryptographically by the Go backend on every API request.

3. **Action-Level Permission Constraints**:
   - Even on shared read views (e.g., `/inventory`), action buttons requiring `inventory:write` or `pos:void` render disabled with tooltip: `"Akses terbatas: Memerlukan izin pos:void"`.
