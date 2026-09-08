"use client";

import * as React from "react";
import { apiClient } from "@/lib/api";
import { MAX_LINE_ITEM_QTY, MAX_TENDER_AMOUNT, MAX_TRANSACTION_AMOUNT } from "@/lib/money";
import { createCheckoutController } from "../checkout-controller";
import type { CheckoutResponse } from "../types";

export function useCheckout() {
  const [controller] = React.useState(() => createCheckoutController({
    createKey: () => crypto.randomUUID(),
    maxTotal: MAX_TRANSACTION_AMOUNT,
    maxTender: MAX_TENDER_AMOUNT,
    maxQuantity: MAX_LINE_ITEM_QTY,
    send: (body, key) => apiClient.post<CheckoutResponse>("/pos/checkout", body, {
      headers: { "Idempotency-Key": key },
    }),
  }));
  const state = React.useSyncExternalStore(controller.subscribe, controller.getSnapshot, controller.getSnapshot);

  return {
    ...state,
    isSubmitting: state.step === "submitting",
    startReview: controller.startReview,
    closeReview: controller.closeReview,
    setCashTendered: controller.setCashTendered,
    submitCheckout: controller.submitCheckout,
    finishCompleted: controller.finishCompleted,
  };
}
