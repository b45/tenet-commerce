"use client";

import * as React from "react";
import { X, ShieldCheck } from "lucide-react";
import { useTranslation } from "@/lib/i18n";
import type {
  InventoryProduct,
  Category,
  CreateProductPayload,
  UpdateProductPayload,
} from "../types";

interface ProductModalProps {
  isOpen: boolean;
  onClose: () => void;
  product?: InventoryProduct | null;
  categories: Category[];
  onSubmitCreate: (payload: CreateProductPayload) => Promise<{ success: boolean; error?: string }>;
  onSubmitUpdate: (id: string, payload: UpdateProductPayload) => Promise<{ success: boolean; error?: string }>;
}

export function ProductModal({
  isOpen,
  onClose,
  product,
  categories,
  onSubmitCreate,
  onSubmitUpdate,
}: ProductModalProps) {
  const { t } = useTranslation();
  const dialogRef = React.useRef<HTMLDialogElement>(null);

  const isEdit = Boolean(product);

  // Form State
  const [sku, setSku] = React.useState("");
  const [barcode, setBarcode] = React.useState("");
  const [name, setName] = React.useState("");
  const [description, setDescription] = React.useState("");
  const [categoryId, setCategoryId] = React.useState("");
  const [unitPrice, setUnitPrice] = React.useState<number>(0);
  const [costPrice, setCostPrice] = React.useState<number>(0);
  const [initialStock, setInitialStock] = React.useState<number>(0);
  const [reorderThreshold, setReorderThreshold] = React.useState<number>(5);
  const [isHalal, setIsHalal] = React.useState(true);
  const [isActive, setIsActive] = React.useState(true);

  const [submitting, setSubmitting] = React.useState(false);
  const [errorMessage, setErrorMessage] = React.useState<string | null>(null);

  // Sync state when product changes or modal opens
  React.useEffect(() => {
    if (product) {
      setSku(product.sku);
      setBarcode(product.barcode || "");
      setName(product.name);
      setDescription(product.description || "");
      setCategoryId(product.category_id || "");
      setUnitPrice(product.unit_price);
      setCostPrice(product.cost_price || 0);
      setInitialStock(product.stock_quantity);
      setReorderThreshold(product.reorder_threshold ?? 5);
      setIsHalal(product.is_halal_certified);
      setIsActive(product.is_active);
    } else {
      setSku("");
      setBarcode("");
      setName("");
      setDescription("");
      setCategoryId("");
      setUnitPrice(0);
      setCostPrice(0);
      setInitialStock(0);
      setReorderThreshold(5);
      setIsHalal(true);
      setIsActive(true);
    }
    setErrorMessage(null);
  }, [product, isOpen]);

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

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorMessage(null);

    if (!name.trim()) {
      setErrorMessage(t("inventory.productModal.name") + " wajib diisi");
      return;
    }

    if (!isEdit && !sku.trim()) {
      setErrorMessage(t("inventory.productModal.sku") + " wajib diisi");
      return;
    }

    setSubmitting(true);
    try {
      if (isEdit && product) {
        const payload: UpdateProductPayload = {
          name: name.trim(),
          barcode: barcode.trim() || undefined,
          description: description.trim() || undefined,
          category_id: categoryId || undefined,
          unit_price: Number(unitPrice),
          cost_price: Number(costPrice),
          reorder_threshold: Number(reorderThreshold),
          compliance_tags: isHalal ? ["halal_certified"] : [],
          is_active: isActive,
        };

        const res = await onSubmitUpdate(product.id, payload);
        if (res.success) {
          onClose();
        } else {
          setErrorMessage(res.error || "Gagal memperbarui produk");
        }
      } else {
        const payload: CreateProductPayload = {
          name: name.trim(),
          sku: sku.trim().toUpperCase(),
          barcode: barcode.trim() || undefined,
          description: description.trim() || undefined,
          category_id: categoryId || undefined,
          unit_price: Number(unitPrice),
          cost_price: Number(costPrice),
          initial_stock: Number(initialStock),
          reorder_threshold: Number(reorderThreshold),
          compliance_tags: isHalal ? ["halal_certified"] : [],
          is_active: isActive,
        };

        const res = await onSubmitCreate(payload);
        if (res.success) {
          onClose();
        } else {
          setErrorMessage(res.error || "Gagal membuat produk");
        }
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
      className="m-auto w-full max-w-lg rounded-2xl border border-neutral-200 bg-white p-0 shadow-2xl backdrop:bg-black/40 backdrop:backdrop-blur-sm dark:border-neutral-800 dark:bg-neutral-900"
    >
      <div className="flex flex-col max-h-[90vh]">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-neutral-200 px-6 py-4 dark:border-neutral-800">
          <h3 className="text-base font-semibold text-neutral-900 dark:text-neutral-100">
            {isEdit
              ? t("inventory.productModal.editTitle")
              : t("inventory.productModal.createTitle")}
          </h3>
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg p-1 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-700 dark:hover:bg-neutral-800"
            aria-label="Tutup modal"
          >
            <X className="h-5 w-5" aria-hidden="true" />
          </button>
        </div>

        {/* Scrollable Form Body */}
        <form onSubmit={handleSubmit} className="overflow-y-auto px-6 py-4 space-y-4">
          {errorMessage && (
            <div className="rounded-xl border border-rose-500/20 bg-rose-500/10 p-3 text-xs text-rose-600 dark:text-rose-400">
              {errorMessage}
            </div>
          )}

          {/* SKU & Barcode */}
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div>
              <label className="block text-xs font-medium text-neutral-700 dark:text-neutral-300">
                {t("inventory.productModal.sku")} *
              </label>
              <input
                type="text"
                required
                disabled={isEdit}
                value={sku}
                onChange={(e) => setSku(e.target.value.toUpperCase())}
                placeholder={t("inventory.productModal.skuPlaceholder")}
                className="mt-1.5 h-10 w-full rounded-xl border border-neutral-200 bg-neutral-50/50 px-3 font-mono text-xs text-neutral-900 uppercase focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/20 disabled:bg-neutral-100 disabled:text-neutral-400 dark:border-neutral-800 dark:bg-neutral-800/50 dark:text-neutral-100 dark:disabled:bg-neutral-800"
              />
            </div>

            <div>
              <label className="block text-xs font-medium text-neutral-700 dark:text-neutral-300">
                {t("inventory.productModal.barcode")}
              </label>
              <input
                type="text"
                value={barcode}
                onChange={(e) => setBarcode(e.target.value)}
                placeholder={t("inventory.productModal.barcodePlaceholder")}
                className="mt-1.5 h-10 w-full rounded-xl border border-neutral-200 bg-white px-3 font-mono text-xs text-neutral-900 focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/20 dark:border-neutral-800 dark:bg-neutral-800/50 dark:text-neutral-100"
              />
            </div>
          </div>

          {/* Product Name */}
          <div>
            <label className="block text-xs font-medium text-neutral-700 dark:text-neutral-300">
              {t("inventory.productModal.name")} *
            </label>
            <input
              type="text"
              required
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={t("inventory.productModal.namePlaceholder")}
              className="mt-1.5 h-10 w-full rounded-xl border border-neutral-200 bg-white px-3 text-sm text-neutral-900 focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/20 dark:border-neutral-800 dark:bg-neutral-800/50 dark:text-neutral-100"
            />
          </div>

          {/* Category */}
          <div>
            <label className="block text-xs font-medium text-neutral-700 dark:text-neutral-300">
              {t("inventory.productModal.category")}
            </label>
            <select
              value={categoryId}
              onChange={(e) => setCategoryId(e.target.value)}
              className="mt-1.5 h-10 w-full rounded-xl border border-neutral-200 bg-white px-3 text-xs text-neutral-900 focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/20 dark:border-neutral-800 dark:bg-neutral-800/50 dark:text-neutral-100"
            >
              <option value="">{t("inventory.productModal.selectCategory")}</option>
              {categories.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>
          </div>

          {/* Pricing: Sale Price & Cost Price */}
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div>
              <label className="block text-xs font-medium text-neutral-700 dark:text-neutral-300">
                {t("inventory.productModal.unitPrice")} *
              </label>
              <input
                type="number"
                min="0"
                step="100"
                required
                value={unitPrice || ""}
                onChange={(e) => setUnitPrice(Number(e.target.value))}
                className="mt-1.5 h-10 w-full rounded-xl border border-neutral-200 bg-white px-3 font-mono text-sm font-semibold text-neutral-900 focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/20 dark:border-neutral-800 dark:bg-neutral-800/50 dark:text-neutral-100"
              />
            </div>

            <div>
              <label className="block text-xs font-medium text-neutral-700 dark:text-neutral-300">
                {t("inventory.productModal.costPrice")}
              </label>
              <input
                type="number"
                min="0"
                step="100"
                value={costPrice || ""}
                onChange={(e) => setCostPrice(Number(e.target.value))}
                className="mt-1.5 h-10 w-full rounded-xl border border-neutral-200 bg-white px-3 font-mono text-sm text-neutral-900 focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/20 dark:border-neutral-800 dark:bg-neutral-800/50 dark:text-neutral-100"
              />
            </div>
          </div>

          {/* Initial Stock (Only on Create) & Reorder Threshold */}
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            {!isEdit && (
              <div>
                <label className="block text-xs font-medium text-neutral-700 dark:text-neutral-300">
                  {t("inventory.productModal.initialStock")}
                </label>
                <input
                  type="number"
                  min="0"
                  value={initialStock || ""}
                  onChange={(e) => setInitialStock(Number(e.target.value))}
                  className="mt-1.5 h-10 w-full rounded-xl border border-neutral-200 bg-white px-3 font-mono text-sm text-neutral-900 focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/20 dark:border-neutral-800 dark:bg-neutral-800/50 dark:text-neutral-100"
                />
              </div>
            )}

            <div>
              <label className="block text-xs font-medium text-neutral-700 dark:text-neutral-300">
                {t("inventory.productModal.reorderThreshold")}
              </label>
              <input
                type="number"
                min="0"
                value={reorderThreshold || ""}
                onChange={(e) => setReorderThreshold(Number(e.target.value))}
                className="mt-1.5 h-10 w-full rounded-xl border border-neutral-200 bg-white px-3 font-mono text-sm text-neutral-900 focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/20 dark:border-neutral-800 dark:bg-neutral-800/50 dark:text-neutral-100"
              />
            </div>
          </div>

          {/* Description */}
          <div>
            <label className="block text-xs font-medium text-neutral-700 dark:text-neutral-300">
              {t("inventory.productModal.description")}
            </label>
            <textarea
              rows={2}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder={t("inventory.productModal.descriptionPlaceholder")}
              className="mt-1.5 w-full rounded-xl border border-neutral-200 bg-white p-3 text-xs text-neutral-900 focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/20 dark:border-neutral-800 dark:bg-neutral-800/50 dark:text-neutral-100"
            />
          </div>

          {/* Halal Certified & Active Checkboxes */}
          <div className="space-y-3 rounded-xl border border-neutral-200 bg-neutral-50/50 p-4 dark:border-neutral-800 dark:bg-neutral-800/30">
            <label className="flex items-center gap-3 cursor-pointer">
              <input
                type="checkbox"
                checked={isHalal}
                onChange={(e) => setIsHalal(e.target.checked)}
                className="h-4 w-4 rounded border-neutral-300 text-emerald-600 focus:ring-emerald-500 dark:border-neutral-700"
              />
              <div className="flex items-center gap-1.5 text-xs font-medium text-neutral-900 dark:text-neutral-100">
                <ShieldCheck className="h-4 w-4 text-emerald-600" aria-hidden="true" />
                <span>{t("inventory.productModal.halalCertified")}</span>
              </div>
            </label>

            <label className="flex items-center gap-3 cursor-pointer">
              <input
                type="checkbox"
                checked={isActive}
                onChange={(e) => setIsActive(e.target.checked)}
                className="h-4 w-4 rounded border-neutral-300 text-emerald-600 focus:ring-emerald-500 dark:border-neutral-700"
              />
              <span className="text-xs font-medium text-neutral-900 dark:text-neutral-100">
                {t("inventory.productModal.activeStatus")}
              </span>
            </label>
          </div>

          {/* Footer Buttons */}
          <div className="flex items-center justify-end gap-3 pt-4 border-t border-neutral-200 dark:border-neutral-800">
            <button
              type="button"
              onClick={onClose}
              className="h-10 rounded-xl px-4 text-xs font-medium text-neutral-600 transition hover:bg-neutral-100 dark:text-neutral-400 dark:hover:bg-neutral-800"
            >
              {t("inventory.productModal.cancel")}
            </button>
            <button
              type="submit"
              disabled={submitting}
              className="h-10 rounded-xl bg-emerald-600 px-5 text-xs font-medium text-white shadow-sm transition hover:bg-emerald-700 disabled:opacity-50 dark:bg-emerald-600 dark:hover:bg-emerald-500"
            >
              {submitting
                ? t("inventory.productModal.saving")
                : t("inventory.productModal.save")}
            </button>
          </div>
        </form>
      </div>
    </dialog>
  );
}
