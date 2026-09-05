# ADR 001: Session & Authentication Architecture Strategy

- **Status:** Accepted
- **Date:** 2026-09-05
- **Author:** Tenet Commerce Engineering Team
- **Deciders:** Lead Architect, Frontend Team, Security Compliance
- **Context Domain:** `frontend/`, `docs/design/`, `backend/pkg/auth/`

---

## 1. Context and Problem Statement

Tenet Commerce is a multi-tenant retail and enterprise platform handling financial transactions, Point-of-Sale (POS) cash checkouts, and Sharia-compliant general ledger journals. The backend provides JWT-based authentication via `/auth/login`, `/auth/refresh`, and `/auth/me` (`backend/pkg/auth/jwt.go`), returning:
- An **Access Token** (short-lived, 15 minutes) carrying `sub`, `tenant_id`, `tenant_slug`, `role`, and `permissions[]`.
- A **Refresh Token** (long-lived, 7 days).

Because the frontend is used in high-concurrency retail environments and shared physical terminals (cashier counters, warehouse receiving stations, financial management workstations), session management must strictly protect against:
1. **Token Exfiltration via XSS**: If third-party dependencies or injected scripts can read raw JWT tokens from browser storage (`localStorage`), entire tenant accounts can be hijacked.
2. **Terminal Session Residuals**: Terminals left open or refreshed must have deterministic expiration, rotation, and logout behaviors.
3. **Cross-Site Request Forgery (CSRF)**: When using browser cookies, cross-site mutations must be prevented.

---

## 2. Decision Drivers

- **Security & Integrity**: Prevent credential theft and unauthorized financial mutations (POS checkout, void, journal entries).
- **Multi-Tenant Isolation**: Ensure tenant slug and ID are validated cryptographically on every request without client-side tampering.
- **Developer Experience & Performance**: Seamless token refresh without abrupt cashier logout during active checkout.
- **Offline / Edge Compatibility**: Clear separation between online authenticated operations and offline draft persistence (IndexedDB).

---

## 3. Considered Options

### Option A: Direct Client-Side Token Storage (`localStorage` / `sessionStorage`)
The browser app calls the Go API directly (`http://localhost:8080`). Tokens are saved in `localStorage` and attached via `Authorization: Bearer <token>` in `fetch()`.

- *Pros:*
  - Zero backend-for-frontend (BFF) proxy overhead.
  - Extremely simple to implement initial prototypes.
- *Cons:*
  - **Severe Security Risk:** Fully vulnerable to Cross-Site Scripting (XSS). Any script running in the browser can steal the long-lived refresh token.
  - Unacceptable for enterprise financial and compliance portfolios.

### Option B: Next.js Backend-for-Frontend (BFF) Proxy with `httpOnly` Secure Cookies (Chosen)
The frontend leverages Next.js App Router Route Handlers (`app/api/auth/*`) as a dedicated BFF authentication proxy. 

- When the user logs in, Next.js calls the Go backend, receives tokens, and stores them in encrypted `httpOnly`, `Secure`, `SameSite=Lax` cookies.
- Browser JavaScript **never** has direct access to the raw JWT strings.
- Outgoing API calls made from Next.js Server Components or Route Handlers attach the token directly.
- Client-side fetch requests go through lightweight Next.js proxy handlers (`/api/backend/*`) or Server Actions that inject the `Authorization: Bearer` header before communicating with the Go API.

- *Pros:*
  - **XSS Resilience:** JavaScript cannot read `httpOnly` cookies, preventing credential theft.
  - **Automatic SSR Hydration:** Next.js Server Components can immediately read session context and tenant routing before sending HTML to the client.
  - **Clean Terminal Teardown:** Clearing cookies completely invalidates the terminal session.
- *Cons:*
  - Slight latency addition when proxying requests through Next.js server runtime.
  - Requires CSRF mitigation for state-mutating requests (`SameSite=Lax` + custom anti-CSRF request header `X-Requested-With: XMLHttpRequest` or origin validation).

### Option C: In-Memory Client Storage with Web Worker Refresh
Tokens are kept strictly in JavaScript memory; a background Web Worker handles silent refreshing.

- *Pros:*
  - No localStorage exposure; direct Go API calls.
- *Cons:*
  - State is wiped on every browser refresh or navigation, requiring complex fallback mechanisms.
  - High operational complexity for a solo maintainer.

---

## 4. Decision Outcome

**Chosen Option:** **Option B (Next.js BFF with `httpOnly` Secure Cookies)**.

### Architectural Blueprint:

1. **Authentication Endpoints in Next.js**:
   - `POST /api/auth/login`: Forwards `{ tenant_slug, email, password }` to Go API `POST /auth/login`. On 200, sets:
     - `tenet_access_token` (`HttpOnly`, `Path=/`, `SameSite=Lax`, `Max-Age=900`)
     - `tenet_refresh_token` (`HttpOnly`, `Path=/api/auth`, `SameSite=Lax`, `Max-Age=604800`)
     - Returns `{ user: { id, email, role, permissions, tenant_slug } }` (excluding token strings) to client state.
   - `POST /api/auth/refresh`: Reads `tenet_refresh_token` cookie, calls Go API `POST /auth/refresh`, and updates the `tenet_access_token` cookie.
   - `POST /api/auth/logout`: Clears both cookies and invalidates the session.
   - `GET /api/auth/session`: Returns currently authenticated user context without exposing token strings.

2. **Client-Side HTTP Client (`frontend/src/lib/api.ts`)**:
   - For protected domain requests, the client calls `/api/proxy/[...path]` or calls Server Actions.
   - Proxy handler validates cookie, injects `Authorization: Bearer <access_token>` and `X-Tenant-ID`, and streams responses from Go API `http://backend:8080`.
   - On 401 response from Go API, the BFF proxy attempts automatic refresh via `tenet_refresh_token`. If successful, the original request is retried transparently. If refresh fails, cookies are cleared and 401 is returned to redirect to `/login`.

3. **CSRF & Origin Protection**:
   - All mutating requests (`POST`, `PUT`, `DELETE`) require custom header `X-Tenet-Client: Web-POS`.
   - Next.js proxy verifies that `Origin` or `Referer` headers match the active application host.

---

## 5. Consequences

### Positive:
- Adheres to OWASP Top 10 and enterprise security baselines for financial and POS software.
- Full compliance with zero-trust tenant isolation: client JavaScript cannot manipulate tenant claims or bearer tokens.
- Next.js Server Components can perform authenticated data pre-fetching during SSR without client flash.

### Negative / Trade-offs:
- Requires maintenance of Next.js Route Handlers in `frontend/src/app/api/`.
- Local development requires both Go API (`:8080`) and Next.js (`:3000`) running simultaneously.
