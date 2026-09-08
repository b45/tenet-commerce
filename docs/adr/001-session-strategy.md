# ADR 001: BFF cookie sessions

Status: accepted direction, reconciled against source `203257a`.

The browser calls `/api/auth/login`, `/api/auth/refresh`, `/api/auth/logout`, `/api/auth/me` and `/api/backend/[...path]`. The [auth handler](../../frontend/src/app/api/auth/[action]/route.ts) sets HttpOnly access/refresh cookies with conditional Secure attributes. Cookies are not additionally encrypted by this handler. The tenant slug cookie is not an authorization credential.

Keeping bearer tokens in JavaScript-readable storage was rejected because scripts could extract them. HttpOnly reduces direct token access but does not eliminate XSS-driven authenticated requests. Origin/referer checks in the [BFF proxy](../../frontend/src/app/api/backend/[...path]/route.ts) provide mutation provenance checks; the API still authenticates and authorizes the request.

The [client](../../frontend/src/lib/api.ts) attempts refresh and one retry after eligible 401 responses. This is client-driven recovery, not automatic refresh inside every proxy call. A failed session probe redirects the dashboard to login. The current logout flow clears disposable IndexedDB state; accepted offline paid commands require a separate preservation and ownership policy before enablement.

The BFF adds a server hop and cookie/CSRF responsibilities compared with direct API calls. Future migration must preserve refresh-cookie path, logout invalidation and tenant isolation. Secure-cookie production behavior requires deployment testing; local HTTP development is not evidence of a working HTTPS production session.

See [architecture evidence](../ARCHITECTURE_EVIDENCE.md) and [Phase 3 design](../FRONTEND_PHASE3_DESIGN.md).
