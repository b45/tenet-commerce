"use client";

import * as React from "react";
import { Edit2, Sliders, Trash2, ShieldCheck, AlertCircle, Package } from "lucide-react";
import { useTranslation } from "@/lib/i18n";
import { formatIDR } from "@/lib/money";
import type { InventoryProduct } from "../types";

interface InventoryTableProps {
  products: InventoryProduct[];
  onAdjustStock: (product: InventoryProduct) => void;
  onEditProduct: (product: InventoryProduct) => void;
  onDeleteProduct: (product: InventoryProduct) => void;
  canWrite: boolean;
  isLoading: boolean;
}

export function InventoryTable({
  products,
  onAdjustStock,
  onEditProduct,
  onDeleteProduct,
  canWrite,
  isLoading,
}: InventoryTableProps) {
  const { t } = useTranslation();

  if (isLoading && products.length === 0) {
    return (
      <div className="flex h-64 items-center justify-center rounded-2xl border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] shadow-[var(--shadow-card)]">
        <div className="flex flex-col items-center gap-3">
          <div className="h-8 w-8 animate-spin rounded-full border-2 border-[var(--color-action-primary)] border-t-transparent" />
          <span className="text-xs text-[var(--color-text-muted)]">Memuat inventori...</span>
        </div>
      </div>
    );
  }

  if (products.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center rounded-2xl border border-dashed border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] px-6 py-16 text-center shadow-[var(--shadow-card)]">
        <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-[var(--color-surface-muted)] text-[var(--color-text-muted)]">
          <Package className="h-6 w-6" aria-hidden="true" />
        </div>
        <h3 className="mt-4 text-base font-semibold text-[var(--color-text-primary)]">
          {t("inventory.table.emptyTitle")}
        </h3>
        <p className="mt-1 text-sm text-[var(--color-text-secondary)]">
          {t("inventory.table.emptyDescription")}
        </p>
      </div>
    );
  }

  return (
    <div className="overflow-hidden rounded-2xl border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] shadow-[var(--shadow-card)]">
      {/* Desktop Table View */}
      <div className="hidden lg:block overflow-x-auto">
        <table className="w-full text-left text-sm rtl:text-right">
          <thead className="border-b border-[var(--color-border-hairline)] bg-[var(--color-surface-muted)] text-xs font-semibold uppercase tracking-wider text-[var(--color-text-secondary)] select-none">
            <tr>
              <th scope="col" className="px-6 py-3.5">
                {t("inventory.table.productName")}
              </th>
              <th scope="col" className="px-6 py-3.5">
                {t("inventory.table.sku")}
              </th>
              <th scope="col" className="px-6 py-3.5">
                {t("inventory.table.category")}
              </th>
              <th scope="col" className="px-6 py-3.5">
                {t("inventory.table.costPrice")}
              </th>
              <th scope="col" className="px-6 py-3.5">
                {t("inventory.table.unitPrice")}
              </th>
              <th scope="col" className="px-6 py-3.5">
                {t("inventory.table.stock")}
              </th>
              <th scope="col" className="px-6 py-3.5">
                {t("inventory.table.status")}
              </th>
              <th scope="col" className="px-6 py-3.5 text-right rtl:text-left">
                {t("inventory.table.actions")}
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-[var(--color-border-hairline)]">
            {products.map((item) => {
              const threshold = item.reorder_threshold ?? 5;
              const isOutOfStock = item.stock_quantity <= 0;
              const isLowStock = !isOutOfStock && item.stock_quantity <= threshold;

              return (
                <tr
                  key={item.id}
                  className="transition-colors hover:bg-[var(--color-surface-muted)]/60"
                >
                  {/* Name & Halal Badge */}
                  <td className="px-6 py-4">
                    <div className="flex flex-col gap-1">
                      <div className="flex items-center gap-2">
                        <span className="font-medium text-[var(--color-text-primary)]">
                          {item.name}
                        </span>
                        {item.is_halal_certified && (
                          <span
                            className="inline-flex items-center gap-1 rounded-full bg-[var(--color-status-success-bg)] px-2 py-0.5 text-[11px] font-medium text-[var(--color-status-success-text)] border border-[var(--color-status-success-border)]"
                            title="Tersertifikasi Halal Resmi"
                          >
                            <ShieldCheck className="h-3 w-3" aria-hidden="true" />
                            <span>{t("inventory.table.halalBadge")}</span>
                          </span>
                        )}
                      </div>
                      {item.description && (
                        <p className="line-clamp-1 text-xs text-[var(--color-text-muted)]">
                          {item.description}
                        </p>
                      )}
                    </div>
                  </td>

                  {/* SKU / Barcode */}
                  <td className="px-6 py-4">
                    <div className="flex flex-col font-mono text-xs">
                      <span className="font-semibold text-[var(--color-text-primary)]">
                        {item.sku}
                      </span>
                      {item.barcode && (
                        <span className="text-[var(--color-text-muted)]">{item.barcode}</span>
                      )}
                    </div>
                  </td>

                  {/* Category */}
                  <td className="px-6 py-4 text-xs text-[var(--color-text-secondary)]">
                    {item.category_name || "—"}
                  </td>

                  {/* Cost Price */}
                  <td className="px-6 py-4 text-xs font-mono text-[var(--color-text-secondary)]">
                    {formatIDR(item.cost_price || 0)}
                  </td>

                  {/* Sale / Unit Price */}
                  <td className="px-6 py-4 text-xs font-mono font-semibold text-[var(--color-text-primary)]">
                    {formatIDR(item.unit_price)}
                  </td>

                  {/* Stock Quantity */}
                  <td className="px-6 py-4">
                    <div className="inline-flex items-center gap-1.5">
                      <span
                        className={`inline-flex items-center gap-1 rounded-lg px-2.5 py-1 text-xs font-mono font-medium border ${
                          isOutOfStock
                            ? "bg-[var(--color-status-danger-bg)] text-[var(--color-status-danger-text)] border-[var(--color-status-danger-border)]"
                            : isLowStock
                            ? "bg-[var(--color-status-warning-bg)] text-[var(--color-status-warning-text)] border-[var(--color-status-warning-border)]"
                            : "bg-[var(--color-status-success-bg)] text-[var(--color-status-success-text)] border-[var(--color-status-success-border)]"
                        }`}
                      >
                        {isLowStock && <AlertCircle className="h-3 w-3" aria-hidden="true" />}
                        <span>{item.stock_quantity} unit</span>
                      </span>
                    </div>
                  </td>

                  {/* Status */}
                  <td className="px-6 py-4">
                    <span
                      className={`inline-block h-2 w-2 rounded-full ${
                        item.is_active ? "bg-emerald-500" : "bg-neutral-300"
                      }`}
                      title={
                        item.is_active
                          ? t("inventory.table.active")
                          : t("inventory.table.inactive")
                      }
                    />
                  </td>

                  {/* Actions */}
                  <td className="px-6 py-4 text-right rtl:text-left">
                    <div className="flex items-center justify-end gap-1.5 rtl:justify-start">
                      {/* Adjust Stock Button */}
                      <button
                        type="button"
                        onClick={() => onAdjustStock(item)}
                        disabled={!canWrite}
                        title={
                          canWrite
                            ? t("inventory.table.adjustAction")
                            : t("inventory.permissions.readOnlyTooltip")
                        }
                        className="flex h-8 w-8 items-center justify-center rounded-lg border border-[var(--color-border-hairline)] bg-white text-[var(--color-text-secondary)] shadow-sm transition hover:border-[var(--color-action-primary)]/40 hover:text-[var(--color-action-primary)] hover:bg-[var(--color-surface-muted)] disabled:cursor-not-allowed disabled:opacity-40"
                      >
                        <Sliders className="h-3.5 w-3.5" aria-hidden="true" />
                      </button>

                      {/* Edit Button */}
                      <button
                        type="button"
                        onClick={() => onEditProduct(item)}
                        disabled={!canWrite}
                        title={
                          canWrite
                            ? t("inventory.table.editAction")
                            : t("inventory.permissions.readOnlyTooltip")
                        }
                        className="flex h-8 w-8 items-center justify-center rounded-lg border border-[var(--color-border-hairline)] bg-white text-[var(--color-text-secondary)] shadow-sm transition hover:border-[var(--color-action-primary)]/40 hover:text-[var(--color-action-primary)] hover:bg-[var(--color-surface-muted)] disabled:cursor-not-allowed disabled:opacity-40"
                      >
                        <Edit2 className="h-3.5 w-3.5" aria-hidden="true" />
                      </button>

                      {/* Delete / Deactivate Button */}
                      <button
                        type="button"
                        onClick={() => onDeleteProduct(item)}
                        disabled={!canWrite}
                        title={
                          canWrite
                            ? t("inventory.table.deleteAction")
                            : t("inventory.permissions.readOnlyTooltip")
                        }
                        className="flex h-8 w-8 items-center justify-center rounded-lg border border-[var(--color-border-hairline)] bg-white text-[var(--color-text-secondary)] shadow-sm transition hover:border-[var(--color-status-danger-border)] hover:text-[var(--color-status-danger-text)] hover:bg-[var(--color-surface-muted)] disabled:cursor-not-allowed disabled:opacity-40"
                      >
                        <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
                      </button>
                    </div>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      {/* Mobile Card Grid View */}
      <div className="divide-y divide-[var(--color-border-hairline)] lg:hidden">
        {products.map((item) => {
          const threshold = item.reorder_threshold ?? 5;
          const isOutOfStock = item.stock_quantity <= 0;
          const isLowStock = !isOutOfStock && item.stock_quantity <= threshold;

          return (
            <article key={item.id} className="p-4 space-y-3 bg-white">
              <div className="flex items-start justify-between gap-2">
                <div>
                  <div className="flex items-center gap-2">
                    <h4 className="font-medium text-[var(--color-text-primary)]">
                      {item.name}
                    </h4>
                    {item.is_halal_certified && (
                      <span className="inline-flex items-center gap-1 rounded-full bg-[var(--color-status-success-bg)] px-2 py-0.5 text-[10px] font-medium text-[var(--color-status-success-text)] border border-[var(--color-status-success-border)]">
                        <ShieldCheck className="h-3 w-3" aria-hidden="true" />
                        <span>{t("inventory.table.halalBadge")}</span>
                      </span>
                    )}
                  </div>
                  <span className="font-mono text-xs text-[var(--color-text-muted)]">
                    SKU: {item.sku} {item.barcode ? `• Barcode: ${item.barcode}` : ""}
                  </span>
                </div>

                <span
                  className={`inline-flex items-center gap-1 rounded-lg px-2 py-0.5 text-xs font-mono font-medium border ${
                    isOutOfStock
                      ? "bg-[var(--color-status-danger-bg)] text-[var(--color-status-danger-text)] border-[var(--color-status-danger-border)]"
                      : isLowStock
                      ? "bg-[var(--color-status-warning-bg)] text-[var(--color-status-warning-text)] border-[var(--color-status-warning-border)]"
                      : "bg-[var(--color-status-success-bg)] text-[var(--color-status-success-text)] border-[var(--color-status-success-border)]"
                  }`}
                >
                  {isLowStock && <AlertCircle className="h-3 w-3" aria-hidden="true" />}
                  <span>{item.stock_quantity} unit</span>
                </span>
              </div>

              <div className="flex items-center justify-between text-xs">
                <span className="text-[var(--color-text-secondary)]">
                  {item.category_name || "Umum"}
                </span>
                <span className="font-mono font-semibold text-[var(--color-text-primary)]">
                  {formatIDR(item.unit_price)}
                </span>
              </div>

              {/* Mobile Actions */}
              <div className="flex items-center justify-end gap-2 border-t border-[var(--color-border-hairline)] pt-2.5">
                <button
                  type="button"
                  onClick={() => onAdjustStock(item)}
                  disabled={!canWrite}
                  className="flex items-center gap-1 rounded-lg border border-[var(--color-border-hairline)] bg-white px-2.5 py-1.5 text-xs text-[var(--color-text-secondary)] shadow-sm transition hover:bg-[var(--color-surface-muted)] disabled:opacity-40"
                >
                  <Sliders className="h-3.5 w-3.5" aria-hidden="true" />
                  <span>{t("inventory.table.adjustAction")}</span>
                </button>
                <button
                  type="button"
                  onClick={() => onEditProduct(item)}
                  disabled={!canWrite}
                  className="flex items-center gap-1 rounded-lg border border-[var(--color-border-hairline)] bg-white px-2.5 py-1.5 text-xs text-[var(--color-text-secondary)] shadow-sm transition hover:bg-[var(--color-surface-muted)] disabled:opacity-40"
                >
                  <Edit2 className="h-3.5 w-3.5" aria-hidden="true" />
                  <span>{t("inventory.table.editAction")}</span>
                </button>
                <button
                  type="button"
                  onClick={() => onDeleteProduct(item)}
                  disabled={!canWrite}
                  className="flex items-center gap-1 rounded-lg border border-[var(--color-status-danger-border)] bg-white px-2.5 py-1.5 text-xs text-[var(--color-status-danger-text)] shadow-sm transition hover:bg-[var(--color-status-danger-bg)] disabled:opacity-40"
                >
                  <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
                </button>
              </div>
            </article>
          );
        })}
      </div>
    </div>
  );
}
