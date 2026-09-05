import { NextRequest, NextResponse } from "next/server";
import { getBackendUrl } from "@/lib/config";

const BACKEND_URL = getBackendUrl();

function resolveClientIp(req: NextRequest): string {
  const forwarded = req.headers.get("x-forwarded-for");
  if (forwarded) {
    const first = forwarded.split(",")[0]?.trim();
    if (first) return first;
  }
  return (
    req.headers.get("x-real-ip") ||
    req.headers.get("cf-connecting-ip") ||
    "127.0.0.1"
  );
}

export async function POST(
  request: NextRequest,
  context: { params: Promise<{ action: string }> }
) {
  const { action } = await context.params;
  const traceId = request.headers.get("X-Trace-ID") || `auth_${crypto.randomUUID()}`;
  const clientIp = resolveClientIp(request);
  const userAgent = request.headers.get("user-agent") || "Web-POS-Auth";

  if (action === "login") {
    try {
      const body = await request.json();
      const startTime = Date.now();

      const backendRes = await fetch(`${BACKEND_URL}/api/v1/auth/login`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Trace-ID": traceId,
          "X-Forwarded-For": clientIp,
          "X-Real-IP": clientIp,
          "User-Agent": userAgent,
        },
        body: JSON.stringify(body),
      });

      const data = await backendRes.json();
      const serverTraceId = backendRes.headers.get("X-Trace-ID") || traceId;
      const durationMs = Date.now() - startTime;

      // Enterprise audit log for Grafana Loki
      // eslint-disable-next-line no-console
      console.log(JSON.stringify({
        source: "tenet_auth_audit",
        action: "USER_LOGIN_ATTEMPT",
        timestamp: new Date().toISOString(),
        trace_id: serverTraceId,
        client_ip: clientIp,
        user_agent: userAgent,
        tenant_slug: body.tenant_slug || "unknown",
        email: body.email,
        success: backendRes.ok && data.success,
        status: backendRes.status,
        duration_ms: durationMs,
      }));

      if (!backendRes.ok || !data.success) {
        const errRes = NextResponse.json(
          data || { success: false, error: { message: "Login failed" } },
          { status: backendRes.status }
        );
        errRes.headers.set("X-Trace-ID", serverTraceId);
        return errRes;
      }

      const { access_token, refresh_token, user } = data.data;

      const response = NextResponse.json({
        success: true,
        data: { user },
      });
      response.headers.set("X-Trace-ID", serverTraceId);

      const isProd = process.env.NODE_ENV === "production";
      response.cookies.set("tenet_access_token", access_token, {
        httpOnly: true,
        secure: isProd,
        sameSite: "lax",
        path: "/",
        maxAge: 15 * 60,
      });

      response.cookies.set("tenet_refresh_token", refresh_token, {
        httpOnly: true,
        secure: isProd,
        sameSite: "lax",
        path: "/api/auth",
        maxAge: 7 * 24 * 60 * 60,
      });

      response.cookies.set("tenet_tenant_slug", user.tenant_slug, {
        httpOnly: false,
        secure: isProd,
        sameSite: "lax",
        path: "/",
        maxAge: 7 * 24 * 60 * 60,
      });

      return response;
    } catch {
      const errRes = NextResponse.json(
        { success: false, error: { code: "BFF_ERROR", message: "Gagal menghubungi auth service", trace_id: traceId } },
        { status: 502 }
      );
      errRes.headers.set("X-Trace-ID", traceId);
      return errRes;
    }
  }

  if (action === "refresh") {
    const refreshToken = request.cookies.get("tenet_refresh_token")?.value;
    if (!refreshToken) {
      return NextResponse.json(
        { success: false, error: { code: "NO_REFRESH_TOKEN", message: "Sesi kedaluwarsa", trace_id: traceId } },
        { status: 401 }
      );
    }

    try {
      const backendRes = await fetch(`${BACKEND_URL}/api/v1/auth/refresh`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Trace-ID": traceId,
          "X-Forwarded-For": clientIp,
          "X-Real-IP": clientIp,
          "User-Agent": userAgent,
        },
        body: JSON.stringify({ refresh_token: refreshToken }),
      });

      const data = await backendRes.json();
      const serverTraceId = backendRes.headers.get("X-Trace-ID") || traceId;

      if (!backendRes.ok || !data.success) {
        const res = NextResponse.json(data, { status: backendRes.status });
        res.headers.set("X-Trace-ID", serverTraceId);
        res.cookies.delete("tenet_access_token");
        res.cookies.delete("tenet_refresh_token");
        return res;
      }

      const { access_token } = data.data;
      const res = NextResponse.json({ success: true });
      res.headers.set("X-Trace-ID", serverTraceId);
      res.cookies.set("tenet_access_token", access_token, {
        httpOnly: true,
        secure: process.env.NODE_ENV === "production",
        sameSite: "lax",
        path: "/",
        maxAge: 15 * 60,
      });
      return res;
    } catch {
      const errRes = NextResponse.json(
        { success: false, error: { code: "BFF_ERROR", message: "Gagal memperbarui token", trace_id: traceId } },
        { status: 502 }
      );
      errRes.headers.set("X-Trace-ID", traceId);
      return errRes;
    }
  }

  if (action === "logout") {
    // eslint-disable-next-line no-console
    console.log(JSON.stringify({
      source: "tenet_auth_audit",
      action: "USER_LOGOUT",
      timestamp: new Date().toISOString(),
      trace_id: traceId,
      client_ip: clientIp,
      user_agent: userAgent,
    }));

    const response = NextResponse.json({ success: true });
    response.headers.set("X-Trace-ID", traceId);
    response.cookies.delete("tenet_access_token");
    response.cookies.delete("tenet_refresh_token");
    response.cookies.delete("tenet_tenant_slug");
    return response;
  }

  return NextResponse.json({ error: "Action not found" }, { status: 404 });
}

