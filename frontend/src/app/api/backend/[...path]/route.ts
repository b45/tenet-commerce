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

async function proxyRequest(request: NextRequest, context: { params: Promise<{ path: string[] }> }) {
  const { path } = await context.params;
  const backendPath = path.join("/");
  const targetUrl = new URL(`${BACKEND_URL}/api/v1/${backendPath}`);

  // Forward query strings
  request.nextUrl.searchParams.forEach((value, key) => {
    targetUrl.searchParams.set(key, value);
  });

  // Client telemetry & IP resolution
  const clientIp = resolveClientIp(request);
  const userAgent = request.headers.get("user-agent") || "Web-POS";
  const traceId = request.headers.get("X-Trace-ID") || `trace_${crypto.randomUUID()}`;

  const accessToken = request.cookies.get("tenet_access_token")?.value;
  const tenantSlug = request.cookies.get("tenet_tenant_slug")?.value;

  const forwardHeaders = new Headers();
  forwardHeaders.set("Content-Type", request.headers.get("Content-Type") || "application/json");
  forwardHeaders.set("X-Trace-ID", traceId);
  forwardHeaders.set("X-Tenet-Client", "Web-POS-BFF");
  forwardHeaders.set("X-Forwarded-For", clientIp);
  forwardHeaders.set("X-Real-IP", clientIp);
  forwardHeaders.set("User-Agent", userAgent);

  if (accessToken) {
    forwardHeaders.set("Authorization", `Bearer ${accessToken}`);
  }
  if (tenantSlug) {
    forwardHeaders.set("X-Tenant-Slug", tenantSlug);
  }

  // Forward mutation idempotency & tenant context
  const customHeaders = ["Idempotency-Key", "X-Tenant-ID"];
  for (const h of customHeaders) {
    const val = request.headers.get(h);
    if (val) forwardHeaders.set(h, val);
  }

  let body: BodyInit | null = null;
  if (["POST", "PUT", "PATCH"].includes(request.method)) {
    try {
      body = await request.text();
    } catch {
      body = null;
    }
  }

  const startTime = Date.now();
  try {
    const res = await fetch(targetUrl.toString(), {
      method: request.method,
      headers: forwardHeaders,
      body,
    });

    const responseBody = await res.text();
    const duration = Date.now() - startTime;

    // Enterprise structured audit log for Grafana Loki / Tempo
    // eslint-disable-next-line no-console
    console.log(JSON.stringify({
      source: "tenet_bff_gateway",
      level: res.ok ? "info" : "warn",
      timestamp: new Date().toISOString(),
      trace_id: traceId,
      client_ip: clientIp,
      user_agent: userAgent,
      method: request.method,
      path: backendPath,
      status: res.status,
      duration_ms: duration,
      tenant_slug: tenantSlug || "anonymous",
    }));

    const headers = new Headers();
    const contentType = res.headers.get("content-type");
    if (contentType) headers.set("content-type", contentType);

    const serverTraceId = res.headers.get("X-Trace-ID") || traceId;
    headers.set("X-Trace-ID", serverTraceId);

    return new NextResponse(responseBody, {
      status: res.status,
      headers,
    });
  } catch (err: unknown) {
    // eslint-disable-next-line no-console
    console.error(JSON.stringify({
      source: "tenet_bff_gateway",
      level: "error",
      timestamp: new Date().toISOString(),
      trace_id: traceId,
      client_ip: clientIp,
      user_agent: userAgent,
      method: request.method,
      path: backendPath,
      error: err instanceof Error ? err.message : "Network error",
    }));

    const headers = new Headers();
    headers.set("X-Trace-ID", traceId);

    return NextResponse.json(
      {
        success: false,
        error: {
          code: "BACKEND_UNREACHABLE",
          message: "Tidak dapat terhubung ke backend server Tenet Commerce.",
          trace_id: traceId,
        },
      },
      { status: 502, headers }
    );
  }
}

export async function GET(req: NextRequest, ctx: { params: Promise<{ path: string[] }> }) {
  return proxyRequest(req, ctx);
}

export async function POST(req: NextRequest, ctx: { params: Promise<{ path: string[] }> }) {
  return proxyRequest(req, ctx);
}

export async function PUT(req: NextRequest, ctx: { params: Promise<{ path: string[] }> }) {
  return proxyRequest(req, ctx);
}

export async function DELETE(req: NextRequest, ctx: { params: Promise<{ path: string[] }> }) {
  return proxyRequest(req, ctx);
}
