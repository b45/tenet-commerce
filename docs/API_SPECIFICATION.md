# REST API Specification & Endpoint Contracts
## Tenet Commerce: Enterprise POS & Halal Supply Chain API

> **Implementation boundary (2026-09-03):** this specification documents the routes registered by `backend/cmd/api/router.go`. Tenant provisioning, Zakat, and AI-audit endpoints below are future design references only and are **not registered**. Phase 3/4 work must not rely on them.

---

## 1. Global API Standards & Protocols

### 1.1 Base URL & Content Negotiation
- **Local base URL:** `http://localhost:8081/api/v1`
- **Content-Type:** `application/json; charset=utf-8`
- **Production transport:** deployment-specific; this repository does not yet provide a production TLS/API-gateway deployment.

### 1.2 Required Request Headers
```http
Authorization: Bearer <JWT_ACCESS_TOKEN>
Idempotency-Key: <client-generated key> # Required by current middleware only for POS checkout, void, and stock adjustment
X-Tenant-ID: <TENANT_SLUG>              # Fallback only; authenticated JWT tenant context has priority
```

### 1.3 Health Check
- **Endpoint:** `GET /health`
- **Auth:** Public
- **Response:** `200 OK` with the application health payload. This endpoint is outside the `/api/v1` namespace.

### 1.4 Standard Response Envelope
```json
{
  "success": true,
  "data": {},
  "meta": {
    "total": 120,
    "page": 1,
    "limit": 50,
    "offset": 0
  }
}
```

### 1.5 Standard Error Envelope
```json
{
  "success": false,
  "error": {
    "code": "HALAL_CERT_EXPIRED",
    "message": "Supplier Halal Certificate [MUI-123456] expired on 2026-08-01. Operation rejected.",
    "details": [
      {
        "field": "supplier_id",
        "issue": "Expired certificate prevents Purchase Order creation."
      }
    ]
  }
}
```

---

## 2. Authentication

### 2.1 Authenticate User (Login)
- **Endpoint:** `POST /api/v1/auth/login`
- **Auth:** Public
- **Request Body:**
```json
{
  "tenant_slug": "al-barakah-mart",
  "email": "cashier1@albarakah.com",
  "password": "Password123!"
}
```
- **Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "token_type": "Bearer",
    "expires_in": 900,
    "user": {
      "id": "usr_9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d",
      "email": "cashier1@albarakah.com",
      "full_name": "Ahmad Fauzi",
      "role": "CASHIER",
      "tenant_slug": "al-barakah-mart"
    }
  }
}
```

#### Seeded Testing Credentials (Dev / Testing)
Password for all seeded dev accounts: `Password123!`

| Role | Tenant Slug | Email | Permissions Scope |
|---|---|---|---|
| **SUPER_ADMIN (Tenant A)** | `al-barakah-mart` | `superadmin@albarakah.com` | Full unrestricted access (`pos:*`, `inventory:*`, `supply_chain:*`, `ledger:*`, `ai_audit:*`, `tenant:*`) |
| **SUPER_ADMIN (Tenant B)** | `darussalam-store` | `superadmin@darussalam.com` | Full unrestricted access (`pos:*`, `inventory:*`, `supply_chain:*`, `ledger:*`, `ai_audit:*`, `tenant:*`) |
| **MANAGER (Tenant A)** | `al-barakah-mart` | `manager1@albarakah.com` | POS, Inventory CRUD, Supply Chain Management, Ledger Read, AI Audit |
| **MANAGER (Tenant B)** | `darussalam-store` | `manager1@darussalam.com` | POS, Inventory CRUD, Supply Chain Management, Ledger Read, AI Audit |
| **CASHIER** | `al-barakah-mart` | `cashier1@albarakah.com` | POS Checkout, Order History (`pos:read`), Void/Refund (`pos:void`), Inventory Read |
| **FINANCIAL_ADMIN** | `al-barakah-mart` | `finance1@albarakah.com` | Ledger Read/Write, Inventory Read, AI Audit |
| **COMPLIANCE_OFFICER** | `al-barakah-mart` | `compliance1@albarakah.com` | Supply Chain Management, Inventory Read, AI Audit |

### 2.2 Refresh Token
- **Endpoint:** `POST /api/v1/auth/refresh`
- **Auth:** Public
- **Request Body:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```
- **Response:** The same token-pair envelope as login. The current implementation issues a new access and refresh token after validating the submitted refresh token.

