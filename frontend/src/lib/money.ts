/**
 * Money Representation & Formatting Utility
 * Adheres strictly to ADR 002: Money Representation, Rounding, and Bounds
 */

export const CURRENCY_IDR = "IDR";
export const MAX_TRANSACTION_AMOUNT = 1_000_000_000; // Rp 1,000,000,000 (1 Billion IDR)
export const MAX_TENDER_AMOUNT = 2_000_000_000;      // Rp 2,000,000,000 (2 Billion IDR)
export const MAX_LINE_ITEM_QTY = 99_999;

const IDR_FORMATTER = new Intl.NumberFormat("id-ID", {
  style: "currency",
  currency: "IDR",
  minimumFractionDigits: 0,
  maximumFractionDigits: 0,
});

/**
 * Formats an exact integer amount into localized Indonesian Rupiah string (e.g. "Rp 45.000").
 */
export function formatIDR(amount: number): string {
  if (!Number.isFinite(amount)) return "Rp 0";
  const rounded = Math.round(amount);
  return IDR_FORMATTER.format(rounded).replace(/\s+/g, " ");
}

/**
 * Parses user input or raw strings into an exact integer minor unit.
 * Rejects NaN, negative (unless allowed), fractions, or values exceeding MAX_TENDER_AMOUNT.
 */
export function parseIDR(input: string | number, options?: { allowNegative?: boolean }): number | null {
  if (typeof input === "number") {
    if (!Number.isSafeInteger(input)) return null;
    if (!options?.allowNegative && input < 0) return null;
    if (Math.abs(input) > MAX_TENDER_AMOUNT) return null;
    return input;
  }

  if (typeof input !== "string") return null;

  let clean = input.trim();
  if (!clean) return null;

  // Detect negative
  const isNegative = clean.startsWith("-");
  if (isNegative && !options?.allowNegative) return null;
  if (isNegative) clean = clean.slice(1).trim();

  // Strip currency prefixes and spaces
  clean = clean.replace(/^(rp|idr)\.?\s*/i, "");
  // Accept only whole Rupiah with valid thousands grouping and optional zero decimals.
  if (!/^(?:\d+|\d{1,3}(?:\.\d{3})+)(?:,0{1,2})?$/.test(clean)) return null;
  // Standard IDR formatting uses '.' as thousand separator and ',' as decimal separator
  // Remove thousand dots
  clean = clean.replace(/\./g, "");
  // Disallow non-zero decimals or comma-separated subunits
  if (clean.includes(",")) {
    const parts = clean.split(",");
    if (parts[1] && parts[1] !== "00" && parts[1] !== "0") {
      return null; // Reject fractional cents
    }
    clean = parts[0];
  }

  // Ensure only numeric digits remain
  if (!/^\d+$/.test(clean)) return null;

  const parsed = parseInt(clean, 10);
  if (isNaN(parsed) || !Number.isSafeInteger(parsed)) return null;
  if (parsed > MAX_TENDER_AMOUNT) return null;

  return isNegative ? -parsed : parsed;
}

/**
 * Safely adds two integer money amounts, preventing overflow.
 */
export function addMoney(a: number, b: number): number {
  const res = Math.round(a) + Math.round(b);
  if (!Number.isSafeInteger(res)) throw new Error("Monetary addition overflow");
  return res;
}

/**
 * Safely subtracts two integer money amounts.
 */
export function subMoney(a: number, b: number): number {
  const res = Math.round(a) - Math.round(b);
  if (!Number.isSafeInteger(res)) throw new Error("Monetary subtraction overflow");
  return res;
}

/**
 * Multiplies an exact money amount with a factor (e.g. tax rate or discount percentage),
 * rounding half up to the nearest integer Rupiah.
 */
export function mulMoney(amount: number, factor: number): number {
  if (!Number.isFinite(factor)) return 0;
  const res = Math.round(Math.round(amount) * factor);
  if (!Number.isSafeInteger(res)) throw new Error("Monetary multiplication overflow");
  return res;
}

/**
 * Calculates tax amount (e.g. 11% PPN) given a subtotal and integer percentage rate.
 */
export function calculateTax(subtotal: number, ratePercent: number): number {
  if (subtotal <= 0 || ratePercent <= 0) return 0;
  return mulMoney(subtotal, ratePercent / 100);
}

/**
 * Normalizes an API response value (float or string) into exact integer minor units.
 */
export function parseMoneyFromAPI(value: unknown): number {
  if (typeof value === "number") {
    return Number.isFinite(value) ? Math.round(value) : 0;
  }
  if (typeof value === "string") {
    const parsed = parseIDR(value, { allowNegative: true });
    return parsed ?? 0;
  }
  return 0;
}
