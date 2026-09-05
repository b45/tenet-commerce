# ADR 002: Money Representation, Rounding, and Bounds

- **Status:** Accepted
- **Date:** 2026-09-05
- **Author:** Tenet Commerce Engineering Team
- **Deciders:** Lead Architect, Frontend Team, Core Backend Team
- **Context Domain:** `frontend/src/lib/money.ts`, `backend/pkg/money/`, `docs/design/POS_CONTRACT_MAP.md`

---

## 1. Context and Problem Statement

Tenet Commerce processes Point-of-Sale checkouts, inventory valuations, goods receipts, and general ledger journal postings. In accordance with Sharia compliance and standard accounting principles:
1. **Zero Tolerance for Floating-Point Inexactness**: Binary floating-point representation (`IEEE 754` float64/float32) suffers from rounding drift (e.g., `0.1 + 0.2 = 0.30000000000000004`). In retail and general ledgers, unbalanced pennies violate double-entry balancing ($\sum \text{Debits} = \sum \text{Credits}$).
2. **Current Backend Implementation**: The backend `pkg/money/money.go` uses an exact integer minor unit (`int64`). For Indonesian Rupiah (`IDR`), 1 minor unit = Rp 1 (IDR has no subunit/cents in active commercial circulation).
3. **Current API Wire DTOs**: Some legacy DTO fields exposed float numbers (identified in `docs/design/POS_CONTRACT_MAP.md` as dependency G-03).
4. **Client-Side Requirements**: The Next.js frontend must calculate cart totals, taxes, discounts, tender amounts, and change with guaranteed consistency matching backend validation.

---

## 2. Decision Drivers

- **Precision & Ledger Balance**: Calculations on the client must produce values that strictly match the backend's `pkg/money/money.go`.
- **Validation Bounds**: Enforce realistic retail and B2B limits to prevent overflow, negative tender, and runaway quantities.
- **Formatting Consistency**: Standardized Indonesian Rupiah presentation (`Rp 40.000` without decimal subunits).
- **Graceful Parsing**: Tolerate user inputs with or without thousand separators (`.`), whitespace, or `Rp` prefixes.

---

## 3. Decision Outcome

### 3.1 Currency & Representation
- **Primary Currency**: `IDR` (Indonesian Rupiah).
- **Internal Storage**: Integer minor units (`number` constrained within `Number.MAX_SAFE_INTEGER`, or `bigint`).
  - In JavaScript, `Number.isSafeInteger()` is valid up to $9 \times 10^{15}$ (9 quadrillion IDR), which exceeds retail requirements while avoiding BigInt serialization friction.
- **Decimals**: IDR operations forbid fractions. No cents or decimal places are allowed in customer-facing inputs (e.g., tender, price adjustments).

### 3.2 System Bounds and Limits
- **Minimum Transaction Amount**: `Rp 0` (or `Rp 1` for paid checkouts).
- **Maximum Line Item Quantity**: `99,999` units per SKU.
- **Maximum Transaction Amount (Cap)**: `Rp 1,000,000,000` (1 Billion IDR) per standard retail POS checkout. Mutations exceeding this cap require supervisor override or enterprise B2B invoice workflows.
- **Maximum Tender Amount**: `Rp 2,000,000,000` (2 Billion IDR).

### 3.3 Rounding Policy
- Tax (PPN) and percentage discounts calculate as:
  $$\text{Tax} = \text{round}\left( \frac{\text{Subtotal} \times \text{Rate}}{100} \right)$$
- Standard mathematical rounding (`Math.round`, round-half-up to nearest Rp 1) is used on both frontend and backend.
- When splitting amounts across lines, remainders are distributed deterministically using the same algorithm as `pkg/money.Split()`.

### 3.4 Wire Format Protocol (Frontend $\leftrightarrow$ Backend)
- **Outbound to Backend (Requests)**:
  - Payloads (`POST /pos/checkout`, `POST /supply-chain/po`) transmit exact integer amounts for minor units (`amount: 45000`) or standard numeric values without fractional exponents.
  - Tender amounts must be integers: `amount_tendered >= total_amount`.
- **Inbound from Backend (Responses)**:
  - Frontend parsing utility `parseMoneyFromAPI(val: unknown): number` normalizes input:
    - If `number`: rounded via `Math.round(val)` if float.
    - If `string`: stripped of non-digit characters except negative sign and parsed via `parseInt(clean, 10)`.

### 3.5 Display Formatting (`lib/money.ts`)
- Use `Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', minimumFractionDigits: 0, maximumFractionDigits: 0 })`.
- Formatted output example: `Rp 40.000` (using non-breaking space `\u00A0` between currency symbol and amount).

---

## 4. Consequences

### Positive:
- Eliminates floating-point calculation discrepancies between POS UI and Go backend validation.
- Clear contract bounds prevent UI freezes, integer overflows, and negative change calculations.
- Seamless compatibility with both current API JSON envelopes and future integer-strict backend schemas.

### Negative / Trade-offs:
- Inputs with commas or decimal dots must be sanitized or rejected with clear validation messages (`"Nominal harus berupa angka bulat rupiah"`).
