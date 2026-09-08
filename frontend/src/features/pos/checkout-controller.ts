import type { ClientResponse } from "../../lib/api";
import type { CartItem, CheckoutRequest, CheckoutResponse, CheckoutStep } from "./types";

interface CheckoutState {
  step: CheckoutStep;
  cashTendered: number;
  idempotencyKey: string;
  receipt: CheckoutResponse | null;
  errorMessage: string | null;
  command: CheckoutRequest | null;
}

interface Dependencies {
  send: (body: CheckoutRequest, key: string) => Promise<ClientResponse<CheckoutResponse>>;
  createKey: () => string;
  maxTotal: number;
  maxTender: number;
  maxQuantity: number;
}

const unknownMessage = "Hasil transaksi belum diketahui. Jangan menagih ulang atau memulai transaksi baru. Minta penanggung jawab merekonsiliasi transaksi ini sebelum melanjutkan.";

// Only documented pre-commit business rejections permit a new review.
export function isDefiniteRejection(error: ClientResponse<unknown>["error"]): boolean {
  if (!error) return false;
  return (
    (error.status === 400 && ["VALIDATION_ERROR", "INSUFFICIENT_CASH_TENDERED", "INVALID_CASH_TENDERED", "MISSING_IDEMPOTENCY_KEY"].includes(error.code)) ||
    (error.status === 409 && error.code === "INSUFFICIENT_STOCK") ||
    (error.status === 404 && error.code === "PRODUCT_NOT_FOUND")
  );
}

function isReceipt(value: unknown): value is CheckoutResponse {
  if (!value || typeof value !== "object") return false;
  const r = value as CheckoutResponse;
  const money = [r.subtotal_amount, r.tax_amount, r.discount_amount, r.total_amount, r.cash_tendered, r.change_amount];
  return r.status === "COMPLETED" && typeof r.transaction_id === "string" && !!r.transaction_id &&
    typeof r.transaction_number === "string" && !!r.transaction_number &&
    typeof r.created_at === "string" && Number.isFinite(Date.parse(r.created_at)) &&
    money.every(n => Number.isSafeInteger(n) && n >= 0) &&
    r.cash_tendered - r.total_amount === r.change_amount &&
    Array.isArray(r.items) && r.items.length > 0 && r.items.every(item =>
      item && typeof item.sku === "string" && Number.isSafeInteger(item.quantity) && item.quantity > 0 &&
      Number.isSafeInteger(item.unit_price) && item.unit_price >= 0 &&
      Number.isSafeInteger(item.subtotal ?? item.subtotal_amount) && (item.subtotal ?? item.subtotal_amount ?? -1) >= 0
    );
}

/** One mounted POS session. No retry, durable storage or cross-tab recovery claim. */
export function createCheckoutController(deps: Dependencies) {
  let state: CheckoutState = { step: "idle", cashTendered: 0, idempotencyKey: "", receipt: null, errorMessage: null, command: null };
  const listeners = new Set<() => void>();
  const update = (patch: Partial<CheckoutState>) => {
    state = { ...state, ...patch };
    listeners.forEach(listener => listener());
  };
  return {
    getSnapshot: () => state,
    subscribe: (listener: () => void) => {
      listeners.add(listener);
      return () => { listeners.delete(listener); };
    },
    startReview: (total: number) => {
      if (state.step !== "idle") return false;
      if (!Number.isSafeInteger(total) || total < 0 || total > deps.maxTotal) return false;
      update({ step: "review", cashTendered: total, idempotencyKey: "", receipt: null, errorMessage: null, command: null });
      return true;
    },
    setCashTendered: (amount: number) => {
      if (state.step !== "review") return;
      update({ cashTendered: amount, errorMessage: null });
    },
    closeReview: () => {
      if (state.step !== "review" && state.step !== "rejected") return false;
      update({ step: "idle", errorMessage: null });
      return true;
    },
    finishCompleted: (clearPaidCart: () => void) => {
      if (state.step !== "completed") return false;
      // Every receipt dismissal clears the paid cart and keeps the server receipt.
      clearPaidCart();
      update({ step: "idle", cashTendered: 0, idempotencyKey: "", errorMessage: null });
      return true;
    },
    submitCheckout: async (items: CartItem[], total: number) => {
      // Synchronous guard, including two calls before React renders again.
      if (state.step !== "review") return;
      if (!items.length || items.some(item => !item.product.sku || !Number.isSafeInteger(item.quantity) || item.quantity < 1 || item.quantity > deps.maxQuantity)) {
        update({ errorMessage: "Periksa isi dan jumlah barang dalam keranjang." });
        return;
      }
      if (!Number.isSafeInteger(total) || total < 0 || total > deps.maxTotal ||
          !Number.isSafeInteger(state.cashTendered) || state.cashTendered < total || state.cashTendered > deps.maxTender) {
        update({ errorMessage: "Nominal tunai harus berupa Rupiah utuh, cukup untuk membayar, dan dalam batas transaksi." });
        return;
      }
      let key: string;
      try {
        key = deps.createKey();
        if (!key) throw new Error("Empty command key");
      } catch {
        update({ errorMessage: "Identitas transaksi aman tidak tersedia. Pembayaran belum dikirim." });
        return;
      }
      const body: CheckoutRequest = {
        items: items.map(item => ({ sku: item.product.sku, quantity: item.quantity })),
        payment_method: "CASH", discount_amount: 0, cash_tendered: state.cashTendered,
      };
      body.items.forEach(Object.freeze);
      Object.freeze(body.items);
      Object.freeze(body);
      update({ step: "submitting", idempotencyKey: key, command: body, errorMessage: null });
      try {
        const result = await deps.send(body, key);
        if (result.success && isReceipt(result.data)) {
          update({ step: "completed", receipt: result.data });
        } else if (!result.success && isDefiniteRejection(result.error)) {
          update({ step: "rejected", errorMessage: "Transaksi ditolak sebelum selesai. Kembali ke keranjang untuk memeriksa stok, harga, dan nominal sebelum membuat review baru." });
        } else {
          const auth = result.error?.status === 401 || result.error?.status === 403;
          update({ step: "unknown_error", errorMessage: auth ? `Sesi atau izin perlu diperiksa. ${unknownMessage}` : unknownMessage });
        }
      } catch {
        update({ step: "unknown_error", errorMessage: unknownMessage });
      }
    },
  };
}
