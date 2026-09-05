import { NextRequest, NextResponse } from "next/server";

export async function POST(request: NextRequest) {
  try {
    const raw = await request.text();
    const payload = JSON.parse(raw);

    // Format structured log entry matching Grafana Loki / Tempo ingestion
    // eslint-disable-next-line no-console
    console.log(
      JSON.stringify({
        source: "frontend_client_telemetry",
        timestamp: payload.timestamp || new Date().toISOString(),
        level: payload.level || "info",
        trace_id: payload.trace_id,
        session_id: payload.session_id,
        tenant_slug: payload.tenant_slug,
        user_id: payload.user_id,
        path: payload.path,
        message: payload.message,
        context: payload.context,
        error: payload.error,
      })
    );

    return NextResponse.json({ received: true });
  } catch {
    return NextResponse.json({ received: false }, { status: 400 });
  }
}
