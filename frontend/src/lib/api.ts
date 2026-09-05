/**
 * Tenet Commerce Unified HTTP Client
 * Communicates with Next.js BFF Route Handlers with Distributed Tracing (X-Trace-ID).
 */

import { logger } from "./logger";

export interface ApiResponse<T = unknown> {
  success: boolean;
  data: T;
  error?: {
    code: string;
    message: string;
    details?: unknown;
  } | null;
  meta?: {
    page?: number;
    limit?: number;
    total?: number;
    timestamp?: string;
  };
}

export interface UserProfile {
  id: string;
  email: string;
  full_name: string;
  role: "CASHIER" | "MANAGER" | "COMPLIANCE_OFFICER" | "FINANCIAL_ADMIN" | "SUPER_ADMIN" | string;
  tenant_slug: string;
  permissions: string[];
}

export class ApiError extends Error {
  code: string;
  status: number;
  traceId?: string;
  details?: unknown;

  constructor(
    message: string,
    code: string = "INTERNAL_ERROR",
    status: number = 500,
    traceId?: string,
    details?: unknown
  ) {
    super(message);
    this.name = "ApiError";
    this.code = code;
    this.status = status;
    this.traceId = traceId;
    this.details = details;
  }
}

export class ValidationError extends ApiError {
  constructor(message: string, traceId?: string, details?: unknown) {
    super(message, "VALIDATION_FAILED", 400, traceId, details);
    this.name = "ValidationError";
  }
}

export class AuthError extends ApiError {
  constructor(message: string, code: string = "UNAUTHORIZED", status: number = 401, traceId?: string) {
    super(message, code, status, traceId);
    this.name = "AuthError";
  }
}

export class ConflictError extends ApiError {
  constructor(message: string, code: string = "CONFLICT", traceId?: string, details?: unknown) {
    super(message, code, 409, traceId, details);
    this.name = "ConflictError";
  }
}

export class NetworkError extends ApiError {
  constructor(traceId?: string, message: string = "Koneksi ke server terputus. Periksa jaringan Anda.") {
    super(message, "NETWORK_ERROR", 0, traceId);
    this.name = "NetworkError";
  }
}

/**
 * Universal fetch wrapper for frontend components.
 * Automatically injects X-Trace-ID and targets internal BFF routes (`/api/backend/*` or `/api/auth/*`).
 */
export async function apiFetch<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const url = endpoint.startsWith("http") ? endpoint : endpoint.startsWith("/") ? endpoint : `/${endpoint}`;

  // 1. Trace ID generation and propagation
  const traceId = logger.renewTraceId();

  const headers = new Headers(options.headers || {});
  if (!headers.has("Content-Type") && options.body && typeof options.body === "string") {
    headers.set("Content-Type", "application/json");
  }
  headers.set("X-Tenet-Client", "Web-POS");
  headers.set("X-Trace-ID", traceId);

  let res: Response;
  const startTime = Date.now();

  try {
    res = await fetch(url, {
      ...options,
      headers,
      credentials: "same-origin",
    });
  } catch (networkErr) {
    logger.error(`Network fault contacting ${url}`, networkErr, { trace_id: traceId });
    throw new NetworkError(traceId);
  }

  const durationMs = Date.now() - startTime;
  // Resolve returned trace_id from Go backend or Next.js BFF response headers
  const serverTraceId = res.headers.get("X-Trace-ID") || traceId;

  let body: ApiResponse<T> | null = null;
  try {
    body = await res.json();
  } catch {
    if (!res.ok) {
      const err = new ApiError(`HTTP Error ${res.status}: ${res.statusText}`, "HTTP_ERROR", res.status, serverTraceId);
      logger.error(`Non-JSON Error ${res.status}`, err, { url, durationMs, trace_id: serverTraceId });
      throw err;
    }
    return (null as unknown) as T;
  }

  if (!res.ok || (body && body.success === false)) {
    const errCode = body?.error?.code || `HTTP_${res.status}`;
    const errMsg = body?.error?.message || res.statusText || "Terjadi kesalahan pada sistem";
    const details = body?.error?.details;

    let errorToThrow: ApiError;
    if (res.status === 400) errorToThrow = new ValidationError(errMsg, serverTraceId, details);
    else if (res.status === 401 || res.status === 403) errorToThrow = new AuthError(errMsg, errCode, res.status, serverTraceId);
    else if (res.status === 409) errorToThrow = new ConflictError(errMsg, errCode, serverTraceId, details);
    else errorToThrow = new ApiError(errMsg, errCode, res.status, serverTraceId, details);

    logger.error(`API rejection on ${url}`, errorToThrow, {
      status: res.status,
      durationMs,
      code: errCode,
      trace_id: serverTraceId,
      details,
    });

    throw errorToThrow;
  }

  logger.debug(`API request successful: ${url}`, {
    durationMs,
    trace_id: serverTraceId,
    status: res.status,
  });

  return body?.data !== undefined ? body.data : ((body as unknown) as T);
}

/**
 * Client authentication endpoints
 */
export const authApi = {
  async login(payload: { tenant_slug: string; email: string; password: string }): Promise<UserProfile> {
    const data = await apiFetch<{ user: UserProfile }>("/api/auth/login", {
      method: "POST",
      body: JSON.stringify(payload),
    });
    logger.setUserContext(data.user.tenant_slug, data.user.id);
    return data.user;
  },

  async me(): Promise<UserProfile> {
    const data = await apiFetch<{ user: UserProfile }>("/api/auth/me");
    if (data?.user) {
      logger.setUserContext(data.user.tenant_slug, data.user.id);
    }
    return data.user;
  },

  async logout(): Promise<void> {
    logger.clearUserContext();
    await apiFetch<{ success: boolean }>("/api/auth/logout", {
      method: "POST",
    });
  },
};
