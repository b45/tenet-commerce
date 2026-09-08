/**
 * Tenet Commerce — Enterprise Distributed Tracing & Audit Trail Logger
 * Propagates trace_id, device telemetry, and audit events to Grafana Loki / Tempo.
 */

import { getDeviceTelemetry, type DeviceTelemetry } from "./device";

export type LogLevel = "debug" | "info" | "warn" | "error" | "audit";

export interface LogPayload {
  timestamp: string;
  level: LogLevel;
  message: string;
  trace_id: string;
  session_id?: string;
  tenant_slug?: string;
  user_id?: string;
  user_email?: string;
  path?: string;
  device?: DeviceTelemetry;
  context?: Record<string, unknown>;
  error?: {
    name?: string;
    message?: string;
    stack?: string;
  };
}

class TelemetryLogger {
  private activeTraceId: string | null = null;
  private sessionId: string;
  private tenantSlug: string | null = null;
  private userId: string | null = null;
  private userEmail: string | null = null;

  constructor() {
    this.sessionId = this.generateId("sess");
    if (typeof window !== "undefined") {
      const stored = sessionStorage.getItem("tc_session_id");
      if (stored) {
        this.sessionId = stored;
      } else {
        sessionStorage.setItem("tc_session_id", this.sessionId);
      }
    }
  }

  public generateId(prefix: string = "trace"): string {
    if (typeof crypto !== "undefined" && crypto.randomUUID) {
      return `${prefix}_${crypto.randomUUID()}`;
    }
    const rand = Math.random().toString(36).substring(2, 10);
    const ts = Date.now().toString(36);
    return `${prefix}_${ts}_${rand}`;
  }

  public getOrCreateTraceId(): string {
    if (!this.activeTraceId) {
      this.activeTraceId = this.generateId("tr");
    }
    return this.activeTraceId;
  }

  public renewTraceId(): string {
    this.activeTraceId = this.generateId("tr");
    return this.activeTraceId;
  }

  public setUserContext(tenantSlug: string, userId: string, userEmail?: string) {
    this.tenantSlug = tenantSlug;
    this.userId = userId;
    if (userEmail) this.userEmail = userEmail;
  }

  public clearUserContext() {
    this.tenantSlug = null;
    this.userId = null;
    this.userEmail = null;
  }

  public getDiagnosticSnapshot() {
    return {
      trace_id: this.getOrCreateTraceId(),
      session_id: this.sessionId,
      tenant_slug: this.tenantSlug,
      user_id: this.userId,
      user_email: this.userEmail,
      timestamp: new Date().toISOString(),
      device: getDeviceTelemetry(),
    };
  }

  private createPayload(
    level: LogLevel,
    message: string,
    context?: Record<string, unknown>,
    err?: unknown
  ): LogPayload {
    const errorObj =
      err instanceof Error
        ? {
            name: err.name,
            message: err.message,
            stack: err.stack,
          }
        : undefined;

    return {
      timestamp: new Date().toISOString(),
      level,
      message,
      trace_id: this.getOrCreateTraceId(),
      session_id: this.sessionId,
      tenant_slug: this.tenantSlug ?? undefined,
      user_id: this.userId ?? undefined,
      user_email: this.userEmail ?? undefined,
      path: typeof window !== "undefined" ? window.location.pathname : undefined,
      device: getDeviceTelemetry(),
      context,
      error: errorObj,
    };
  }

  private emit(payload: LogPayload) {
    const isDev = process.env.NODE_ENV !== "production";

    // 1. Console Output with clear visual trace & device badges
    if (isDev) {
      const badgeStyle = "background: #0066CC; color: white; padding: 1px 5px; border-radius: 4px; font-weight: bold; font-size: 10px;";
      const traceStyle = "color: #888; font-size: 10px;";
      const deviceTag = payload.device ? `[${payload.device.os} | ${payload.device.browser}]` : "";

      /* eslint-disable no-console */
      if (payload.level === "error") {
        console.error(`%cTC%c [${payload.trace_id}] ${deviceTag} ${payload.message}`, badgeStyle, traceStyle, payload.context || "", payload.error || "");
      } else if (payload.level === "warn") {
        console.warn(`%cTC%c [${payload.trace_id}] ${deviceTag} ${payload.message}`, badgeStyle, traceStyle, payload.context || "");
      } else if (payload.level === "audit") {
        console.info(`%cAUDIT%c [${payload.trace_id}] ${deviceTag} ${payload.message}`, "background: #10B981; color: white; padding: 1px 5px; border-radius: 4px; font-weight: bold; font-size: 10px;", traceStyle, payload.context || "");
      } else {
        console.log(`%cTC%c [${payload.trace_id}] ${deviceTag} ${payload.message}`, badgeStyle, traceStyle, payload.context || "");
      }
      /* eslint-enable no-console */
    }

    // 2. Dispatch telemetry to ingestion endpoint for errors and audit trails
    if (typeof window !== "undefined" && (payload.level === "error" || payload.level === "audit" || payload.level === "warn")) {
      try {
        navigator.sendBeacon?.(
          "/api/telemetry/logs",
          JSON.stringify(payload)
        );
      } catch {
        // Silently ignore telemetry transmission errors
      }
    }
  }

  public debug(message: string, context?: Record<string, unknown>) {
    this.emit(this.createPayload("debug", message, context));
  }

  public info(message: string, context?: Record<string, unknown>) {
    this.emit(this.createPayload("info", message, context));
  }

  public warn(message: string, context?: Record<string, unknown>) {
    this.emit(this.createPayload("warn", message, context));
  }

  public error(message: string, err?: unknown, context?: Record<string, unknown>) {
    this.emit(this.createPayload("error", message, context, err));
  }

  /**
   * Records high-value enterprise audit events (logins, checkouts, voids, adjustments)
   */
  public audit(action: string, metadata?: Record<string, unknown>) {
    this.emit(this.createPayload("audit", `AUDIT_EVENT: ${action}`, metadata));
  }
}

export const logger = new TelemetryLogger();