### 2.3 Current Identity
- **Endpoint:** `GET /api/v1/auth/me`
- **Auth:** Bearer access token
- **Response:** The authenticated JWT identity (`id`, `tenant_slug`, `role`, and `permissions`).

### 2.4 Tenant Provisioning — Planned / Not Registered
- **Status:** Not implemented as an HTTP endpoint. Tenant registry and schema provisioning are currently development/setup concerns and will be formalized by the tenant-migration hardening workstream.
- **Future design reference (not an active contract):**
- **Endpoint:** `POST /api/v1/tenants`
- **Auth:** `SUPER_ADMIN`
- **Request Body:**
```json
{
  "slug": "darussalam-supermarket",
  "company_name": "PT Darussalam Ritel Syariah",
  "admin_email": "admin@darussalam.com",
  "admin_full_name": "Haji Mansur",
  "admin_password": "SecurePassword#2026"
}
```
- **Response (201 Created):**
```json
{
  "success": true,
  "data": {
    "tenant_id": "ten_6a7b8c9d-0e1f-2a3b-4c5d-6e7f8a9b0c1d",
    "slug": "darussalam-supermarket",
    "schema_name": "tenant_darussalam_supermarket",
    "status": "ACTIVE",
    "provisioned_at": "2026-08-30T10:05:00Z"
  }
}
```

---

## 3. POS & Transaction Engine