export async function GET(
  request: NextRequest,
  context: { params: Promise<{ action: string }> }
) {
  const { action } = await context.params;
  const traceId = request.headers.get("X-Trace-ID") || `auth_${crypto.randomUUID()}`;
  const clientIp = resolveClientIp(request);
  const userAgent = request.headers.get("user-agent") || "Web-POS-Auth";

  if (action === "me") {
    const accessToken = request.cookies.get("tenet_access_token")?.value;
    if (!accessToken) {
      return NextResponse.json(
        { success: false, error: { code: "UNAUTHORIZED", message: "Belum login", trace_id: traceId } },
        { status: 401 }
      );
    }

    try {
      const backendRes = await fetch(`${BACKEND_URL}/api/v1/auth/me`, {
        headers: {
          Authorization: `Bearer ${accessToken}`,
          "X-Trace-ID": traceId,
          "X-Forwarded-For": clientIp,
          "X-Real-IP": clientIp,
          "User-Agent": userAgent,
        },
      });

      const data = await backendRes.json();
      const serverTraceId = backendRes.headers.get("X-Trace-ID") || traceId;

      if (!backendRes.ok) {
        const res = NextResponse.json(data, { status: backendRes.status });
        res.headers.set("X-Trace-ID", serverTraceId);
        return res;
      }

      const res = NextResponse.json({
        success: true,
        data: { user: data.data },
      });
      res.headers.set("X-Trace-ID", serverTraceId);
      return res;
    } catch {
      const errRes = NextResponse.json(
        { success: false, error: { code: "BFF_ERROR", message: "Gagal mengambil profil user", trace_id: traceId } },
        { status: 502 }
      );
      errRes.headers.set("X-Trace-ID", traceId);
      return errRes;
    }
  }

  return NextResponse.json({ error: "Not found" }, { status: 404 });
}
