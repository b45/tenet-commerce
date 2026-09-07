"use client";

import * as React from "react";
import { Edit2, Sliders, Trash2, ShieldCheck, AlertCircle, Package, BookOpen, Truck } from "lucide-react";
import { useTranslation } from "@/lib/i18n";
import { formatIDR } from "@/lib/money";
import { Skeleton } from "@/components/ui/skeleton";
import type { InventoryProduct } from "../types";

interface InventoryTableProps {
  products: InventoryProduct[];
  onAdjustStock: (product: InventoryProduct) => void;
  onEditProduct: (product: InventoryProduct) => void;
  onDeleteProduct: (product: InventoryProduct) => void;
  onOpenStockCard: (product: InventoryProduct) => void;
  onOpenProcureDraft: (product: InventoryProduct) => void;
  canWrite: boolean;
  isLoading: boolean;
}

export function InventoryTable({
  products,
  onAdjustStock,
  onEditProduct,
  onDeleteProduct,
  onOpenStockCard,
  onOpenProcureDraft,
  canWrite,
  isLoading,
}: InventoryTableProps) {
  const { t } = useTranslation();

  if (isLoading && products.length === 0) {
    return (
      <div className="overflow-hidden rounded-2xl border border-black/[0.06] bg-white shadow-[0_1px_2px_rgba(0,0,0,0.02)]">
        {/* Desktop Table Skeleton */}
        <div className="hidden lg:block overflow-x-auto">
          <table className="w-full text-left text-xs rtl:text-right">
            <thead className="border-b border-black/[0.06] bg-[#F8F8FA] text-[11px] font-semibold uppercase tracking-wider text-[#555D6E] select-none">
              <tr>
                <th scope="col" className="px-5 py-3">
                  {t("inventory.table.productName")}
                </th>
                <th scope="col" className="px-5 py-3">
                  {t("inventory.table.sku")}
                </th>
                <th scope="col" className="px-5 py-3">
                  {t("inventory.table.category")}
                </th>
                <th scope="col" className="px-5 py-3">
                  {t("inventory.table.costPrice")}
                </th>
                <th scope="col" className="px-5 py-3">
                  {t("inventory.table.unitPrice")}
                </th>
                <th scope="col" className="px-5 py-3">
                  {t("inventory.table.stock")}
                </th>
                <th scope="col" className="px-5 py-3">
                  {t("inventory.table.status")}
                </th>
                <th scope="col" className="px-5 py-3 text-right rtl:text-left">
                  {t("inventory.table.actions")}
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-black/[0.05]">
              {[1, 2, 3, 4, 5].map((row) => (
                <tr key={row} className="hover:bg-[#F8F8FA]/40">
                  <td className="px-5 py-3.5">
                    <div className="flex flex-col gap-1.5">
                      <div className="flex items-center gap-2">
                        <Skeleton className={`h-4 rounded ${row % 2 === 0 ? "w-36" : "w-44"}`} />
                        {row % 2 === 1 && <Skeleton className="h-4 w-12 rounded-full" />}
                      </div>
                      <Skeleton className={`h-3 rounded ${row % 2 === 0 ? "w-48" : "w-32"}`} />
                    </div>
                  </td>
                  <td className="px-5 py-3.5">
                    <div className="flex flex-col gap-1">
                      <Skeleton className="h-3.5 w-20 rounded" />
                      <Skeleton className="h-2.5 w-16 rounded" />
                    </div>
                  </td>
                  <td className="px-5 py-3.5">
                    <Skeleton className="h-3.5 w-16 rounded" />
                  </td>
                  <td className="px-5 py-3.5">
                    <Skeleton className="h-3.5 w-20 rounded" />
                  </td>
                  <td className="px-5 py-3.5">
                    <Skeleton className="h-4 w-20 rounded" />
                  </td>
                  <td className="px-5 py-3.5">
                    <Skeleton className="h-5 w-16 rounded-lg" />
                  </td>
                  <td className="px-5 py-3.5">
                    <Skeleton className="h-2.5 w-2.5 rounded-full" />
                  </td>
                  <td className="px-5 py-3.5 text-right rtl:text-left">
                    <div className="flex items-center justify-end gap-1.5 rtl:justify-start">
                      <Skeleton className="h-7 w-7 rounded-lg" />
                      <Skeleton className="h-7 w-7 rounded-lg" />
                      <Skeleton className="h-7 w-7 rounded-lg" />
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Mobile Card Skeleton */}
        <div className="divide-y divide-black/[0.05] lg:hidden">
          {[1, 2, 3].map((card) => (
            <div key={card} className="p-4 space-y-3 bg-white">
              <div className="flex items-start justify-between gap-3">
                <div className="space-y-1.5 flex-1">
                  <Skeleton className={`h-4 rounded ${card % 2 === 0 ? "w-32" : "w-40"}`} />
                  <Skeleton className="h-3 w-24 rounded" />
                </div>
                <Skeleton className="h-5 w-16 rounded-lg" />
              </div>
              <div className="flex items-center justify-between pt-1">
                <Skeleton className="h-3.5 w-16 rounded" />
                <Skeleton className="h-4 w-20 rounded" />
              </div>
              <div className="flex items-center justify-end gap-2 border-t border-black/[0.05] pt-2">
                <Skeleton className="h-6 w-16 rounded-lg" />
                <Skeleton className="h-6 w-14 rounded-lg" />
                <Skeleton className="h-6 w-7 rounded-lg" />
              </div>
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (products.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center rounded-2xl border border-dashed border-black/[0.08] bg-white/70 px-6 py-14 text-center shadow-[0_1px_2px_rgba(0,0,0,0.02)]">
        <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-[#F5F5F7] text-[#8B95A5]">
          <Package className="h-5 w-5" aria-hidden="true" />
        </div>
        <h3 className="mt-3.5 text-sm font-semibold text-[#0B0F19]">
          {t("inventory.table.emptyTitle")}
        </h3>
        <p className="mt-1 text-xs text-[#555D6E]">
          {t("inventory.table.emptyDescription")}
        </p>
      </div>
    );
  }

  return (
    <div className="overflow-hidden rounded-2xl border border-black/[0.06] bg-white shadow-[0_1px_2px_rgba(0,0,0,0.02)]">
      {/* Desktop Table View */}
      <div className="hidden lg:block overflow-x-auto">
        <table className="w-full text-left text-xs rtl:text-right">
          <thead className="border-b border-black/[0.06] bg-[#F8F8FA] text-[11px] font-semibold uppercase tracking-wider text-[#555D6E] select-none">
            <tr>
              <th scope="col" className="px-5 py-3">
                {t("inventory.table.productName")}
              </th>
              <th scope="col" className="px-5 py-3">
                {t("inventory.table.sku")}
              </th>
              <th scope="col" className="px-5 py-3">
                {t("inventory.table.category")}
              </th>
              <th scope="col" className="px-5 py-3">
                {t("inventory.table.costPrice")}
              </th>
              <th scope="col" className="px-5 py-3">
                {t("inventory.table.unitPrice")}
              </th>
              <th scope="col" className="px-5 py-3">
                {t("inventory.table.stock")}
              </th>
              <th scope="col" className="px-5 py-3">
                {t("inventory.table.status")}
              </th>
              <th scope="col" className="px-5 py-3 text-right rtl:text-left">
                {t("inventory.table.actions")}
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-black/[0.05]">
            {products.map((item) => {
              const threshold = item.reorder_threshold ?? 5;
              const isOutOfStock = item.stock_quantity <= 0;
              const isLowStock = !isOutOfStock && item.stock_quantity <= threshold;

              return (
                <tr
                  key={item.id}
                  className="transition-colors hover:bg-[#F8F8FA]/70"
                >
                  {/* Name & Halal Badge */}
                  <td className="px-5 py-3.5">
                    <div className="flex flex-col gap-0.5">
                      <div className="flex items-center gap-1.5">
                        <span className="font-semibold text-[#0B0F19]">
                          {item.name}
                        </span>
                        {item.is_halal_certified && (
                          <span
                            className="inline-flex items-center gap-1 rounded-full bg-emerald-50 px-2 py-0.5 text-[10px] font-medium text-emerald-700 border border-emerald-200/60"
                            title={t("inventory.table.halalNotice")}
                          >
                            <ShieldCheck className="h-3 w-3" aria-hidden="true" />
                            <span>{t("inventory.table.halalBadge")}</span>
                          </span>
                        )}
                      </div>
                      {item.description && (
                        <p className="line-clamp-1 text-[11px] text-[#8B95A5]">
                          {item.description}
                        </p>
                      )}
                    </div>
                  </td>

                  {/* SKU / Barcode */}
                  <td className="px-5 py-3.5">
                    <div className="flex flex-col font-mono text-[11px]">
                      <span className="font-semibold text-[#0B0F19]">
                        {item.sku}
                      </span>
                      {item.barcode && (
                        <span className="text-[#8B95A5]">{item.barcode}</span>
                      )}
                    </div>
                  </td>

                  {/* Category */}
                  <td className="px-5 py-3.5 text-xs text-[#555D6E]">
                    {item.category_name || t("inventory.table.generalCategory")}
                  </td>

                  {/* Cost Price */}
                  <td className="px-5 py-3.5 text-xs font-mono text-[#555D6E]">
                    {formatIDR(item.cost_price || 0)}
                  </td>

                  {/* Sale / Unit Price */}
                  <td className="px-5 py-3.5 text-xs font-mono font-bold text-[#0B0F19]">
                    {formatIDR(item.unit_price)}
                  </td>

                  {/* Stock Quantity */}
                  <td className="px-5 py-3.5">
                    <div className="inline-flex items-center gap-1.5">
                      <span
                        className={`inline-flex items-center gap-1 rounded-lg px-2 py-0.5 text-xs font-mono font-medium border ${
                          isOutOfStock
                            ? "bg-rose-50 text-rose-700 border-rose-200/60"
                            : isLowStock
                            ? "bg-amber-50 text-amber-700 border-amber-200/60"
                            : "bg-emerald-50 text-emerald-700 border-emerald-200/60"
                        }`}
                      >
                        {isLowStock && <AlertCircle className="h-3 w-3" aria-hidden="true" />}
                        <span>{item.stock_quantity} {t("inventory.unit")}</span>
                      </span>
                    </div>
                  </td>

                  {/* Status */}
                  <td className="px-5 py-3.5">
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
                  <td className="px-5 py-3.5 text-right rtl:text-left">
                    <div className="flex items-center justify-end gap-1 rtl:justify-start">
                      {/* View Stock Card Button */}
                      <button
                        type="button"
                        onClick={() => onOpenStockCard(item)}
                        title={t("inventory.table.stockCardAction")}
                        aria-label={`${t("inventory.table.stockCardAction")} ${item.name}`}
                        className="p-1.5 rounded-lg text-[#555D6E] hover:text-[#0066CC] hover:bg-[#F5F5F7] transition-colors"
                      >
                        <BookOpen className="h-3.5 w-3.5" aria-hidden="true" />
                      </button>

                      {/* Reorder / Procure Draft Button (especially visible for low stock or zero stock) */}
                      {(isLowStock || isOutOfStock) && (
                        <button
                          type="button"
                          onClick={() => onOpenProcureDraft(item)}
                          title={t("inventory.table.procureAction")}
                          aria-label={`${t("inventory.table.procureAction")} ${item.name}`}
                          className="p-1.5 rounded-lg text-amber-700 hover:text-amber-800 hover:bg-amber-50 transition-colors"
                        >
                          <Truck className="h-3.5 w-3.5" aria-hidden="true" />
                        </button>
                      )}

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
                        aria-label={`${t("inventory.table.adjustAction")} ${item.name}`}
                        className="p-1.5 rounded-lg text-[#555D6E] hover:text-[#0066CC] hover:bg-[#F5F5F7] transition-colors disabled:cursor-not-allowed disabled:opacity-40"
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
                        aria-label={`${t("inventory.table.editAction")} ${item.name}`}
                        className="p-1.5 rounded-lg text-[#555D6E] hover:text-[#0066CC] hover:bg-[#F5F5F7] transition-colors disabled:cursor-not-allowed disabled:opacity-40"
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
                        aria-label={`${t("inventory.table.deleteAction")} ${item.name}`}
                        className="p-1.5 rounded-lg text-[#555D6E] hover:text-rose-600 hover:bg-rose-50 transition-colors disabled:cursor-not-allowed disabled:opacity-40"
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
      <div className="divide-y divide-black/[0.05] lg:hidden">
        {products.map((item) => {
          const threshold = item.reorder_threshold ?? 5;
          const isOutOfStock = item.stock_quantity <= 0;
          const isLowStock = !isOutOfStock && item.stock_quantity <= threshold;

          return (
            <article key={item.id} className="p-4 space-y-2.5 bg-white">
              <div className="flex items-start justify-between gap-2">
                <div>
                  <div className="flex items-center gap-1.5">
                    <h4 className="font-semibold text-sm text-[#0B0F19]">
                      {item.name}
                    </h4>
                    {item.is_halal_certified && (
                      <span className="inline-flex items-center gap-0.5 rounded-full bg-emerald-50 px-1.5 py-0.5 text-[10px] font-medium text-emerald-700 border border-emerald-200/60">
                        <ShieldCheck className="h-3 w-3" aria-hidden="true" />
                        <span>{t("inventory.table.halalBadge")}</span>
                      </span>
                    )}
                  </div>
                  <span className="font-mono text-[11px] text-[#8B95A5]">
                    SKU: {item.sku} {item.barcode ? `• ${item.barcode}` : ""}
                  </span>
                </div>

                <span
                  className={`inline-flex items-center gap-1 rounded-lg px-2 py-0.5 text-xs font-mono font-medium border ${
                    isOutOfStock
                      ? "bg-rose-50 text-rose-700 border-rose-200/60"
                      : isLowStock
                      ? "bg-amber-50 text-amber-700 border-amber-200/60"
                      : "bg-emerald-50 text-emerald-700 border-emerald-200/60"
                  }`}
                >
                  {isLowStock && <AlertCircle className="h-3 w-3" aria-hidden="true" />}
                  <span>{item.stock_quantity} {t("inventory.unit")}</span>
                </span>
              </div>

              <div className="flex items-center justify-between text-xs">
                <span className="text-[#555D6E]">
                  {item.category_name || t("inventory.table.generalCategory")}
                </span>
                <span className="font-mono font-bold text-sm text-[#0B0F19]">
                  {formatIDR(item.unit_price)}
                </span>
              </div>

              {/* Mobile Actions */}
              <div className="flex flex-wrap items-center justify-end gap-1 border-t border-black/[0.05] pt-2">
                <button
                  type="button"
                  onClick={() => onOpenStockCard(item)}
                  className="flex items-center gap-1 rounded-lg px-2.5 py-1 text-xs text-[#555D6E] hover:bg-[#F5F5F7] transition-colors"
                >
                  <BookOpen className="h-3.5 w-3.5" aria-hidden="true" />
                  <span>{t("inventory.table.stockCardAction")}</span>
                </button>

                {(isLowStock || isOutOfStock) && (
                  <button
                    type="button"
                    onClick={() => onOpenProcureDraft(item)}
                    className="flex items-center gap-1 rounded-lg px-2.5 py-1 text-xs text-amber-800 bg-amber-50 hover:bg-amber-100 transition-colors"
                  >
                    <Truck className="h-3.5 w-3.5" aria-hidden="true" />
                    <span>{t("inventory.table.procureAction")}</span>
                  </button>
                )}

                <button
                  type="button"
                  onClick={() => onAdjustStock(item)}
                  disabled={!canWrite}
                  className="flex items-center gap-1 rounded-lg px-2.5 py-1 text-xs text-[#555D6E] hover:bg-[#F5F5F7] transition-colors disabled:opacity-40"
                >
                  <Sliders className="h-3.5 w-3.5" aria-hidden="true" />
                  <span>{t("inventory.table.adjustAction")}</span>
                </button>
                <button
                  type="button"
                  onClick={() => onEditProduct(item)}
                  disabled={!canWrite}
                  className="flex items-center gap-1 rounded-lg px-2.5 py-1 text-xs text-[#555D6E] hover:bg-[#F5F5F7] transition-colors disabled:opacity-40"
                >
                  <Edit2 className="h-3.5 w-3.5" aria-hidden="true" />
                  <span>{t("inventory.table.editAction")}</span>
                </button>
                <button
                  type="button"
                  onClick={() => onDeleteProduct(item)}
                  disabled={!canWrite}
                  aria-label={`${t("inventory.table.deleteAction")} ${item.name}`}
                  className="flex items-center gap-1 rounded-lg p-1 text-[#555D6E] hover:text-rose-600 hover:bg-rose-50 transition-colors disabled:opacity-40"
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