### 3.1 List Products & Inventory Catalog
- **Endpoint:** `GET /api/v1/pos/products`
- **Auth:** `CASHIER`, `MANAGER`, `SUPER_ADMIN` (Requires permission: `inventory:read`)
- **Response (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "id": "10000000-0000-0000-0000-000000000001",
      "category_id": "c0000000-0000-0000-0000-000000000001",
      "category_name": "Fresh Meat & Poultry",
      "sku": "SKU-BEEF-01",
      "barcode": "8991001000011",
      "name": "Daging Sapi Halal Al-Barakah 500g",
      "description": "Daging sapi segar bersertifikat Halal MUI",
      "unit_price": 75000.00,
      "cost_price": 60000.00,
      "stock_quantity": 50,
      "is_halal_certified": true,
      "is_active": true,
      "created_at": "2026-09-01T00:00:00Z",
      "updated_at": "2026-09-01T00:00:00Z"
    }
  ],
  "meta": {
    "total": 5
  }
}
```

### 3.2 Idempotent POS Checkout
- **Endpoint:** `POST /api/v1/pos/checkout`
- **Headers:** `Idempotency-Key: <UUIDv4>` (Mandatory)
- **Auth:** `CASHIER`, `MANAGER`, `SUPER_ADMIN` (Requires permission: `pos:checkout`)
- **Request Body:**
```json
{
  "items": [
    {
      "sku": "SKU-BEEF-01",
      "quantity": 2
    }
  ],
  "payment_method": "CASH",
  "cash_tendered": 200000.00,
  "customer_name": "Ibu Kartika",
  "notes": "Tulisan: Selamat Ulang Tahun Salsa (Lilin angka 7)",
  "discount_amount": 5000.00
}
```
- **Cash settlement rule:** when `payment_method` is `CASH`, `cash_tendered` is required and must be at least the calculated total. For `QRIS` and `SIMULATED_CARD`, omit `cash_tendered`.
- **Response (201 Created):**
```json
{
  "success": true,
  "data": {
    "transaction_id": "3f4a5b6c-7d8e-9f0a-1b2c-3d4e5f6a7b8c",
    "transaction_number": "TXN-20260901-0001",
    "idempotency_key": "7b8a1c9e-2f3a-4b5c-6d7e-8f9a0b1c2d3e",
    "cashier_id": "22222222-2222-2222-2222-222222222222",
    "payment_method": "CASH",
    "status": "COMPLETED",
    "customer_name": "Ibu Kartika",
    "notes": "Tulisan: Selamat Ulang Tahun Salsa (Lilin angka 7)",
    "cash_tendered": 200000.00,
    "change_amount": 55000.00,
    "payment_reference": null,
    "items": [
      {
        "id": "item_uuid",
        "transaction_id": "3f4a5b6c-7d8e-9f0a-1b2c-3d4e5f6a7b8c",
        "product_id": "10000000-0000-0000-0000-000000000001",
        "sku": "SKU-BEEF-01",
        "name": "Daging Sapi Halal Al-Barakah 500g",
        "quantity": 2,
        "unit_price": 75000.00,
        "cost_price": 60000.00,
        "subtotal": 150000.00
      }
    ],
    "subtotal_amount": 150000.00,
    "tax_amount": 0.00,
    "discount_amount": 5000.00,
    "total_amount": 145000.00,
    "created_at": "2026-09-01T01:30:00Z"
  }
}
```

### 3.3 List Orders (Order History)
- **Endpoint:** `GET /api/v1/pos/orders`
- **Auth:** `CASHIER`, `MANAGER`, `SUPER_ADMIN` (Requires permission: `pos:read`)
- **Query Parameters:**
  - `limit`: number of records (default: 20, max: 100)
  - `offset`: page offset (default: 0)
  - `start_date`: filter orders after date (`YYYY-MM-DD`)
  - `end_date`: filter orders before date (`YYYY-MM-DD`)
  - `status`: `COMPLETED` | `VOIDED`
  - `payment_method`: `CASH` | `QRIS` | `SIMULATED_CARD`
  - `search`: search by transaction number, customer name, or notes
- **Response (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "id": "3f4a5b6c-7d8e-9f0a-1b2c-3d4e5f6a7b8c",
      "transaction_number": "TXN-20260901-0001",
      "cashier_id": "22222222-2222-2222-2222-222222222222",
      "total_amount": 145000.00,
      "payment_method": "CASH",
      "status": "COMPLETED",
      "customer_name": "Ibu Kartika",
      "total_items": 2,
      "void_reason": null,
      "created_at": "2026-09-01T01:30:00Z"
    }
  ],
  "meta": {
    "total": 1,
    "limit": 20,
    "offset": 0
  }
}
```

### 3.4 Get Order Detail & Receipt
- **Endpoint:** `GET /api/v1/pos/orders/:id`
- **Auth:** `CASHIER`, `MANAGER`, `SUPER_ADMIN` (Requires permission: `pos:read`)
- **Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "transaction": {
      "id": "3f4a5b6c-7d8e-9f0a-1b2c-3d4e5f6a7b8c",
      "transaction_number": "TXN-20260901-0001",
      "idempotency_key": "7b8a1c9e-2f3a-4b5c-6d7e-8f9a0b1c2d3e",
      "cashier_id": "22222222-2222-2222-2222-222222222222",
      "subtotal_amount": 150000.00,
      "tax_amount": 0.00,
      "discount_amount": 5000.00,
      "total_amount": 145000.00,
      "payment_method": "CASH",
      "status": "COMPLETED",
      "customer_name": "Ibu Kartika",
      "notes": "Tulisan: Selamat Ulang Tahun Salsa",
      "cash_tendered": 200000.00,
      "change_amount": 55000.00,
      "payment_reference": null,
      "void_reason": null,
      "voided_at": null,
      "voided_by": null,
      "created_at": "2026-09-01T01:30:00Z"
    },
    "items": [
      {
        "id": "item_uuid",
        "transaction_id": "3f4a5b6c-7d8e-9f0a-1b2c-3d4e5f6a7b8c",
        "product_id": "10000000-0000-0000-0000-000000000001",
        "sku": "SKU-BEEF-01",
        "name": "Daging Sapi Halal Al-Barakah 500g",
        "quantity": 2,
        "unit_price": 75000.00,
        "cost_price": 60000.00,
        "subtotal": 150000.00
      }
    ]
  }
}
```

### 3.5 Atomic Void / Refund Transaction
- **Endpoint:** `POST /api/v1/pos/orders/:id/void`
- **Headers:** `Idempotency-Key: <UUIDv4>` (Mandatory)
- **Auth:** `CASHIER`, `MANAGER`, `SUPER_ADMIN` (Requires permission: `pos:void`)
- **Request Body:**
```json
{
  "reason": "Customer cancel - wrong cake flavor selected"
}
```
- **Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "transaction_id": "3f4a5b6c-7d8e-9f0a-1b2c-3d4e5f6a7b8c",
    "transaction_number": "TXN-20260901-0001",
    "status": "VOIDED",
    "void_reason": "Customer cancel - wrong cake flavor selected",
    "voided_at": "2026-09-01T02:00:00Z",
    "voided_by": "22222222-2222-2222-2222-222222222222",
    "items_restocked": 1,
    "total_refunded": 145000.00
  }
}
```

