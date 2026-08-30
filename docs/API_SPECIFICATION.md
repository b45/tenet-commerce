# REST API Specification & Endpoint Contracts
## Tenet Commerce: Enterprise POS & Halal Supply Chain API

---

## 1. Global API Standards & Protocols

### 1.1 Base URL & Content Negotiation
- **Base URL:** `https://api.tenet-commerce.internal/api/v1`
- **Content-Type:** `application/json; charset=utf-8`
- **Protocol:** HTTP/2 over TLS 1.3

### 1.2 Required Request Headers
```http
Authorization: Bearer <JWT_ACCESS_TOKEN>
Idempotency-Key: <UUIDv4>             # Mandatory on all POST / PUT mutating requests
X-Tenant-ID: <TENANT_SLUG_OR_UUID>     # Optional override (default extracted from JWT)
```

### 1.3 Standard Response Envelope
```json
{
  "success": true,
  "data": {},
  "meta": {
    "timestamp": "2026-08-30T10:00:00Z",
    "request_id": "req_8f12c3e4-a1b2-4c3d-8e5f-123456789abc",
    "page": 1,
    "limit": 50,
    "total": 120
  }
}
```

### 1.4 Standard Error Envelope
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

## 2. Authentication & Tenant Management

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

### 2.2 Provision New Tenant
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

### 3.1 List Products & Inventory
- **Endpoint:** `GET /api/v1/products`
- **Auth:** `CASHIER`, `MANAGER`, `ADMIN`
- **Query Params:** `query=halal`, `category_id=<UUID>`, `limit=50`
- **Response (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "id": "prod_1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d",
      "sku": "SKU-HALAL-BEEF-01",
      "barcode": "8992753123456",
      "name": "Wagyu Halal Ribeye Cut 250g",
      "unit_price": 185000.00,
      "stock_quantity": 42,
      "is_halal_certified": true
    }
  ]
}
```

### 3.2 Idempotent POS Checkout
- **Endpoint:** `POST /api/v1/transactions`
- **Headers:** `Idempotency-Key: 7b8a1c9e-2f3a-4b5c-6d7e-8f9a0b1c2d3e`
- **Auth:** `CASHIER`, `MANAGER`
- **Request Body:**
```json
{
  "cashier_id": "usr_9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d",
  "payment_method": "CASH",
  "items": [
    {
      "product_id": "prod_1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d",
      "quantity": 2,
      "unit_price": 185000.00
    }
  ],
  "discount_amount": 0.00,
  "tax_amount": 37000.00,
  "total_amount": 407000.00
}
```
- **Response (201 Created):**
```json
{
  "success": true,
  "data": {
    "transaction_id": "txn_3f4a5b6c-7d8e-9f0a-1b2c-3d4e5f6a7b8c",
    "transaction_number": "TXN-20260830-0042",
    "idempotency_key": "7b8a1c9e-2f3a-4b5c-6d7e-8f9a0b1c2d3e",
    "status": "COMPLETED",
    "total_amount": 407000.00,
    "ledger_entry_number": "JRNL-20260830-0089",
    "created_at": "2026-08-30T10:15:30Z"
  }
}
```

---

## 4. Halal Supply Chain Management

### 4.1 Register Supplier with Halal Certificate
- **Endpoint:** `POST /api/v1/suppliers`
- **Auth:** `MANAGER`, `COMPLIANCE_OFFICER`
- **Request Body:**
```json
{
  "company_name": "PT Halal Daging Nusantara",
  "code": "SUPP-HDN-001",
  "contact_person": "Umar Bakri",
  "contact_email": "umar@halaldaging.id",
  "certificate": {
    "certificate_number": "ID31110000123450824",
    "issuing_authority": "BPJPH Indonesia",
    "scope": "Fresh and Frozen Beef Slaughtering",
    "valid_from": "2024-08-01",
    "expiry_date": "2028-08-01",
    "document_url": "https://storage.tenet.internal/certs/ID31110000123450824.pdf"
  }
}
```

### 4.2 Create Purchase Order (with Hard-Validation)
- **Endpoint:** `POST /api/v1/purchase-orders`
- **Auth:** `MANAGER`, `COMPLIANCE_OFFICER`
- **Request Body:**
```json
{
  "supplier_id": "sup_11223344-5566-7788-99aa-bbccddeeff00",
  "items": [
    {
      "product_id": "prod_1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d",
      "quantity": 50,
      "unit_cost": 140000.00
    }
  ]
}
```
- **Error Response if Certificate Expired (422 Unprocessable Entity):**
```json
{
  "success": false,
  "error": {
    "code": "HALAL_CERTIFICATE_EXPIRED",
    "message": "Cannot create Purchase Order: Supplier certificate expired on 2026-08-01",
    "details": {
      "supplier_id": "sup_11223344-5566-7788-99aa-bbccddeeff00",
      "expiry_date": "2026-08-01",
      "authority": "BPJPH"
    }
  }
}
```

---

## 5. Sharia Ledger & Financial Reporting

### 5.1 Query Real-Time Zakat Tijarah
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

## 6. Continuous AI Auditor Endpoints

### 6.1 Retrieve AI Audit Reports
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

*Tenet Commerce — REST API Specification v1.0.0*
