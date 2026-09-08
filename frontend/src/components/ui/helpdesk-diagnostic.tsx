"use client";

import * as React from "react";
import { getDeviceTelemetry } from "@/lib/device";
import { Copy, Check, ChevronDown, ChevronUp, LifeBuoy } from "lucide-react";

interface HelpdeskDiagnosticProps {
  traceId?: string | null;
  errorMessage?: string | null;
  errorCode?: string | null;
}

export function HelpdeskDiagnostic({ traceId, errorMessage, errorCode }: HelpdeskDiagnosticProps) {
  const [isOpen, setIsOpen] = React.useState(false);
  const [copied, setCopied] = React.useState(false);

  if (!traceId && !errorMessage) return null;

  const device = getDeviceTelemetry();

  const diagnosticPayload = {
    app: "Tenet Commerce POS",
    version: "v0.3.0",
    trace_id: traceId || "unknown",
    error_code: errorCode || "UNKNOWN_ERROR",
    error_message: errorMessage || "",
    timestamp: new Date().toISOString(),
    device: {
      os: device.os,
      browser: device.browser,
      form_factor: device.form_factor,
      screen_resolution: device.screen_resolution,
      viewport: device.viewport_size,
      touch: device.touch_capable,
      network: device.network_type || "normal",
      user_agent: device.user_agent,
    },
  };

  const handleCopy = () => {
    navigator.clipboard.writeText(JSON.stringify(diagnosticPayload, null, 2));
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="mt-3 pt-3 border-t border-black/[0.06] text-xs">
      <div className="flex items-center justify-between">
        <button
          type="button"
          onClick={() => setIsOpen(!isOpen)}
          className="inline-flex items-center gap-1.5 text-[#555D6E] hover:text-[#0B0F19] font-medium transition-colors"
        >
          <LifeBuoy className="w-3.5 h-3.5 text-[#0066CC]" />
          <span>Info Bantuan Helpdesk</span>
          {isOpen ? <ChevronUp className="w-3 h-3" /> : <ChevronDown className="w-3 h-3" />}
        </button>

        {traceId && (
          <div className="flex items-center gap-1.5 font-mono text-[10px] text-[#555D6E]">
            <span>Trace:</span>
            <code className="bg-black/[0.05] px-1.5 py-0.5 rounded select-all font-semibold text-[#0B0F19]">
              {traceId}
            </code>
          </div>
        )}
      </div>

      {isOpen && (
        <div className="mt-2.5 p-3 rounded-[11px] bg-[#F5F5F7] border border-black/[0.06] space-y-2">
          <div className="grid grid-cols-2 gap-2 text-[11px] text-[#555D6E]">
            <div>
              <span className="text-[#8B95A5] block text-[10px] uppercase font-semibold">Perangkat & OS</span>
              <span className="font-medium text-[#0B0F19]">{device.os} ({device.form_factor})</span>
            </div>
            <div>
              <span className="text-[#8B95A5] block text-[10px] uppercase font-semibold">Browser</span>
              <span className="font-medium text-[#0B0F19]">{device.browser}</span>
            </div>
            <div>
              <span className="text-[#8B95A5] block text-[10px] uppercase font-semibold">Resolusi Layar</span>
              <span className="font-mono text-[10px] text-[#0B0F19]">{device.screen_resolution} ({device.viewport_size})</span>
            </div>
            <div>
              <span className="text-[#8B95A5] block text-[10px] uppercase font-semibold">Waktu Kejadian</span>
              <span className="font-mono text-[10px] text-[#0B0F19]">
                {new Date().toLocaleTimeString("id-ID")} WIB
              </span>
            </div>
          </div>

          <div className="pt-2 border-t border-black/[0.06] flex items-center justify-between">
            <span className="text-[10px] text-[#8B95A5]">
              Sertakan info ini saat menghubungi tim support / helpdesk toko.
            </span>
            <button
              type="button"
              onClick={handleCopy}
              className="inline-flex items-center gap-1 px-2 py-1 rounded-[7px] bg-white border border-black/[0.1] text-[11px] font-semibold text-[#0B0F19] hover:bg-[#F0F0F3] shadow-xs active:scale-[0.98] transition-all"
            >
              {copied ? (
                <>
                  <Check className="w-3 h-3 text-emerald-600" />
                  <span className="text-emerald-700">Tersalin!</span>
                </>
              ) : (
                <>
                  <Copy className="w-3 h-3 text-[#555D6E]" />
                  <span>Salin Detail Diagnostik</span>
                </>
              )}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
