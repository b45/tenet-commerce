/**
 * Date & Time Formatting Utilities
 * Standardized timezone: Asia/Jakarta (WIB) as primary commercial baseline.
 */

const JAKARTA_TZ = "Asia/Jakarta";

/**
 * Formats ISO date string into Indonesian standard format e.g. "5 Sep 2026, 16:07 WIB".
 */
export function formatDateTime(
  dateInput: string | Date | number,
  options?: { showTimezone?: boolean }
): string {
  try {
    const d = new Date(dateInput);
    if (isNaN(d.getTime())) return "-";

    const formatted = new Intl.DateTimeFormat("id-ID", {
      timeZone: JAKARTA_TZ,
      day: "numeric",
      month: "short",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
      hour12: false,
    }).format(d);

    const suffix = options?.showTimezone !== false ? " WIB" : "";
    return `${formatted}${suffix}`;
  } catch {
    return "-";
  }
}

/**
 * Formats date only e.g. "5 Sep 2026".
 */
export function formatDateOnly(dateInput: string | Date | number): string {
  try {
    const d = new Date(dateInput);
    if (isNaN(d.getTime())) return "-";

    return new Intl.DateTimeFormat("id-ID", {
      timeZone: JAKARTA_TZ,
      day: "numeric",
      month: "short",
      year: "numeric",
    }).format(d);
  } catch {
    return "-";
  }
}

/**
 * Formats time only e.g. "16:07".
 */
export function formatTimeOnly(dateInput: string | Date | number): string {
  try {
    const d = new Date(dateInput);
    if (isNaN(d.getTime())) return "-";

    return new Intl.DateTimeFormat("id-ID", {
      timeZone: JAKARTA_TZ,
      hour: "2-digit",
      minute: "2-digit",
      hour12: false,
    }).format(d);
  } catch {
    return "-";
  }
}

/**
 * Checks if an ISO date string is expired relative to current time.
 */
export function isExpired(isoDate: string): boolean {
  try {
    const expiry = new Date(isoDate).getTime();
    return Number.isFinite(expiry) && expiry < Date.now();
  } catch {
    return false;
  }
}

/**
 * Calculates remaining whole days until an expiry date (negative if already expired).
 */
export function daysUntilExpiry(isoDate: string): number {
  try {
    const target = new Date(isoDate).getTime();
    if (!Number.isFinite(target)) return 0;
    const diffMs = target - Date.now();
    return Math.floor(diffMs / (1000 * 60 * 60 * 24));
  } catch {
    return 0;
  }
}
