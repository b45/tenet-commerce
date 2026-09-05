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
      <div className="flex h-64 items-center justify-center rounded-2xl border border-neutral-200 bg-white/50 dark:border-neutral-800 dark:bg-neutral-900/50">
        <div className="flex flex-col items-center gap-3">
          <div className="h-8 w-8 animate-spin rounded-full border-2 border-emerald-600 border-t-transparent" />
          <span className="text-xs text-neutral-500">Memuat inventori...</span>
        </div>
      </div>
    );
  }

  if (products.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center rounded-2xl border border-dashed border-neutral-200 bg-white/40 px-6 py-16 text-center dark:border-neutral-800 dark:bg-neutral-900/40">
        <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-neutral-100 text-neutral-400 dark:bg-neutral-800 dark:text-neutral-500">
          <Package className="h-6 w-6" aria-hidden="true" />
        </div>
        <h3 className="mt-4 text-base font-semibold text-neutral-900 dark:text-neutral-100">
          {t("inventory.table.emptyTitle")}
        </h3>
        <p className="mt-1 text-sm text-neutral-500 dark:text-neutral-400">
          {t("inventory.table.emptyDescription")}
        </p>
      </div>
    );
  }

  return (
    <div className="overflow-hidden rounded-2xl border border-neutral-200 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-900">
      {/* Desktop Table View */}
      <div className="hidden lg:block overflow-x-auto">
        <table className="w-full text-left text-sm rtl:text-right">
          <thead className="border-b border-neutral-200 bg-neutral-50/75 text-xs font-semibold uppercase tracking-wider text-neutral-500 dark:border-neutral-800 dark:bg-neutral-900/80 dark:text-neutral-400">
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
          <tbody className="divide-y divide-neutral-200 dark:divide-neutral-800">
            {products.map((item) => {
              const threshold = item.reorder_threshold ?? 5;
              const isOutOfStock = item.stock_quantity <= 0;
              const isLowStock = !isOutOfStock && item.stock_quantity <= threshold;

              return (
                <tr
                  key={item.id}
                  className="transition hover:bg-neutral-50/50 dark:hover:bg-neutral-800/50"
                >
                  {/* Name & Halal Badge */}
                  <td className="px-6 py-4">
                    <div className="flex flex-col gap-1">
                      <div className="flex items-center gap-2">
                        <span className="font-medium text-neutral-900 dark:text-neutral-100">
                          {item.name}
                        </span>
                        {item.is_halal_certified && (
                          <span
                            className="inline-flex items-center gap-1 rounded-full bg-emerald-500/10 px-2 py-0.5 text-[11px] font-medium text-emerald-700 dark:bg-emerald-500/20 dark:text-emerald-300"
                            title="Tersertifikasi Halal Resmi"
                          >
                            <ShieldCheck className="h-3 w-3" aria-hidden="true" />
                            <span>{t("inventory.table.halalBadge")}</span>
                          </span>
                        )}
                      </div>
                      {item.description && (
                        <p className="line-clamp-1 text-xs text-neutral-400">
                          {item.description}
                        </p>
                      )}
                    </div>
                  </td>

                  {/* SKU / Barcode */}
                  <td className="px-6 py-4">
                    <div className="flex flex-col font-mono text-xs">
                      <span className="font-semibold text-neutral-800 dark:text-neutral-200">
                        {item.sku}
                      </span>
                      {item.barcode && (
                        <span className="text-neutral-400">{item.barcode}</span>
                      )}
                    </div>
                  </td>

                  {/* Category */}
                  <td className="px-6 py-4 text-xs text-neutral-600 dark:text-neutral-300">
                    {item.category_name || "—"}
                  </td>

                  {/* Cost Price */}
                  <td className="px-6 py-4 text-xs font-mono text-neutral-500 dark:text-neutral-400">
                    {formatIDR(item.cost_price || 0)}
                  </td>

                  {/* Sale / Unit Price */}
                  <td className="px-6 py-4 text-xs font-mono font-semibold text-neutral-900 dark:text-neutral-100">
                    {formatIDR(item.unit_price)}
                  </td>

                  {/* Stock Quantity */}
                  <td className="px-6 py-4">
                    <div className="inline-flex items-center gap-1.5">
                      <span
                        className={`inline-flex items-center gap-1 rounded-lg px-2.5 py-1 text-xs font-mono font-medium ${
                          isOutOfStock
                            ? "bg-rose-500/10 text-rose-700 dark:bg-rose-500/20 dark:text-rose-400"
                            : isLowStock
                            ? "bg-amber-500/10 text-amber-700 dark:bg-amber-500/20 dark:text-amber-400"
                            : "bg-emerald-500/10 text-emerald-700 dark:bg-emerald-500/20 dark:text-emerald-400"
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
                        item.is_active ? "bg-emerald-500" : "bg-neutral-400"
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
                        className="flex h-8 w-8 items-center justify-center rounded-lg border border-neutral-200 bg-white text-neutral-600 transition hover:border-emerald-500 hover:text-emerald-600 disabled:cursor-not-allowed disabled:opacity-40 dark:border-neutral-800 dark:bg-neutral-800 dark:text-neutral-300 dark:hover:border-emerald-500 dark:hover:text-emerald-400"
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
                        className="flex h-8 w-8 items-center justify-center rounded-lg border border-neutral-200 bg-white text-neutral-600 transition hover:border-blue-500 hover:text-blue-600 disabled:cursor-not-allowed disabled:opacity-40 dark:border-neutral-800 dark:bg-neutral-800 dark:text-neutral-300 dark:hover:border-blue-500 dark:hover:text-blue-400"
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
                        className="flex h-8 w-8 items-center justify-center rounded-lg border border-neutral-200 bg-white text-neutral-600 transition hover:border-rose-500 hover:text-rose-600 disabled:cursor-not-allowed disabled:opacity-40 dark:border-neutral-800 dark:bg-neutral-800 dark:text-neutral-300 dark:hover:border-rose-500 dark:hover:text-rose-400"
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
      <div className="divide-y divide-neutral-200 lg:hidden dark:divide-neutral-800">
        {products.map((item) => {
          const threshold = item.reorder_threshold ?? 5;
          const isOutOfStock = item.stock_quantity <= 0;
          const isLowStock = !isOutOfStock && item.stock_quantity <= threshold;

          return (
            <article key={item.id} className="p-4 space-y-3">
              <div className="flex items-start justify-between gap-2">
                <div>
                  <div className="flex items-center gap-2">
                    <h4 className="font-medium text-neutral-900 dark:text-neutral-100">
                      {item.name}
                    </h4>
                    {item.is_halal_certified && (
                      <span className="inline-flex items-center gap-1 rounded-full bg-emerald-500/10 px-2 py-0.5 text-[10px] font-medium text-emerald-700 dark:bg-emerald-500/20 dark:text-emerald-300">
                        <ShieldCheck className="h-3 w-3" aria-hidden="true" />
                        <span>{t("inventory.table.halalBadge")}</span>
                      </span>
                    )}
                  </div>
                  <span className="font-mono text-xs text-neutral-500">
                    SKU: {item.sku} {item.barcode ? `• Barcode: ${item.barcode}` : ""}
                  </span>
                </div>

                <span
                  className={`inline-flex items-center gap-1 rounded-lg px-2 py-0.5 text-xs font-mono font-medium ${
                    isOutOfStock
                      ? "bg-rose-500/10 text-rose-700 dark:bg-rose-500/20 dark:text-rose-400"
                      : isLowStock
                      ? "bg-amber-500/10 text-amber-700 dark:bg-amber-500/20 dark:text-amber-400"
                      : "bg-emerald-500/10 text-emerald-700 dark:bg-emerald-500/20 dark:text-emerald-400"
                  }`}
                >
                  {isLowStock && <AlertCircle className="h-3 w-3" aria-hidden="true" />}
                  <span>{item.stock_quantity} unit</span>
                </span>
              </div>

              <div className="flex items-center justify-between text-xs">
                <span className="text-neutral-500">
                  {item.category_name || "Umum"}
                </span>
                <span className="font-mono font-semibold text-neutral-900 dark:text-neutral-100">
                  {formatIDR(item.unit_price)}
                </span>
              </div>

              {/* Mobile Actions */}
              <div className="flex items-center justify-end gap-2 border-t border-neutral-100 pt-2.5 dark:border-neutral-800">
                <button
                  type="button"
                  onClick={() => onAdjustStock(item)}
                  disabled={!canWrite}
                  className="flex items-center gap-1 rounded-lg border border-neutral-200 px-2.5 py-1.5 text-xs text-neutral-700 disabled:opacity-40 dark:border-neutral-800 dark:text-neutral-300"
                >
                  <Sliders className="h-3.5 w-3.5" aria-hidden="true" />
                  <span>{t("inventory.table.adjustAction")}</span>
                </button>
                <button
                  type="button"
                  onClick={() => onEditProduct(item)}
                  disabled={!canWrite}
                  className="flex items-center gap-1 rounded-lg border border-neutral-200 px-2.5 py-1.5 text-xs text-neutral-700 disabled:opacity-40 dark:border-neutral-800 dark:text-neutral-300"
                >
                  <Edit2 className="h-3.5 w-3.5" aria-hidden="true" />
                  <span>{t("inventory.table.editAction")}</span>
                </button>
                <button
                  type="button"
                  onClick={() => onDeleteProduct(item)}
                  disabled={!canWrite}
                  className="flex items-center gap-1 rounded-lg border border-rose-200 px-2.5 py-1.5 text-xs text-rose-600 disabled:opacity-40 dark:border-rose-900/30 dark:text-rose-400"
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
