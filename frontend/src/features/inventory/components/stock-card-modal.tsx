"use client";

import * as React from "react";
import { X, BookOpen, Calendar, ArrowDownRight, ArrowUpRight, RotateCcw, AlertCircle, Loader2 } from "lucide-react";
import { useTranslation } from "@/lib/i18n";
import type { InventoryProduct, StockCardResponse, StockCardItem } from "../types";

interface StockCardModalProps {
  isOpen: boolean;
  onClose: () => void;
  product: InventoryProduct | null;
  onFetchStockCard: (
    productId: string,
    startDate?: string,
    endDate?: string,
    limit?: number,
    offset?: number
  ) => Promise<{ success: boolean; data?: StockCardResponse; error?: string }>;
}

export function StockCardModal({
  isOpen,
  onClose,
  product,
  onFetchStockCard,
}: StockCardModalProps) {
  const { t } = useTranslation();
  const dialogRef = React.useRef<HTMLDialogElement>(null);

  const [loading, setLoading] = React.useState(false);
  const [cardData, setCardData] = React.useState<StockCardResponse | null>(null);
  const [errorMessage, setErrorMessage] = React.useState<string | null>(null);

  // Date filters
  const [startDate, setStartDate] = React.useState("");
  const [endDate, setEndDate] = React.useState("");

  const loadStockCard = React.useCallback(
    async (start?: string, end?: string) => {
      if (!product) return;
      setLoading(true);
      setErrorMessage(null);
      try {
        const res = await onFetchStockCard(product.id, start || undefined, end || undefined);
        if (res.success && res.data) {
          setCardData(res.data);
        } else {
          setErrorMessage(res.error || t("inventory.stockCardModal.errorFetch"));
        }
      } catch (err: unknown) {
        const msg = err instanceof Error ? err.message : t("inventory.stockCardModal.errorFetch");
        setErrorMessage(msg);
      } finally {
        setLoading(false);
      }
    },
    [product, onFetchStockCard, t]
  );

  // Trigger load when opened
  React.useEffect(() => {
    if (isOpen && product) {
      setStartDate("");
      setEndDate("");
      loadStockCard();
    }
  }, [isOpen, product, loadStockCard]);

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

  const handleFilterSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    loadStockCard(startDate, endDate);
  };

  const handleResetFilter = () => {
    setStartDate("");
    setEndDate("");
    loadStockCard();
  };

  const formatMovementType = (type: string) => {
    switch (type) {
      case "OPENING":
        return t("inventory.stockCardModal.movementTypes.opening");
      case "IN":
        return t("inventory.stockCardModal.movementTypes.inbound");
      case "OUT":
        return t("inventory.stockCardModal.movementTypes.outbound");
      case "ADJUSTMENT":
        return t("inventory.stockCardModal.movementTypes.adjustment");
      default:
        return type;
    }
  };

  const formatSourceDoc = (type: string, docId?: string | null) => {
    let typeLabel = type;
    if (type === "PURCHASE_ORDER") typeLabel = t("inventory.stockCardModal.sourceTypes.purchaseOrder");
    else if (type === "GOODS_RECEIPT") typeLabel = t("inventory.stockCardModal.sourceTypes.goodsReceipt");
    else if (type === "POS_TRANSACTION") typeLabel = t("inventory.stockCardModal.sourceTypes.posTransaction");
    else if (type === "STOCK_OPNAME") typeLabel = t("inventory.stockCardModal.sourceTypes.stockOpname");
    else if (type === "MANUAL") typeLabel = t("inventory.stockCardModal.sourceTypes.manual");

    if (docId) {
      const shortId = docId.length > 8 ? `${docId.slice(0, 8)}...` : docId;
      return `${typeLabel} (${shortId})`;
    }
    return typeLabel;
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
      className="m-auto w-full max-w-4xl rounded-2xl border border-[var(--color-border-subtle)] bg-[var(--color-surface-base)] p-0 shadow-2xl backdrop:bg-black/40 backdrop:backdrop-blur-sm"
    >
      <div className="flex flex-col max-h-[90vh]">
        {/* Modal Header */}
        <div className="flex items-center justify-between border-b border-[var(--color-border-hairline)] px-6 py-4">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-blue-50 text-[#0066CC]">
              <BookOpen className="h-5 w-5" aria-hidden="true" />
            </div>
            <div>
              <h3 className="text-base font-semibold text-[var(--color-text-primary)]">
                {t("inventory.stockCardModal.title")}
              </h3>
              <p className="text-xs text-[var(--color-text-secondary)]">
                {t("inventory.stockCardModal.subtitle", { name: product.name, sku: product.sku })}
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg p-1.5 text-[var(--color-text-muted)] hover:bg-[var(--color-surface-muted)] hover:text-[var(--color-text-primary)]"
            aria-label={t("inventory.stockCardModal.close")}
          >
            <X className="h-5 w-5" aria-hidden="true" />
          </button>
        </div>

        {/* Balance Metric Strip */}
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 bg-[var(--color-surface-muted)] px-6 py-3 border-b border-[var(--color-border-hairline)]">
          <div>
            <div className="text-[11px] text-[var(--color-text-secondary)]">
              {t("inventory.stockCardModal.openingBalance")}
            </div>
            <div className="font-mono text-base font-bold text-[var(--color-text-primary)]">
              {cardData ? cardData.opening_balance : "-"} <span className="text-xs font-normal">{t("inventory.unit")}</span>
            </div>
          </div>
          <div>
            <div className="text-[11px] text-[var(--color-text-secondary)]">
              {t("inventory.stockCardModal.closingBalance")}
            </div>
            <div className="font-mono text-base font-bold text-[var(--color-text-primary)]">
              {cardData ? cardData.closing_balance : "-"} <span className="text-xs font-normal">{t("inventory.unit")}</span>
            </div>
          </div>
          <div>
            <div className="text-[11px] text-[var(--color-text-secondary)]">
              {t("inventory.stockCardModal.currentStock")}
            </div>
            <div className="font-mono text-base font-bold text-emerald-600">
              {cardData ? cardData.current_on_hand : product.stock_quantity} <span className="text-xs font-normal">{t("inventory.unit")}</span>
            </div>
          </div>
          <div>
            <div className="text-[11px] text-[var(--color-text-secondary)]">
              {t("inventory.stockCardModal.totalMovements")}
            </div>
            <div className="font-mono text-base font-bold text-[var(--color-text-primary)]">
              {cardData ? cardData.total_movements : 0}
            </div>
          </div>
        </div>

        {/* Filter Toolbar */}
        <form onSubmit={handleFilterSubmit} className="flex flex-wrap items-center gap-2.5 px-6 py-3 border-b border-[var(--color-border-hairline)] bg-white">
          <div className="flex items-center gap-1.5 text-xs text-[var(--color-text-secondary)]">
            <Calendar className="h-3.5 w-3.5" aria-hidden="true" />
            <span>{t("inventory.stockCardModal.filterDate")}:</span>
          </div>

          <div className="flex items-center gap-2">
            <input
              type="date"
              value={startDate}
              onChange={(e) => setStartDate(e.target.value)}
              aria-label={t("inventory.stockCardModal.startDate")}
              className="h-8 rounded-lg border border-black/[0.1] px-2.5 text-xs text-[var(--color-text-primary)] focus:border-[#0066CC] focus:outline-none"
            />
            <span className="text-xs text-neutral-400">-</span>
            <input
              type="date"
              value={endDate}
              onChange={(e) => setEndDate(e.target.value)}
              aria-label={t("inventory.stockCardModal.endDate")}
              className="h-8 rounded-lg border border-black/[0.1] px-2.5 text-xs text-[var(--color-text-primary)] focus:border-[#0066CC] focus:outline-none"
            />
          </div>

          <div className="flex items-center gap-1.5 ml-auto">
            <button
              type="submit"
              disabled={loading}
              className="h-8 rounded-lg bg-[#0066CC] px-3 text-xs font-medium text-white hover:bg-[#0055B3] transition disabled:opacity-50"
            >
              {t("inventory.stockCardModal.applyFilter")}
            </button>
            {(startDate || endDate) && (
              <button
                type="button"
                onClick={handleResetFilter}
                disabled={loading}
                className="flex items-center gap-1 h-8 rounded-lg border border-black/[0.1] px-2.5 text-xs text-[var(--color-text-secondary)] hover:bg-[#F5F5F7] transition"
              >
                <RotateCcw className="h-3 w-3" aria-hidden="true" />
                <span>{t("inventory.stockCardModal.resetFilter")}</span>
              </button>
            )}
          </div>
        </form>

        {/* Content Area / Table */}
        <div className="flex-1 overflow-y-auto p-6">
          {errorMessage && (
            <div className="mb-4 rounded-xl border border-rose-500/20 bg-rose-50 p-3 text-xs text-rose-700">
              {errorMessage}
            </div>
          )}

          {loading ? (
            <div className="flex flex-col items-center justify-center py-12 text-center text-xs text-[var(--color-text-secondary)]">
              <Loader2 className="h-6 w-6 animate-spin text-[#0066CC] mb-2" aria-hidden="true" />
              <span>{t("inventory.stockCardModal.loading")}</span>
            </div>
          ) : !cardData || cardData.movements.length === 0 ? (
            <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-black/[0.08] py-12 text-center">
              <AlertCircle className="h-8 w-8 text-neutral-300 mb-2" aria-hidden="true" />
              <p className="text-xs text-[var(--color-text-secondary)]">
                {t("inventory.stockCardModal.emptyMovements")}
              </p>
            </div>
          ) : (
            <div className="overflow-x-auto rounded-xl border border-black/[0.06]">
              <table className="w-full text-left text-xs rtl:text-right">
                <thead className="border-b border-black/[0.06] bg-[#F8F8FA] text-[11px] font-semibold uppercase tracking-wider text-[#555D6E] select-none">
                  <tr>
                    <th scope="col" className="px-4 py-2.5">{t("inventory.stockCardModal.columns.dateTime")}</th>
                    <th scope="col" className="px-4 py-2.5">{t("inventory.stockCardModal.columns.type")}</th>
                    <th scope="col" className="px-4 py-2.5">{t("inventory.stockCardModal.columns.sourceDoc")}</th>
                    <th scope="col" className="px-4 py-2.5 text-right rtl:text-left">{t("inventory.stockCardModal.columns.quantityDelta")}</th>
                    <th scope="col" className="px-4 py-2.5 text-right rtl:text-left">{t("inventory.stockCardModal.columns.runningBalance")}</th>
                    <th scope="col" className="px-4 py-2.5">{t("inventory.stockCardModal.columns.actorOrReason")}</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-black/[0.04]">
                  {cardData.movements.map((mov: StockCardItem) => {
                    const isPositive = mov.quantity_delta > 0;
                    const isNegative = mov.quantity_delta < 0;
                    const dateStr = new Date(mov.occurred_at).toLocaleString([], {
                      year: "numeric",
                      month: "short",
                      day: "numeric",
                      hour: "2-digit",
                      minute: "2-digit",
                    });

                    return (
                      <tr key={mov.movement_id} className="hover:bg-[#F8F8FA]/60 transition-colors">
                        <td className="px-4 py-2.5 font-mono text-[11px] text-[var(--color-text-secondary)] whitespace-nowrap">
                          {dateStr}
                        </td>
                        <td className="px-4 py-2.5">
                          <span
                            className={`inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-[10px] font-semibold uppercase ${
                              mov.movement_type === "IN" || mov.movement_type === "OPENING"
                                ? "bg-emerald-50 text-emerald-700 border border-emerald-200/60"
                                : mov.movement_type === "OUT"
                                ? "bg-rose-50 text-rose-700 border border-rose-200/60"
                                : "bg-blue-50 text-blue-700 border border-blue-200/60"
                            }`}
                          >
                            {isPositive && <ArrowDownRight className="h-3 w-3" aria-hidden="true" />}
                            {isNegative && <ArrowUpRight className="h-3 w-3" aria-hidden="true" />}
                            {formatMovementType(mov.movement_type)}
                          </span>
                        </td>
                        <td className="px-4 py-2.5 text-[11px] font-mono text-[var(--color-text-primary)]">
                          {formatSourceDoc(mov.source_document_type, mov.source_document_id)}
                        </td>
                        <td className={`px-4 py-2.5 text-right rtl:text-left font-mono font-semibold ${
                          isPositive ? "text-emerald-600" : isNegative ? "text-rose-600" : "text-neutral-600"
                        }`}>
                          {isPositive ? `+${mov.quantity_delta}` : mov.quantity_delta}
                        </td>
                        <td className="px-4 py-2.5 text-right rtl:text-left font-mono font-bold text-[var(--color-text-primary)]">
                          {mov.running_balance}
                        </td>
                        <td className="px-4 py-2.5 text-[11px] text-[var(--color-text-secondary)] truncate max-w-xs">
                          {mov.reason || "-"}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </div>

        {/* Modal Footer */}
        <div className="flex items-center justify-end border-t border-[var(--color-border-hairline)] px-6 py-3 bg-[var(--color-surface-muted)]">
          <button
            type="button"
            onClick={onClose}
            className="h-9 rounded-xl bg-white border border-black/[0.08] px-4 text-xs font-medium text-[var(--color-text-primary)] hover:bg-[#F5F5F7] transition"
          >
            {t("inventory.stockCardModal.close")}
          </button>
        </div>
      </div>
    </dialog>
  );
}