### 3.6 Daily Cashier Sales Summary (X/Z-Report)
- **Endpoint:** `GET /api/v1/pos/daily-summary`
- **Auth:** `CASHIER`, `MANAGER`, `SUPER_ADMIN` (Requires permission: `pos:read`)
- **Query Parameters:**
  - `date`: `YYYY-MM-DD` (optional, default: current date)
  - `cashier_id`: UUID (optional, filter by specific cashier)
- **Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "date": "2026-09-02",
    "cashier_id": null,
    "total_orders": 12,
    "completed_orders": 11,
    "voided_orders": 1,
    "gross_sales": 1500000.00,
    "discounts": 50000.00,
    "net_sales": 1450000.00,
    "total_cogs": 900000.00,
    "gross_profit": 550000.00,
    "payment_breakdown": {
      "CASH": {
        "count": 7,
        "total_amount": 850000.00
      },
      "QRIS": {
        "count": 4,
        "total_amount": 600000.00
      }
    }
  }
}
```

### 3.7 Tenant QRIS Configuration
- **Endpoint:** `GET /api/v1/pos/qris`
- **Auth:** `CASHIER`, `MANAGER`, `SUPER_ADMIN` (Requires permission: `inventory:read`)
- **Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "merchant_name": "Al Barakah Bakery Syariah",
    "nmid": "ID1020030040050",
    "qr_string": "00020101021126580014ID.LINKAJA.WWW011893600914300000222202151234567890123450303UMI51440014ID.CO.QRIS.WWW0215ID10200300400500303UMI5204549953033605802ID5924AL BARAKAH BAKERY SYARIAH6010JAKARTA SE61051234062070703A0163041D2B",
    "qr_image_url": "https://api.qrserver.com/v1/create-qr-code/?size=300x300&data=..."
  }
}
```

- **Update QRIS Configuration:** `PUT /api/v1/pos/qris`
- **Auth:** `MANAGER`, `SUPER_ADMIN` (Requires permission: `inventory:write`)
- **Request Body:**
```json
{
  "merchant_name": "Toko Kue B45 Bakery QRIS",
  "nmid": "ID1987654321",
  "qr_string": "00020101021126580014ID.LINKAJA.WWW011893600914300000222202151234567890123450303UMI51440014ID.CO.QRIS.WWW0215ID19876543210303UMI5204549953033605802ID5925TOKO KUE B45 BAKERY QRIS6010JAKARTA SE61051234062070703A0163041D2B",
  "qr_image_url": "https://api.qrserver.com/v1/create-qr-code/?size=300x300&data=sample-b45-bakery"
}
```

