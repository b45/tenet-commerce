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
          <h1 className="text-2xl font-bold tracking-tight text-[var(--color-text-primary)]">
            {t("inventory.title")}
          </h1>
          <p className="text-sm text-[var(--color-text-secondary)]">
            {t("inventory.subtitle")}
          </p>
        </div>

        <div className="flex items-center gap-2.5">
          <button
            type="button"
            onClick={onRefresh}
            disabled={isLoading}
            className="flex h-10 w-10 items-center justify-center rounded-xl border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] text-[var(--color-text-secondary)] shadow-sm transition hover:bg-[var(--color-surface-muted)] hover:text-[var(--color-text-primary)] disabled:opacity-50"
            title="Refresh"
            aria-label="Refresh data"
          >
            <RotateCw
              className={`h-4 w-4 ${isLoading ? "animate-spin text-[var(--color-action-primary)]" : ""}`}
              aria-hidden="true"
            />
          </button>

          <button
            type="button"
            onClick={onAddProduct}
            disabled={!canWrite}
            title={!canWrite ? t("inventory.permissions.readOnlyTooltip") : undefined}
            className="flex h-10 items-center gap-2 rounded-xl bg-[var(--color-action-primary)] px-4 text-sm font-medium text-white shadow-sm transition hover:bg-[var(--color-action-primary-hover)] disabled:cursor-not-allowed disabled:opacity-50"
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
            className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[var(--color-text-muted)] rtl:left-auto rtl:right-3"
            aria-hidden="true"
          />
          <input
            type="search"
            value={search}
            onChange={(e) => onSearchChange(e.target.value)}
            placeholder={t("inventory.searchPlaceholder")}
            className="h-10 w-full rounded-xl border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] pl-9 pr-4 text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] focus:border-[var(--color-action-primary)] focus:outline-none focus:ring-2 focus:ring-[var(--color-action-focus-ring)] rtl:pl-4 rtl:pr-9 shadow-sm"
          />
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {/* Category Dropdown */}
          <div className="relative">
            <select
              value={selectedCategory}
              onChange={(e) => onCategoryChange(e.target.value)}
              className="h-10 appearance-none rounded-xl border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] px-3 pr-8 text-xs font-medium text-[var(--color-text-primary)] focus:border-[var(--color-action-primary)] focus:outline-none focus:ring-2 focus:ring-[var(--color-action-focus-ring)] rtl:pl-8 rtl:pr-3 shadow-sm"
            >
              <option value="">{t("inventory.allCategories")}</option>
              {categories.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>
            <Filter
              className="pointer-events-none absolute right-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-[var(--color-text-muted)] rtl:left-2.5 rtl:right-auto"
              aria-hidden="true"
            />
          </div>

          {/* Status Segmented Buttons */}
          <div className="flex rounded-xl border border-[var(--color-border-hairline)] bg-[var(--color-surface-muted)] p-1 text-xs font-medium">
            <button
              type="button"
              onClick={() => onStockStatusChange("all")}
              className={`rounded-lg px-3 py-1.5 transition ${
                stockStatus === "all"
                  ? "bg-[var(--color-surface-base)] text-[var(--color-text-primary)] shadow-sm font-semibold"
                  : "text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
              }`}
            >
              {t("inventory.filterAll")}
            </button>
            <button
              type="button"
              onClick={() => onStockStatusChange("low_stock")}
              className={`rounded-lg px-3 py-1.5 transition ${
                stockStatus === "low_stock"
                  ? "bg-amber-500 text-white shadow-sm font-semibold"
                  : "text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
              }`}
            >
              {t("inventory.filterLowStock")}
            </button>
            <button
              type="button"
              onClick={() => onStockStatusChange("out_of_stock")}
              className={`rounded-lg px-3 py-1.5 transition ${
                stockStatus === "out_of_stock"
                  ? "bg-rose-600 text-white shadow-sm font-semibold"
                  : "text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
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
