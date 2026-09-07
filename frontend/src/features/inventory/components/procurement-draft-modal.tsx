"use client";

import * as React from "react";
import { X, Truck, ShieldCheck, ArrowRight, CheckCircle2 } from "lucide-react";
import { useTranslation } from "@/lib/i18n";
import type { InventoryProduct } from "../types";

interface ProcurementDraftModalProps {
  isOpen: boolean;
  onClose: () => void;
  product: InventoryProduct | null;
  onConfirmDraft?: (product: InventoryProduct, suggestedQuantity: number) => void;
}

export function ProcurementDraftModal({
  isOpen,
  onClose,
  product,
  onConfirmDraft,
}: ProcurementDraftModalProps) {
  const { t } = useTranslation();
  const dialogRef = React.useRef<HTMLDialogElement>(null);

  const [orderQuantity, setOrderQuantity] = React.useState(20);
  const [confirmed, setConfirmed] = React.useState(false);

  React.useEffect(() => {
    if (isOpen && product) {
      const threshold = product.reorder_threshold ?? 5;
      const deficit = Math.max(0, threshold - product.stock_quantity);
      setOrderQuantity(Math.max(20, deficit + 15));
      setConfirmed(false);
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

  const threshold = product.reorder_threshold ?? 5;

  const handleConfirm = () => {
    if (onConfirmDraft) {
      onConfirmDraft(product, orderQuantity);
    }
    setConfirmed(true);
    setTimeout(() => {
      onClose();
    }, 1200);
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
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-amber-100 text-amber-800">
              <Truck className="h-4 w-4" aria-hidden="true" />
            </div>
            <div>
              <h3 className="text-base font-semibold text-[var(--color-text-primary)]">
                {t("inventory.procurementDraft.title")}
              </h3>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg p-1 text-[var(--color-text-muted)] hover:bg-[var(--color-surface-muted)] hover:text-[var(--color-text-primary)]"
            aria-label={t("inventory.procurementDraft.close")}
          >
            <X className="h-5 w-5" aria-hidden="true" />
          </button>
        </div>

        {/* Content */}
        <div className="p-6 space-y-4">
          {confirmed ? (
            <div className="flex flex-col items-center justify-center py-6 text-center space-y-2">
              <CheckCircle2 className="h-10 w-10 text-emerald-600 animate-in zoom-in-50 duration-200" aria-hidden="true" />
              <p className="text-xs font-semibold text-emerald-800">
                {t("inventory.procurementDraft.successMessage", { product: product.name })}
              </p>
            </div>
          ) : (
            <>
              <p className="text-xs text-[var(--color-text-secondary)]">
                {t("inventory.procurementDraft.description")}
              </p>

              {/* Product Info Card */}
              <div className="rounded-xl border border-black/[0.08] bg-[#F8F8FA] p-3.5 space-y-2">
                <div className="flex items-start justify-between gap-2">
                  <div>
                    <div className="text-[11px] text-[var(--color-text-secondary)]">
                      {t("inventory.procurementDraft.productLabel")}
                    </div>
                    <div className="font-semibold text-sm text-[var(--color-text-primary)]">
                      {product.name}
                    </div>
                    <div className="font-mono text-[11px] text-[#8B95A5]">
                      SKU: {product.sku}
                    </div>
                  </div>
                  {product.is_halal_certified && (
                    <span className="inline-flex items-center gap-1 rounded-full bg-emerald-50 px-2 py-0.5 text-[10px] font-medium text-emerald-700 border border-emerald-200/60">
                      <ShieldCheck className="h-3 w-3" aria-hidden="true" />
                      <span>{t("inventory.table.halalBadge")}</span>
                    </span>
                  )}
                </div>

                <div className="rounded-lg bg-amber-50 border border-amber-200/60 p-2 text-xs font-medium text-amber-900">
                  {t("inventory.procurementDraft.thresholdNotice", {
                    current: product.stock_quantity,
                    threshold,
                  })}
                </div>
              </div>

              {/* Suggested Quantity Input */}
              <div className="space-y-1.5">
                <label className="text-xs font-medium text-[var(--color-text-primary)]">
                  {t("inventory.procurementDraft.suggestedQty")}
                </label>
                <div className="flex items-center gap-2">
                  <input
                    type="number"
                    min="1"
                    value={orderQuantity}
                    onChange={(e) => setOrderQuantity(Math.max(1, parseInt(e.target.value) || 1))}
                    className="h-10 w-full rounded-xl border border-black/[0.1] px-3 font-mono text-sm text-[var(--color-text-primary)] focus:border-[#0066CC] focus:outline-none"
                  />
                  <span className="text-xs text-[var(--color-text-secondary)] shrink-0 font-medium">
                    {t("inventory.unit")}
                  </span>
                </div>
              </div>

              {/* Action Buttons */}
              <div className="flex items-center justify-end gap-2.5 pt-3 border-t border-[var(--color-border-hairline)]">
                <button
                  type="button"
                  onClick={onClose}
                  className="h-10 rounded-xl px-4 text-xs font-medium text-[var(--color-text-secondary)] hover:bg-[#F5F5F7] transition"
                >
                  {t("inventory.procurementDraft.close")}
                </button>
                <button
                  type="button"
                  onClick={handleConfirm}
                  className="flex items-center gap-1.5 h-10 rounded-xl bg-[var(--color-action-primary)] px-4 text-xs font-semibold text-white hover:bg-[var(--color-action-primary-hover)] shadow-xs transition"
                >
                  <span>{t("inventory.procurementDraft.action")}</span>
                  <ArrowRight className="h-3.5 w-3.5" aria-hidden="true" />
                </button>
              </div>
            </>
          )}
        </div>
      </div>
    </dialog>
  );
}