### 3.8 Product CRUD & Detail
- **Get Product Detail:** `GET /api/v1/pos/products/:id`
  - **Auth:** Requires permission: `inventory:read`
  - **Response (200 OK):**
    ```json
    {
      "success": true,
      "data": {
        "id": "10000000-0000-0000-0000-000000000011",
        "category_id": "c0000000-0000-0000-0000-000000000010",
        "category_name": "Kue Tart & Custom Cake",
        "sku": "SKU-CAKE-BF20",
        "barcode": "8992001000010",
        "name": "Black Forest Cake 20cm",
        "description": "Kue Black Forest premium dengan dark cherry dan serutan cokelat halal",
        "unit_price": 185000.00,
        "cost_price": 120000.00,
        "stock_quantity": 10,
        "compliance_tags": ["HALAL_MUI"],
        "is_halal_certified": true,
        "is_active": true
      }
    }
    ```

- **Create Product:** `POST /api/v1/pos/products`
  - **Auth:** Requires permission: `inventory:write`
  - **Request Body:**
    ```json
    {
      "name": "Chiffon Pandan Special 20cm",
      "sku": "SKU-CAKE-CF20",
      "barcode": "8992001000099",
      "description": "Bolu chiffon lembut pandan wangi dengan santan murni",
      "category_id": "c0000000-0000-0000-0000-000000000010",
      "unit_price": 55000.00,
      "cost_price": 32000.00,
      "initial_stock": 20,
      "reorder_threshold": 5,
      "warehouse_location": "BAKERY_CHILLER_B",
      "compliance_tags": ["HALAL_MUI"]
    }
    ```
  - **Response (201 Created):** Returns created `Product` object.

- **Update Product:** `PUT /api/v1/pos/products/:id`
  - **Auth:** Requires permission: `inventory:write`
  - **Request Body:** Similar to create product (without SKU and initial stock).
  - **Response (200 OK):** Returns updated `Product` object.

- **Soft Delete Product:** `DELETE /api/v1/pos/products/:id`
  - **Auth:** Requires permission: `inventory:write`
  - **Response (200 OK):** `{"success": true, "data": {"message": "Product soft-deleted successfully", "id": "..."}}`

### 3.9 Category Management
- **List Categories:** `GET /api/v1/pos/categories`
  - **Auth:** Requires permission: `inventory:read`
  - **Response (200 OK):**
    ```json
    {
      "success": true,
      "data": [
        {
          "id": "c0000000-0000-0000-0000-000000000010",
          "name": "Kue Tart & Custom Cake",
          "code": "CAT-CAKE",
          "product_count": 3
        }
      ],
      "meta": {"total": 1}
    }
    ```
- **Create Category:** `POST /api/v1/pos/categories`
  - **Request Body:** `{"name": "Kue Kering Lebaran", "code": "CAT-KERING"}`
  - **Response (201 Created):** Returns created category.
- **Update Category:** `PUT /api/v1/pos/categories/:id`
- **Delete Category:** `DELETE /api/v1/pos/categories/:id`

