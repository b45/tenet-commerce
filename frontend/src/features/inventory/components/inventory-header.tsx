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
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-neutral-900 dark:text-neutral-50">
            {t("inventory.title")}
          </h1>
          <p className="text-sm text-neutral-500 dark:text-neutral-400">
            {t("inventory.subtitle")}
          </p>
        </div>

        <div className="flex items-center gap-2.5">
          <button
            type="button"
            onClick={onRefresh}
            disabled={isLoading}
            className="flex h-10 w-10 items-center justify-center rounded-xl border border-neutral-200 bg-white/80 text-neutral-600 shadow-sm transition hover:bg-neutral-50 hover:text-neutral-900 disabled:opacity-50 dark:border-neutral-800 dark:bg-neutral-900/80 dark:text-neutral-300 dark:hover:bg-neutral-800"
            title="Refresh"
            aria-label="Refresh data"
          >
            <RotateCw
              className={`h-4 w-4 ${isLoading ? "animate-spin text-emerald-600" : ""}`}
              aria-hidden="true"
            />
          </button>

          <button
            type="button"
            onClick={onAddProduct}
            disabled={!canWrite}
            title={!canWrite ? t("inventory.permissions.readOnlyTooltip") : undefined}
            className="flex h-10 items-center gap-2 rounded-xl bg-emerald-600 px-4 text-sm font-medium text-white shadow-sm transition hover:bg-emerald-700 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-emerald-600 dark:hover:bg-emerald-500"
          >
            <Plus className="h-4 w-4" aria-hidden="true" />
            <span>{t("inventory.addProduct")}</span>
          </button>
        </div>
      </div>

      {/* Filter & Search Bar */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="relative flex-1 max-w-md">
          <Search
            className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-neutral-400 rtl:left-auto rtl:right-3"
            aria-hidden="true"
          />
          <input
            type="search"
            value={search}
            onChange={(e) => onSearchChange(e.target.value)}
            placeholder={t("inventory.searchPlaceholder")}
            className="h-10 w-full rounded-xl border border-neutral-200 bg-white/80 pl-9 pr-4 text-sm text-neutral-900 placeholder:text-neutral-400 focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/20 rtl:pl-4 rtl:pr-9 dark:border-neutral-800 dark:bg-neutral-900/80 dark:text-neutral-100 dark:placeholder:text-neutral-500"
          />
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {/* Category Dropdown */}
          <div className="relative">
            <select
              value={selectedCategory}
              onChange={(e) => onCategoryChange(e.target.value)}
              className="h-10 appearance-none rounded-xl border border-neutral-200 bg-white/80 px-3 pr-8 text-xs font-medium text-neutral-700 focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/20 rtl:pl-8 rtl:pr-3 dark:border-neutral-800 dark:bg-neutral-900/80 dark:text-neutral-300"
            >
              <option value="">{t("inventory.allCategories")}</option>
              {categories.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>
            <Filter
              className="pointer-events-none absolute right-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-neutral-400 rtl:left-2.5 rtl:right-auto"
              aria-hidden="true"
            />
          </div>

          {/* Status Segmented Buttons */}
          <div className="flex rounded-xl border border-neutral-200 bg-neutral-100/80 p-1 text-xs font-medium dark:border-neutral-800 dark:bg-neutral-900/80">
            <button
              type="button"
              onClick={() => onStockStatusChange("all")}
              className={`rounded-lg px-3 py-1.5 transition ${
                stockStatus === "all"
                  ? "bg-white text-neutral-900 shadow-sm dark:bg-neutral-800 dark:text-neutral-50"
                  : "text-neutral-600 hover:text-neutral-900 dark:text-neutral-400 dark:hover:text-neutral-200"
              }`}
            >
              {t("inventory.filterAll")}
            </button>
            <button
              type="button"
              onClick={() => onStockStatusChange("low_stock")}
              className={`rounded-lg px-3 py-1.5 transition ${
                stockStatus === "low_stock"
                  ? "bg-amber-500 text-white shadow-sm"
                  : "text-neutral-600 hover:text-neutral-900 dark:text-neutral-400 dark:hover:text-neutral-200"
              }`}
            >
              {t("inventory.filterLowStock")}
            </button>
            <button
              type="button"
              onClick={() => onStockStatusChange("out_of_stock")}
              className={`rounded-lg px-3 py-1.5 transition ${
                stockStatus === "out_of_stock"
                  ? "bg-rose-600 text-white shadow-sm"
                  : "text-neutral-600 hover:text-neutral-900 dark:text-neutral-400 dark:hover:text-neutral-200"
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
