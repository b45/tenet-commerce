/**
 * Client Device & Hardware Telemetry Detection
 * Captures non-invasive environmental diagnostics to facilitate helpdesk triage.
 */

export interface DeviceTelemetry {
  os: string;
  browser: string;
  form_factor: "desktop" | "tablet" | "mobile";
  screen_resolution: string;
  viewport_size: string;
  pixel_ratio: number;
  language: string;
  touch_capable: boolean;
  network_type?: string;
  hardware_concurrency?: number;
  device_memory_gb?: number;
  user_agent: string;
}

export function getDeviceTelemetry(): DeviceTelemetry {
  if (typeof window === "undefined" || typeof navigator === "undefined") {
    return {
      os: "Server",
      browser: "Node.js",
      form_factor: "desktop",
      screen_resolution: "0x0",
      viewport_size: "0x0",
      pixel_ratio: 1,
      language: "id-ID",
      touch_capable: false,
      user_agent: "SSR",
    };
  }

  const ua = navigator.userAgent;

  // 1. Operating System Detection
  let os = "Unknown OS";
  if (/iPad|iPhone|iPod/.test(ua)) os = "iOS";
  else if (/Macintosh|Mac OS X/.test(ua)) os = "macOS";
  else if (/Windows NT/.test(ua)) os = "Windows";
  else if (/Android/.test(ua)) os = "Android";
  else if (/Linux/.test(ua)) os = "Linux";

  // 2. Browser Detection
  let browser = "Unknown Browser";
  if (/Edg\//.test(ua)) browser = "Microsoft Edge";
  else if (/Chrome\//.test(ua) && !/Chromium|Edg/.test(ua)) browser = "Google Chrome";
  else if (/Safari\//.test(ua) && !/Chrome|Chromium/.test(ua)) browser = "Safari";
  else if (/Firefox\//.test(ua)) browser = "Mozilla Firefox";

  // 3. Form Factor Detection
  const width = window.innerWidth;
  const isTouch = "ontouchstart" in window || navigator.maxTouchPoints > 0;
  let formFactor: "desktop" | "tablet" | "mobile" = "desktop";

  if (width < 640 || (/Android|iPhone|Mobile/.test(ua) && width < 768)) {
    formFactor = "mobile";
  } else if (width < 1024 || (isTouch && width <= 1280)) {
    formFactor = "tablet";
  }

  // 4. Network Info (if supported by modern Chromium/Safari)
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const conn = (navigator as any).connection || (navigator as any).mozConnection || (navigator as any).webkitConnection;
  const networkType = conn?.effectiveType || (conn?.type ? String(conn.type) : undefined);

  return {
    os,
    browser,
    form_factor: formFactor,
    screen_resolution: `${window.screen?.width || 0}x${window.screen?.height || 0}`,
    viewport_size: `${window.innerWidth}x${window.innerHeight}`,
    pixel_ratio: window.devicePixelRatio || 1,
    language: navigator.language || "id-ID",
    touch_capable: isTouch,
    network_type: networkType,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    hardware_concurrency: (navigator as any).hardwareConcurrency,
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    device_memory_gb: (navigator as any).deviceMemory,
    user_agent: ua,
  };
}