### 3.10 Inventory Stock Adjustment & Spoilage Write-Off
- **Endpoint:** `POST /api/v1/pos/inventory/adjust`
- **Headers:** `Idempotency-Key: <UUIDv4>`
- **Auth:** Requires permission: `inventory:write`
- **Request Body:**
```json
{
  "product_id": "10000000-0000-0000-0000-000000000011",
  "adjustment_type": "SUBTRACT",
  "quantity": 1,
  "reason": "DAMAGE",
  "notes": "Kue terbentur saat penataan etalase, krim rusak"
}
```
- **Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "adjustment_id": "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d",
    "product_id": "10000000-0000-0000-0000-000000000011",
    "product_name": "Black Forest Cake 20cm",
    "previous_quantity": 10,
    "new_quantity": 9,
    "quantity_delta": -1,
    "reason": "DAMAGE",
    "ledger_entry_number": "JE-ADJ-20260903003211-9b1deb4d",
    "adjusted_at": "2026-09-03T00:32:11Z"
  }
}
```
*(Automatically creates a balanced double-entry journal posting: Debit 5020 Inventory Shrinkage & Loss, Credit 1030 Merchandise Inventory).*

### 3.11 Low Stock Alerts
- **Endpoint:** `GET /api/v1/pos/inventory/low-stock`
- **Auth:** Requires permission: `inventory:read`
- **Response (200 OK):** Returns all products where `stock_quantity <= reorder_threshold` sorted by urgency.

---

## 4. Compliance-Aware Supply Chain Management

### 4.1 Register Supplier (with Optional Compliance Certificate)
- **Endpoint:** `POST /api/v1/supply-chain/suppliers`
- **Auth:** `MANAGER`, `SUPER_ADMIN` (Requires permission: `supply_chain:manage`)
- **Request Body:**
```json
{
  "code": "SUP-AB-02",
  "company_name": "PT Halal Daging Nusantara",
  "contact_person": "Umar Bakri",
  "contact_email": "umar@halaldaging.id",
  "contact_phone": "+628123456789",
  "compliance_certificate": {
    "cert_type": "HALAL_MUI",
    "certificate_number": "ID31110000123450824",
    "issuing_authority": "BPJPH Indonesia",
    "scope": "Fresh and Frozen Beef",
    "valid_from": "2024-01-01",
    "expiry_date": "2026-12-31"
  }
}
```

### 4.2 Create Purchase Order (Enforcing Configurable Compliance)
- **Endpoint:** `POST /api/v1/supply-chain/purchase-orders`
- **Auth:** `MANAGER`, `SUPER_ADMIN` (Requires permission: `supply_chain:manage`)
- **Request Body:**
```json
{
  "supplier_id": "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
  "compliance_cert_id": "c1d2e3f4-a5b6-7c8d-9e0f-1a2b3c4d5e6f",
  "items": [
    {
      "product_id": "10000000-0000-0000-0000-000000000001",
      "quantity": 50,
      "unit_cost": 60000.00
    }
  ]
}
```
- **Error Response if Certificate Expired (422 Unprocessable Entity):**
```json
{
  "success": false,
  "error": {
    "code": "COMPLIANCE_CERT_EXPIRED",
    "message": "Compliance certificate has expired"
  }
}
```
- **Error Response if Certificate Missing under Strict Mode (422 Unprocessable Entity):**
```json
{
  "success": false,
  "error": {
    "code": "COMPLIANCE_CERT_REQUIRED",
    "message": "Compliance certificate is required for this tenant"
  }
}
```

### 4.3 Create Goods Receipt (GR) & Stock Inbound
- **Endpoint:** `POST /api/v1/supply-chain/goods-receipts`
- **Auth:** `MANAGER`, `SUPER_ADMIN` (Requires permission: `supply_chain:manage`)
- **Request Body:**
```json
{
  "purchase_order_id": "po_uuid",
  "notes": "Delivered in good condition",
  "items": [
    {
      "product_id": "10000000-0000-0000-0000-000000000001",
      "received_quantity": 50
    }
  ]
}
```

---

## 5. Sharia Ledger & Financial Reporting

### 5.1 Zakat Tijarah — Planned / Not Registered
- **Status:** No Zakat route is registered in the backend. The following payload is a future-design reference, not an active contract.
- **Endpoint:** `GET /api/v1/ledger/zakat`
- **Auth:** `FINANCIAL_ADMIN`, `MANAGER`
- **Query Params:** `gold_price_per_gram=1350000` (in IDR)
- **Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "calculation_date": "2026-08-30",
    "nisab_gold_price_per_gram": 1350000.00,
    "nisab_threshold_value": 114750000.00,
    "breakdown": {
      "cash_and_bank": 245000000.00,
      "net_receivables": 35000000.00,
      "inventory_valuation": 180000000.00,
      "current_liabilities": 90000000.00
    },
    "zakat_base": 370000000.00,
    "zakat_rate_percentage": 2.5,
    "is_nisab_met": true,
    "zakat_due_amount": 9250000.00
  }
}
```

