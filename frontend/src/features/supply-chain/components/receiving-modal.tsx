"use client";

import * as React from "react";
import { CheckCircle2, AlertTriangle, Truck, ShieldCheck, ArrowRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Alert } from "@/components/ui/alert";
import { useTranslation } from "@/lib/i18n";
import { apiClient } from "@/lib/api";

export interface POItem {
  id: string;
  product_id: string;
  product_name: string;
  product_sku: string;
  quantity: number;
  received_quantity: number;
  remaining_quantity: number;
  unit_cost: number;
}

export interface ReceivingItemForm {
  product_id: string;
  product_name: string;
  product_sku: string;
  ordered_quantity: number;
  remaining_quantity: number;
  delivered_quantity: number;
  accepted_quantity: number;
  rejected_quantity: number;
  qc_reason: string;
}

export interface ReceivingModalProps {
  isOpen: boolean;
  poId: string;
  poNumber: string;
  supplierName: string;
  items: POItem[];
  onClose: () => void;
  onSuccess: () => void;
}

export function ReceivingModal({
  isOpen,
  poId,
  poNumber,
  supplierName,
  items,
  onClose,
  onSuccess,
}: ReceivingModalProps) {
  const { t } = useTranslation();
  const [formItems, setFormItems] = React.useState<ReceivingItemForm[]>([]);
  const [notes, setNotes] = React.useState("");
  const [isSubmitting, setIsSubmitting] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [successResult, setSuccessResult] = React.useState<{ gr_number: string } | null>(null);

  React.useEffect(() => {
    if (isOpen && items.length > 0) {
      setFormItems(
        items.map((item) => ({
          product_id: item.product_id,
          product_name: item.product_name || "Produk",
          product_sku: item.product_sku || "-",
          ordered_quantity: item.quantity,
          remaining_quantity: item.remaining_quantity,
          delivered_quantity: item.remaining_quantity,
          accepted_quantity: item.remaining_quantity,
          rejected_quantity: 0,
          qc_reason: "",
        }))
      );
      setError(null);
      setSuccessResult(null);
      setNotes("");
    }
  }, [isOpen, items]);

  if (!isOpen) return null;

  const handleItemChange = (
    index: number,
    field: "delivered_quantity" | "accepted_quantity" | "rejected_quantity" | "qc_reason",
    value: number | string
  ) => {
    setFormItems((prev) => {
      const updated = [...prev];
      const item = { ...updated[index] };

      if (field === "delivered_quantity") {
        const val = Math.max(0, Number(value));
        item.delivered_quantity = val;
        // Default split: accepted = delivered - rejected (clamp to >= 0)
        item.accepted_quantity = Math.max(0, val - item.rejected_quantity);
      } else if (field === "accepted_quantity") {
        const val = Math.max(0, Number(value));
        item.accepted_quantity = val;
        // Reconcile delivered = accepted + rejected
        item.delivered_quantity = val + item.rejected_quantity;
      } else if (field === "rejected_quantity") {
        const val = Math.max(0, Number(value));
        item.rejected_quantity = val;
        // Reconcile delivered = accepted + rejected
        item.delivered_quantity = item.accepted_quantity + val;
      } else if (field === "qc_reason") {
        item.qc_reason = String(value);
      }

      updated[index] = item;
      return updated;
    });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    // Frontend validation invariants
    for (const item of formItems) {
      if (item.delivered_quantity <= 0) {
        setError(t("supplyChain.receiving.errors.deliveredPositive"));
        return;
      }
      if (item.delivered_quantity !== item.accepted_quantity + item.rejected_quantity) {
        setError(t("supplyChain.receiving.errors.arithmeticMismatch"));
        return;
      }
      if (item.delivered_quantity > item.remaining_quantity) {
        setError(t("supplyChain.receiving.errors.exceedsRemaining"));
        return;
      }
      if (item.rejected_quantity > 0 && !item.qc_reason.trim()) {
        setError(t("supplyChain.receiving.errors.qcReasonRequired"));
        return;
      }
    }

    setIsSubmitting(true);
    const idempotencyKey = `gr_${poId}_${Date.now()}`;

    try {
      const payload = {
        purchase_order_id: poId,
        notes: notes.trim(),
        items: formItems.map((item) => ({
          product_id: item.product_id,
          delivered_quantity: item.delivered_quantity,
          accepted_quantity: item.accepted_quantity,
          rejected_quantity: item.rejected_quantity,
          qc_reason: item.rejected_quantity > 0 ? item.qc_reason.trim() : undefined,
        })),
      };

      const res = await apiClient.post<{ gr_number: string }>(
        "/supply-chain/goods-receipts",
        payload,
        {
          headers: {
            "Idempotency-Key": idempotencyKey,
          },
        }
      );

      if (res.success && res.data) {
        setSuccessResult(res.data);
        onSuccess();
      } else {
        setError(res.error?.message || t("supplyChain.receiving.errors.failed"));
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : t("supplyChain.receiving.errors.failed"));
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4 backdrop-blur-xs">
      <div className="w-full max-w-2xl rounded-2xl border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] shadow-xl overflow-hidden animate-in fade-in zoom-in-95 duration-150 max-h-[90vh] flex flex-col">
        {/* Modal Header */}
        <div className="flex items-center justify-between border-b border-[var(--color-border-hairline)] px-6 py-4">
          <div className="flex items-center gap-2.5">
            <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-[var(--color-action-primary)] text-white">
              <Truck className="h-5 w-5" />
            </div>
            <div>
              <h2 className="text-base font-bold text-[var(--color-text-primary)]">
                {t("supplyChain.receiving.title")}
              </h2>
              <p className="text-xs text-[var(--color-text-secondary)]">
                {poNumber} • {supplierName}
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg p-1.5 text-[var(--color-text-tertiary)] hover:bg-[var(--color-surface-muted)] hover:text-[var(--color-text-primary)]"
          >
            ✕
          </button>
        </div>

        {/* Modal Body */}
        <div className="p-6 overflow-y-auto flex-1 space-y-4">
          {error && (
            <Alert variant="destructive">
              <AlertTriangle className="h-4 w-4" />
              <span>{error}</span>
            </Alert>
          )}

          {successResult ? (
            <div className="py-8 text-center space-y-3">
              <CheckCircle2 className="h-12 w-12 text-emerald-600 mx-auto" />
              <h3 className="text-lg font-bold text-[var(--color-text-primary)]">
                {t("supplyChain.receiving.successTitle")}
              </h3>
              <p className="text-xs text-[var(--color-text-secondary)] font-mono">
                {t("supplyChain.receiving.grNumber")}: {successResult.gr_number}
              </p>
              <div className="pt-4">
                <Button type="button" onClick={onClose} className="w-full">
                  {t("supplyChain.receiving.done")}
                </Button>
              </div>
            </div>
          ) : (
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="rounded-xl border border-[var(--color-border-hairline)] bg-[var(--color-surface-muted)] p-3 text-xs text-[var(--color-text-secondary)] flex items-center gap-2">
                <ShieldCheck className="h-4 w-4 text-[var(--color-action-primary)] shrink-0" />
                <span>{t("supplyChain.receiving.qcNotice")}</span>
              </div>

              {/* Items List */}
              <div className="space-y-3">
                {formItems.map((item, idx) => (
                  <div
                    key={item.product_id}
                    className="rounded-xl border border-[var(--color-border-hairline)] p-4 bg-[var(--color-surface-base)] space-y-3"
                  >
                    <div className="flex items-center justify-between">
                      <div>
                        <div className="font-bold text-sm text-[var(--color-text-primary)]">
                          {item.product_name}
                        </div>
                        <div className="font-mono text-xs text-[var(--color-text-tertiary)]">
                          {item.product_sku}
                        </div>
                      </div>
                      <div className="text-right font-mono text-xs">
                        <span className="text-[var(--color-text-secondary)]">
                          {t("supplyChain.receiving.remaining")}:{" "}
                        </span>
                        <span className="font-bold text-[var(--color-text-primary)]">
                          {item.remaining_quantity}
                        </span>
                      </div>
                    </div>

                    {/* QC Inputs Grid */}
                    <div className="grid grid-cols-3 gap-2.5 pt-1">
                      <div>
                        <label className="text-[11px] font-semibold text-[var(--color-text-secondary)]">
                          {t("supplyChain.receiving.deliveredQty")}
                        </label>
                        <Input
                          type="number"
                          min={0}
                          max={item.remaining_quantity}
                          value={item.delivered_quantity}
                          onChange={(e) =>
                            handleItemChange(idx, "delivered_quantity", parseInt(e.target.value) || 0)
                          }
                          disabled={isSubmitting}
                          required
                        />
                      </div>

                      <div>
                        <label className="text-[11px] font-semibold text-emerald-600 dark:text-emerald-400">
                          {t("supplyChain.receiving.acceptedQty")}
                        </label>
                        <Input
                          type="number"
                          min={0}
                          max={item.delivered_quantity}
                          value={item.accepted_quantity}
                          onChange={(e) =>
                            handleItemChange(idx, "accepted_quantity", parseInt(e.target.value) || 0)
                          }
                          disabled={isSubmitting}
                          required
                        />
                      </div>

                      <div>
                        <label className="text-[11px] font-semibold text-rose-600 dark:text-rose-400">
                          {t("supplyChain.receiving.rejectedQty")}
                        </label>
                        <Input
                          type="number"
                          min={0}
                          max={item.delivered_quantity}
                          value={item.rejected_quantity}
                          onChange={(e) =>
                            handleItemChange(idx, "rejected_quantity", parseInt(e.target.value) || 0)
                          }
                          disabled={isSubmitting}
                        />
                      </div>
                    </div>

                    {item.rejected_quantity > 0 && (
                      <div className="pt-1">
                        <label className="text-[11px] font-semibold text-rose-600 dark:text-rose-400">
                          {t("supplyChain.receiving.rejectionReason")} *
                        </label>
                        <Input
                          type="text"
                          placeholder={t("supplyChain.receiving.rejectionReasonPlaceholder")}
                          value={item.qc_reason}
                          onChange={(e) => handleItemChange(idx, "qc_reason", e.target.value)}
                          disabled={isSubmitting}
                          required
                        />
                      </div>
                    )}
                  </div>
                ))}
              </div>

              <div>
                <label className="text-xs font-semibold text-[var(--color-text-secondary)]">
                  {t("supplyChain.receiving.notes")}
                </label>
                <Input
                  type="text"
                  placeholder={t("supplyChain.receiving.notesPlaceholder")}
                  value={notes}
                  onChange={(e) => setNotes(e.target.value)}
                  disabled={isSubmitting}
                />
              </div>

              <div className="flex items-center justify-end gap-2 pt-3 border-t border-[var(--color-border-hairline)]">
                <Button type="button" variant="outline" onClick={onClose} disabled={isSubmitting}>
                  {t("supplyChain.receiving.cancel")}
                </Button>
                <Button type="submit" isLoading={isSubmitting} className="gap-1.5">
                  <span>{t("supplyChain.receiving.submit")}</span>
                  <ArrowRight className="h-4 w-4" />
                </Button>
              </div>
            </form>
          )}
        </div>
      </div>
    </div>
  );
}
