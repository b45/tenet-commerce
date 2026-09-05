"use client";

import * as React from "react";
import { apiClient } from "@/lib/api";
import type { CartItem, CheckoutResponse, CheckoutStep } from "../types";

export function useCheckout() {
  const [step, setStep] = React.useState<CheckoutStep>("idle");
  const [cashTendered, setCashTendered] = React.useState<number>(0);
  const [idempotencyKey, setIdempotencyKey] = React.useState<string>("");
  const [receipt, setReceipt] = React.useState<CheckoutResponse | null>(null);
  const [errorMessage, setErrorMessage] = React.useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  // Initialize a new payment intent with fresh UUIDv4 Idempotency Key
  const startReview = React.useCallback((totalAmount: number) => {
    const newKey = typeof crypto !== "undefined" && crypto.randomUUID
      ? crypto.randomUUID()
      : `idem_${Date.now()}_${Math.random().toString(36).substring(2, 9)}`;

    setIdempotencyKey(newKey);
    setCashTendered(totalAmount);
    setErrorMessage(null);
    setReceipt(null);
    setStep("review");
  }, []);

  const closeReview = React.useCallback(() => {
    if (isSubmitting) return;
    setStep("idle");
    setErrorMessage(null);
  }, [isSubmitting]);

  const submitCheckout = React.useCallback(
    async (items: CartItem[], totalAmount: number) => {
      if (items.length === 0) {
        setErrorMessage("Keranjang belanja masih kosong.");
        return;
      }

      if (cashTendered < totalAmount) {
        setErrorMessage("Uang tunai yang diterima kurang dari total pembayaran.");
        return;
      }

      try {
        setIsSubmitting(true);
        setStep("submitting");
        setErrorMessage(null);

        const payload = {
          items: items.map((i) => ({
            sku: i.product.sku,
            quantity: i.quantity,
          })),
          payment_method: "CASH" as const,
          discount_amount: 0,
          cash_tendered: cashTendered,
        };

        const res = await apiClient.post<CheckoutResponse>("/pos/checkout", payload, {
          headers: {
            "Idempotency-Key": idempotencyKey,
          },
        });

        if (res.success && res.data) {
          setReceipt(res.data);
          setStep("completed");
        } else {
          // Specific backend rejection
          const msg = res.error?.message || "Pembayaran gagal diproses oleh sistem.";
          setErrorMessage(msg);
          setStep("rejected");
        }
      } catch (err: unknown) {
        // Unknown outcome: network timeout, 502, or server crash
        // Per POS_CONTRACT_MAP.md: Do not auto-retry or allow double-charging
        const msg =
          err instanceof Error
            ? err.message
            : "Koneksi terputus saat memproses pembayaran. Hasil belum diketahui. Jangan menagih ulang ke pelanggan.";
        setErrorMessage(msg);
        setStep("unknown_error");
      } finally {
        setIsSubmitting(false);
      }
    },
    [cashTendered, idempotencyKey]
  );

  const resetCheckout = React.useCallback(() => {
    setStep("idle");
    setCashTendered(0);
    setIdempotencyKey("");
    setReceipt(null);
    setErrorMessage(null);
    setIsSubmitting(false);
  }, []);

  return {
    step,
    cashTendered,
    setCashTendered,
    idempotencyKey,
    receipt,
    errorMessage,
    isSubmitting,
    startReview,
    closeReview,
    submitCheckout,
    resetCheckout,
  };
}
