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
      className="m-auto w-full max-w-md rounded-2xl border border-[var(--color-border-subtle)] bg-[var(--color-surface-base)] p-0 shadow-2xl backdrop:bg-black/40 backdrop:backdrop-blur-sm"
    >
      <div className="flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-[var(--color-border-hairline)] px-6 py-4">
          <div className="flex items-center gap-2.5">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-[var(--color-status-success-bg)] text-[var(--color-status-success-text)]">
              <Sliders className="h-4 w-4" aria-hidden="true" />
            </div>
            <div>
              <h3 className="text-base font-semibold text-[var(--color-text-primary)]">
                {t("inventory.adjustModal.title")}
              </h3>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg p-1 text-[var(--color-text-muted)] hover:bg-[var(--color-surface-muted)] hover:text-[var(--color-text-primary)]"
            aria-label="Tutup modal"
          >
            <X className="h-5 w-5" aria-hidden="true" />
          </button>
        </div>

        {/* Form Body */}
        <form onSubmit={handleSubmit} className="p-6 space-y-4">
          {errorMessage && (
            <div className="rounded-xl border border-rose-500/20 bg-rose-500/10 p-3 text-xs text-rose-600">
              {errorMessage}
            </div>
          )}

          {/* Product Overview Card */}
          <div className="rounded-xl border border-[var(--color-border-hairline)] bg-[var(--color-surface-muted)] p-3">
            <div className="flex items-center justify-between">
              <div>
                <span className="font-mono text-xs text-[var(--color-text-muted)]">
                  {product.sku}
                </span>
                <h4 className="font-medium text-[var(--color-text-primary)]">
                  {product.name}
                </h4>
              </div>
              <div className="text-right">
                <span className="text-[11px] text-[var(--color-text-muted)]">
                  {t("inventory.adjustModal.currentStock")}
                </span>
                <p className="font-mono text-sm font-semibold text-[var(--color-text-primary)]">
                  {product.stock_quantity} unit
                </p>
              </div>
            </div>
          </div>

          {/* Adjustment Type Radio/Tabs */}
          <div>
            <label className="block text-xs font-medium text-[var(--color-text-secondary)]">
              {t("inventory.adjustModal.adjustType")}
            </label>
            <div className="mt-1.5 grid grid-cols-3 gap-2">
              <button
                type="button"
                onClick={() => setAdjustType("SUBTRACT")}
                className={`flex items-center justify-center rounded-xl border px-3 py-2 text-xs font-medium transition ${
                  adjustType === "SUBTRACT"
                    ? "border-rose-300 bg-rose-50 text-rose-800 font-semibold shadow-sm"
                    : "border-[var(--color-border-hairline)] bg-white text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-muted)]"
                }`}
              >
                {t("inventory.adjustModal.types.subtract")}
              </button>

              <button
                type="button"
                onClick={() => setAdjustType("ADD")}
                className={`flex items-center justify-center rounded-xl border px-3 py-2 text-xs font-medium transition ${
                  adjustType === "ADD"
                    ? "border-emerald-300 bg-emerald-50 text-emerald-800 font-semibold shadow-sm"
                    : "border-[var(--color-border-hairline)] bg-white text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-muted)]"
                }`}
              >
                {t("inventory.adjustModal.types.add")}
              </button>

              <button
                type="button"
                onClick={() => setAdjustType("SET")}
                className={`flex items-center justify-center rounded-xl border px-3 py-2 text-xs font-medium transition ${
                  adjustType === "SET"
                    ? "border-blue-300 bg-blue-50 text-blue-800 font-semibold shadow-sm"
                    : "border-[var(--color-border-hairline)] bg-white text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-muted)]"
                }`}
              >
                {t("inventory.adjustModal.types.set")}
              </button>
            </div>
          </div>

          {/* Quantity & Reason */}
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div>
              <label className="block text-xs font-medium text-[var(--color-text-secondary)]">
                {t("inventory.adjustModal.quantity")} *
              </label>
              <input
                type="number"
                min="1"
                required
                value={quantity || ""}
                onChange={(e) => setQuantity(Number(e.target.value))}
                className="mt-1.5 h-10 w-full rounded-xl border border-[var(--color-border-hairline)] bg-white px-3 font-mono text-sm font-semibold text-[var(--color-text-primary)] focus:border-[var(--color-action-primary)] focus:outline-none focus:ring-2 focus:ring-[var(--color-action-focus-ring)]"
              />
            </div>

            <div>
              <label className="block text-xs font-medium text-[var(--color-text-secondary)]">
                {t("inventory.adjustModal.reason")} *
              </label>
              <select
                value={reason}
                onChange={(e) => setReason(e.target.value as AdjustmentReason)}
                className="mt-1.5 h-10 w-full rounded-xl border border-[var(--color-border-hairline)] bg-white px-3 text-xs text-[var(--color-text-primary)] focus:border-[var(--color-action-primary)] focus:outline-none focus:ring-2 focus:ring-[var(--color-action-focus-ring)]"
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
            <label className="block text-xs font-medium text-[var(--color-text-secondary)]">
              {t("inventory.adjustModal.notes")}
            </label>
            <textarea
              rows={2}
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder={t("inventory.adjustModal.notesPlaceholder")}
              className="mt-1.5 w-full rounded-xl border border-[var(--color-border-hairline)] bg-white p-3 text-xs text-[var(--color-text-primary)] focus:border-[var(--color-action-primary)] focus:outline-none focus:ring-2 focus:ring-[var(--color-action-focus-ring)]"
            />
          </div>

          {/* Live Preview Delta Box */}
          <div className="flex items-center justify-between rounded-xl border border-[var(--color-border-hairline)] bg-[var(--color-surface-muted)] px-4 py-3 text-xs font-mono">
            <div>
              <span className="text-[var(--color-text-secondary)]">{t("inventory.adjustModal.currentStock")}: </span>
              <span className="font-semibold text-[var(--color-text-primary)]">
                {currentStock}
              </span>
            </div>

            <div className="flex items-center gap-1.5 font-bold">
              <span className={adjustType === "SUBTRACT" ? "text-rose-600" : "text-emerald-600"}>
                {deltaStr}
              </span>
              <ArrowRight className="h-3.5 w-3.5 text-[var(--color-text-muted)]" aria-hidden="true" />
              <span className="rounded bg-neutral-200/70 px-1.5 py-0.5 text-[var(--color-text-primary)]">
                {newStock} unit
              </span>
            </div>
          </div>

          {/* Footer Buttons */}
          <div className="flex items-center justify-end gap-3 pt-3 border-t border-[var(--color-border-hairline)]">
            <button
              type="button"
              onClick={onClose}
              className="h-10 rounded-xl px-4 text-xs font-medium text-[var(--color-text-secondary)] transition hover:bg-[var(--color-surface-muted)]"
            >
              {t("inventory.adjustModal.cancel")}
            </button>
            <button
              type="submit"
              disabled={submitting}
              className="h-10 rounded-xl bg-[var(--color-action-primary)] px-5 text-xs font-medium text-white shadow-sm transition hover:bg-[var(--color-action-primary-hover)] disabled:opacity-50"
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