---

## 6. Continuous AI Auditor — Planned / Not Registered

### 6.1 Retrieve AI Audit Reports
- **Status:** No AI-audit route is registered in the backend. The following payload is a future-design reference, not an active contract.
- **Endpoint:** `GET /api/v1/ai/audit-reports`
- **Auth:** `FINANCIAL_ADMIN`, `COMPLIANCE_OFFICER`
- **Response (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "id": "rpt_99887766-5544-3322-1100-aabbccddeeff",
      "report_code": "AI-AUDIT-2026-W34",
      "audit_period": {
        "start": "2026-08-23",
        "end": "2026-08-30"
      },
      "severity": "WARNING",
      "total_transactions_scanned": 12450,
      "total_anomalies_detected": 3,
      "anomaly_summary": {
        "temporal_anomalies": [
          {
            "description": "Cluster of 14 high-volume cash transactions between 02:00 AM - 04:00 AM",
            "z_score": 3.84,
            "cashier_id": "usr_9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d"
          }
        ],
        "benford_law_divergence": {
          "first_digit_p_value": 0.012,
          "divergent_accounts": ["4010-Sales-Revenue"]
        }
      },
      "recommended_actions": [
        "Review off-hours store access logs for POS Station 02",
        "Verify physical cash deposit receipts matching batch TXN-20260828"
      ]
    }
  ]
}
```

---

### Ledger Module
`GET /api/v1/ledger/accounts`
- **Description:** Retrieve Chart of Accounts.
- **Permissions:** `ledger:read`

`GET /api/v1/ledger/entries`
- **Description:** Retrieve journal entries with pagination.
- **Permissions:** `ledger:read`

`POST /api/v1/ledger/entries`
- **Description:** Create a manual journal entry (MANUAL_ADJUSTMENT only).
- **Permissions:** `ledger:write`

`GET /api/v1/ledger/trial-balance`
- **Description:** Retrieve trial balance as of a specific date.
- **Permissions:** `ledger:read`

---

## 7. Store Manager Analytics & KPI Dashboard

### 7.1 Get Aggregated Store Dashboard
- **Endpoint:** `GET /api/v1/manager/dashboard`
- **Auth:** Bearer Token (Required Roles: `MANAGER`, `SUPER_ADMIN`)
- **Description:** Real-time business aggregations across sales revenue, inventory depletion alerts, Halal certificate expirations, and ledger account status.
- **Success Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "generated_at": "2026-09-02T12:00:00Z",
    "sales_summary": {
      "today_gross_sales": 3450000.00,
      "today_net_sales": 3350000.00,
      "today_orders_count": 48,
      "all_time_orders_count": 1240,
      "average_order_value": 71875.00
    },
    "inventory_alerts": {
      "low_stock_count": 2,
      "items": [
        {
          "product_id": "prod_7c8d9e0f-1a2b-3c4d-5e6f-7a8b9c0d1e2f",
          "sku": "BEEF-RIBEYE-001",
          "name": "Halal Fresh Ribeye Steak 500g",
          "category_name": "Fresh Meat",
          "current_stock": 4,
          "threshold": 10,
          "unit_price": 85000.00
        }
      ]
    },
    "compliance_alerts": {
      "expiring_certificates_count": 1,
      "expired_certificates_count": 0,
      "items": [
        {
          "certificate_id": "cert_3f2b1a0e-9d8c-7b6a-5f4e-3d2c1b0a9f8e",
          "supplier_id": "supp_1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d",
          "supplier_name": "PT Halal Boga Sejahtera",
          "certificate_number": "BPJPH-2026-00192",
          "issuing_authority": "BPJPH",
          "expiry_date": "2026-09-25T00:00:00Z",
          "days_remaining": 23,
          "status": "EXPIRING_SOON"
        }
      ]
    },
    "financial_summary": {
      "active_accounts_count": 12,
      "today_journal_entries_count": 52
    }
  }
}
```
