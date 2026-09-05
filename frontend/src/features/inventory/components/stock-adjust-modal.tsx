"use client";

import * as React from "react";
import { X, Sliders, ArrowRight } from "lucide-react";
import { useTranslation } from "@/lib/i18n";
import type {
  InventoryProduct,
  AdjustmentType,
  AdjustmentReason,
  StockAdjustmentPayload,
  StockAdjustmentResponse,
} from "../types";

interface StockAdjustModalProps {
  isOpen: boolean;
  onClose: () => void;
  product: InventoryProduct | null;
  onSubmitAdjust: (
    payload: StockAdjustmentPayload
  ) => Promise<{ success: boolean; data?: StockAdjustmentResponse; error?: string }>;
}

export function StockAdjustModal({
  isOpen,
  onClose,
  product,
  onSubmitAdjust,
}: StockAdjustModalProps) {
  const { t } = useTranslation();
  const dialogRef = React.useRef<HTMLDialogElement>(null);

  const [adjustType, setAdjustType] = React.useState<AdjustmentType>("SUBTRACT");
  const [quantity, setQuantity] = React.useState<number>(1);
  const [reason, setReason] = React.useState<AdjustmentReason>("DAMAGE");
  const [notes, setNotes] = React.useState("");

  const [submitting, setSubmitting] = React.useState(false);
  const [errorMessage, setErrorMessage] = React.useState<string | null>(null);

  // Reset form on open
  React.useEffect(() => {
    if (isOpen) {
      setAdjustType("SUBTRACT");
      setQuantity(1);
      setReason("DAMAGE");
      setNotes("");
      setErrorMessage(null);
    }
  }, [isOpen, product]);

  // Dialog open/close lifecycle
  React.useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;

    if (isOpen) {
      if (!dialog.open) {
        dialog.showModal();
      }
    } else {
      if (dialog.open) {
        dialog.close();
      }
    }
  }, [isOpen]);

  if (!product) return null;

  const currentStock = product.stock_quantity;

  // Calculate new stock
  let newStock = currentStock;
  let deltaStr = "0";
  if (adjustType === "ADD") {
    newStock = currentStock + (quantity || 0);
    deltaStr = `+${quantity || 0}`;
  } else if (adjustType === "SUBTRACT") {
    newStock = Math.max(0, currentStock - (quantity || 0));
    deltaStr = `-${quantity || 0}`;
  } else if (adjustType === "SET") {
    newStock = Math.max(0, quantity || 0);
    const diff = newStock - currentStock;
    deltaStr = diff >= 0 ? `+${diff}` : `${diff}`;
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorMessage(null);

    if (quantity <= 0) {
      setErrorMessage("Jumlah penyesuaian harus lebih besar dari 0");
      return;
    }

    setSubmitting(true);
    try {
      const payload: StockAdjustmentPayload = {
        product_id: product.id,
        adjustment_type: adjustType,
        quantity: Number(quantity),
        reason,
        notes: notes.trim() || undefined,
      };

      const res = await onSubmitAdjust(payload);
      if (res.success) {
        onClose();
      } else {
        setErrorMessage(res.error || "Gagal menyesuaikan stok");
      }
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <dialog
      ref={dialogRef}
      onCancel={(e) => {
        e.preventDefault();
        onClose();
      }}
      onClick={(e) => {
        if (e.target === dialogRef.current) {
          onClose();
        }
      }}
      className="m-auto w-full max-w-md rounded-2xl border border-neutral-200 bg-white p-0 shadow-2xl backdrop:bg-black/40 backdrop:backdrop-blur-sm dark:border-neutral-800 dark:bg-neutral-900"
    >
      <div className="flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-neutral-200 px-6 py-4 dark:border-neutral-800">
          <div className="flex items-center gap-2.5">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-emerald-500/10 text-emerald-600 dark:bg-emerald-500/20 dark:text-emerald-400">
              <Sliders className="h-4 w-4" aria-hidden="true" />
            </div>
            <div>
              <h3 className="text-base font-semibold text-neutral-900 dark:text-neutral-100">
                {t("inventory.adjustModal.title")}
              </h3>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg p-1 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-700 dark:hover:bg-neutral-800"
            aria-label="Tutup modal"
          >
            <X className="h-5 w-5" aria-hidden="true" />
          </button>
        </div>

        {/* Form Body */}
        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          {errorMessage && (
            <div className="rounded-xl border border-rose-500/20 bg-rose-500/10 p-3 text-xs text-rose-600 dark:text-rose-400">
              {errorMessage}
            </div>
          )}

          {/* Product Overview Card */}
          <div className="rounded-xl border border-neutral-200 bg-neutral-50/50 p-3 dark:border-neutral-800 dark:bg-neutral-800/40">
            <div className="flex items-center justify-between">
              <div>
                <span className="font-mono text-xs text-neutral-500">
                  {product.sku}
                </span>
                <h4 className="font-medium text-neutral-900 dark:text-neutral-100">
                  {product.name}
                </h4>
              </div>
              <div className="text-right">
                <span className="text-[11px] text-neutral-500">
                  {t("inventory.adjustModal.currentStock")}
                </span>
                <p className="font-mono text-sm font-semibold text-neutral-900 dark:text-neutral-100">
                  {product.stock_quantity} unit
                </p>
              </div>
            </div>
          </div>

          {/* Adjustment Type Radio/Tabs */}
          <div>
            <label className="block text-xs font-medium text-neutral-700 dark:text-neutral-300">
              {t("inventory.adjustModal.adjustType")}
            </label>
            <div className="mt-1.5 grid grid-cols-3 gap-2">
              <button
                type="button"
                onClick={() => setAdjustType("SUBTRACT")}
                className={`flex items-center justify-center rounded-xl border px-3 py-2 text-xs font-medium transition ${
                  adjustType === "SUBTRACT"
                    ? "border-rose-500 bg-rose-500/10 text-rose-700 dark:text-rose-300 font-semibold"
                    : "border-neutral-200 bg-white text-neutral-600 hover:bg-neutral-50 dark:border-neutral-800 dark:bg-neutral-800 dark:text-neutral-300"
                }`}
              >
                {t("inventory.adjustModal.types.subtract")}
              </button>

              <button
                type="button"
                onClick={() => setAdjustType("ADD")}
                className={`flex items-center justify-center rounded-xl border px-3 py-2 text-xs font-medium transition ${
                  adjustType === "ADD"
                    ? "border-emerald-500 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300 font-semibold"
                    : "border-neutral-200 bg-white text-neutral-600 hover:bg-neutral-50 dark:border-neutral-800 dark:bg-neutral-800 dark:text-neutral-300"
                }`}
              >
                {t("inventory.adjustModal.types.add")}
              </button>

              <button
                type="button"
                onClick={() => setAdjustType("SET")}
                className={`flex items-center justify-center rounded-xl border px-3 py-2 text-xs font-medium transition ${
                  adjustType === "SET"
                    ? "border-blue-500 bg-blue-500/10 text-blue-700 dark:text-blue-300 font-semibold"
                    : "border-neutral-200 bg-white text-neutral-600 hover:bg-neutral-50 dark:border-neutral-800 dark:bg-neutral-800 dark:text-neutral-300"
                }`}
              >
                {t("inventory.adjustModal.types.set")}
              </button>
            </div>
          </div>

          {/* Quantity & Reason */}
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div>
              <label className="block text-xs font-medium text-neutral-700 dark:text-neutral-300">
                {t("inventory.adjustModal.quantity")} *
              </label>
              <input
                type="number"
                min="1"
                required
                value={quantity || ""}
                onChange={(e) => setQuantity(Number(e.target.value))}
                className="mt-1.5 h-10 w-full rounded-xl border border-neutral-200 bg-white px-3 font-mono text-sm font-semibold text-neutral-900 focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/20 dark:border-neutral-800 dark:bg-neutral-800/50 dark:text-neutral-100"
              />
            </div>

            <div>
              <label className="block text-xs font-medium text-neutral-700 dark:text-neutral-300">
                {t("inventory.adjustModal.reason")} *
              </label>
              <select
                value={reason}
                onChange={(e) => setReason(e.target.value as AdjustmentReason)}
                className="mt-1.5 h-10 w-full rounded-xl border border-neutral-200 bg-white px-3 text-xs text-neutral-900 focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/20 dark:border-neutral-800 dark:bg-neutral-800/50 dark:text-neutral-100"
              >
                <option value="DAMAGE">{t("inventory.adjustModal.reasons.damage")}</option>
                <option value="EXPIRED">{t("inventory.adjustModal.reasons.expired")}</option>
                <option value="AUDIT_CORRECTION">
                  {t("inventory.adjustModal.reasons.auditCorrection")}
                </option>
                <option value="RESTOCK">{t("inventory.adjustModal.reasons.restock")}</option>
                <option value="OTHER">{t("inventory.adjustModal.reasons.other")}</option>
              </select>
            </div>
          </div>

          {/* Notes */}
          <div>
            <label className="block text-xs font-medium text-neutral-700 dark:text-neutral-300">
              {t("inventory.adjustModal.notes")}
            </label>
            <textarea
              rows={2}
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder={t("inventory.adjustModal.notesPlaceholder")}
              className="mt-1.5 w-full rounded-xl border border-neutral-200 bg-white p-3 text-xs text-neutral-900 focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/20 dark:border-neutral-800 dark:bg-neutral-800/50 dark:text-neutral-100"
            />
          </div>

          {/* Live Preview Delta Box */}
          <div className="flex items-center justify-between rounded-xl border border-neutral-200 bg-neutral-50 px-4 py-3 text-xs font-mono dark:border-neutral-800 dark:bg-neutral-800/50">
            <div>
              <span className="text-neutral-500">{t("inventory.adjustModal.currentStock")}: </span>
              <span className="font-semibold text-neutral-800 dark:text-neutral-200">
                {currentStock}
              </span>
            </div>

            <div className="flex items-center gap-1.5 font-bold">
              <span className={adjustType === "SUBTRACT" ? "text-rose-600" : "text-emerald-600"}>
                {deltaStr}
              </span>
              <ArrowRight className="h-3.5 w-3.5 text-neutral-400" aria-hidden="true" />
              <span className="rounded bg-neutral-200/60 px-1.5 py-0.5 text-neutral-900 dark:bg-neutral-700 dark:text-neutral-100">
                {newStock} unit
              </span>
            </div>
          </div>

          {/* Footer Buttons */}
          <div className="flex items-center justify-end gap-3 pt-3 border-t border-neutral-200 dark:border-neutral-800">
            <button
              type="button"
              onClick={onClose}
              className="h-10 rounded-xl px-4 text-xs font-medium text-neutral-600 transition hover:bg-neutral-100 dark:text-neutral-400 dark:hover:bg-neutral-800"
            >
              {t("inventory.adjustModal.cancel")}
            </button>
            <button
              type="submit"
              disabled={submitting}
              className="h-10 rounded-xl bg-emerald-600 px-5 text-xs font-medium text-white shadow-sm transition hover:bg-emerald-700 disabled:opacity-50 dark:bg-emerald-600 dark:hover:bg-emerald-500"
            >
              {submitting
                ? t("inventory.adjustModal.submitting")
                : t("inventory.adjustModal.submit")}
            </button>
          </div>
        </form>
      </div>
    </dialog>
  );
}
