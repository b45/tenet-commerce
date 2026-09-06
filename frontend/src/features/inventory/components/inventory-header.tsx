"use client";

import * as React from "react";
import { Search, Plus, Filter, RotateCw } from "lucide-react";
import { useTranslation } from "@/lib/i18n";
import type { Category, StockStatusFilter } from "../types";

interface InventoryHeaderProps {
  search: string;
  onSearchChange: (val: string) => void;
  selectedCategory: string;
  onCategoryChange: (catId: string) => void;
  categories: Category[];
  stockStatus: StockStatusFilter;
  onStockStatusChange: (status: StockStatusFilter) => void;
  onAddProduct: () => void;
  onRefresh: () => void;
  isLoading: boolean;
  canWrite: boolean;
}

export function InventoryHeader({
  search,
  onSearchChange,
  selectedCategory,
  onCategoryChange,
  categories,
  stockStatus,
  onStockStatusChange,
  onAddProduct,
  onRefresh,
  isLoading,
  canWrite,
}: InventoryHeaderProps) {
  const { t } = useTranslation();

  return (
    <header className="mb-6 space-y-4">
      {/* Title & Top Action Bar */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-xl sm:text-2xl font-bold tracking-tight text-[var(--color-text-primary)]">
            {t("inventory.title")}
          </h1>
          <p className="text-xs sm:text-sm text-[var(--color-text-secondary)]">
            {t("inventory.subtitle")}
          </p>
        </div>

        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={onRefresh}
            disabled={isLoading}
            className="flex h-9 w-9 items-center justify-center rounded-xl border border-black/[0.08] bg-white text-[#555D6E] shadow-xs transition hover:bg-[#F5F5F7] hover:text-[#0B0F19] disabled:opacity-50"
            title={t("inventory.refresh")}
            aria-label={t("inventory.refresh")}
          >
            <RotateCw
              className={`h-4 w-4 ${isLoading ? "animate-spin text-[#0066CC]" : ""}`}
              aria-hidden="true"
            />
          </button>

          <button
            type="button"
            onClick={onAddProduct}
            disabled={!canWrite}
            title={!canWrite ? t("inventory.permissions.readOnlyTooltip") : undefined}
            className="flex h-9 items-center gap-1.5 rounded-xl bg-[#0066CC] px-3.5 text-xs font-semibold text-white shadow-xs transition hover:bg-[#0055B3] disabled:cursor-not-allowed disabled:opacity-50"
          >
            <Plus className="h-4 w-4" aria-hidden="true" />
            <span>{t("inventory.addProduct")}</span>
          </button>
        </div>
      </div>

      {/* Filter & Search Bar */}
      <div className="flex flex-col gap-2.5 sm:flex-row sm:items-center sm:justify-between">
        <div className="relative flex-1 max-w-md">
          <Search
            className="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-[#8B95A5] rtl:left-auto rtl:right-3"
            aria-hidden="true"
          />
          <input
            type="search"
            value={search}
            onChange={(e) => onSearchChange(e.target.value)}
            placeholder={t("inventory.searchPlaceholder")}
            className="h-9 w-full rounded-xl border border-black/[0.08] bg-white pl-9 pr-4 text-xs text-[#0B0F19] placeholder:text-[#8B95A5] focus:border-[#0066CC] focus:outline-none focus:ring-2 focus:ring-[#0066CC]/20 rtl:pl-4 rtl:pr-9 shadow-xs"
          />
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {/* Category Dropdown */}
          <div className="relative">
            <select
              value={selectedCategory}
              onChange={(e) => onCategoryChange(e.target.value)}
              className="h-9 appearance-none rounded-xl border border-black/[0.08] bg-white px-3 pr-8 text-xs font-medium text-[#0B0F19] focus:border-[#0066CC] focus:outline-none focus:ring-2 focus:ring-[#0066CC]/20 rtl:pl-8 rtl:pr-3 shadow-xs"
            >
              <option value="">{t("inventory.allCategories")}</option>
              {categories.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>
            <Filter
              className="pointer-events-none absolute right-2.5 top-1/2 h-3 w-3 -translate-y-1/2 text-[#8B95A5] rtl:left-2.5 rtl:right-auto"
              aria-hidden="true"
            />
          </div>

          {/* Status Segmented Buttons */}
          <div className="inline-flex h-9 items-center rounded-xl border border-black/[0.06] bg-[#F5F5F7] p-0.5 text-xs font-medium">
            <button
              type="button"
              onClick={() => onStockStatusChange("all")}
              className={`rounded-lg px-2.5 py-1 transition ${
                stockStatus === "all"
                  ? "bg-white text-[#0B0F19] shadow-xs font-semibold"
                  : "text-[#555D6E] hover:text-[#0B0F19]"
              }`}
            >
              {t("inventory.filterAll")}
            </button>
            <button
              type="button"
              onClick={() => onStockStatusChange("low_stock")}
              className={`rounded-lg px-2.5 py-1 transition ${
                stockStatus === "low_stock"
                  ? "bg-amber-500 text-white shadow-xs font-semibold"
                  : "text-[#555D6E] hover:text-[#0B0F19]"
              }`}
            >
              {t("inventory.filterLowStock")}
            </button>
            <button
              type="button"
              onClick={() => onStockStatusChange("out_of_stock")}
              className={`rounded-lg px-2.5 py-1 transition ${
                stockStatus === "out_of_stock"
                  ? "bg-rose-600 text-white shadow-xs font-semibold"
                  : "text-[#555D6E] hover:text-[#0B0F19]"
              }`}
            >
              {t("inventory.filterOutOfStock")}
            </button>
          </div>
        </div>
      </div>
    </header>
  );
}
