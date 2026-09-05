"use client";

import * as React from "react";
import { AlertTriangle, X } from "lucide-react";
import { useTranslation } from "@/lib/i18n";
import type { InventoryProduct } from "../types";

interface ProductDeleteDialogProps {
  isOpen: boolean;
  onClose: () => void;
  product: InventoryProduct | null;
  onConfirmDelete: (id: string) => Promise<{ success: boolean; error?: string }>;
}

export function ProductDeleteDialog({
  isOpen,
  onClose,
  product,
  onConfirmDelete,
}: ProductDeleteDialogProps) {
  const { t } = useTranslation();
  const dialogRef = React.useRef<HTMLDialogElement>(null);
  const [submitting, setSubmitting] = React.useState(false);
  const [errorMessage, setErrorMessage] = React.useState<string | null>(null);

  React.useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;

    if (isOpen) {
      setErrorMessage(null);
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

  const handleDelete = async () => {
    setSubmitting(true);
    setErrorMessage(null);
    try {
      const res = await onConfirmDelete(product.id);
      if (res.success) {
        onClose();
      } else {
        setErrorMessage(res.error || "Gagal menonaktifkan produk");
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
      className="m-auto w-full max-w-sm rounded-2xl border border-neutral-200 bg-white p-0 shadow-2xl backdrop:bg-black/40 backdrop:backdrop-blur-sm dark:border-neutral-800 dark:bg-neutral-900"
    >
      <div className="p-6">
        <div className="flex items-start justify-between gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-rose-500/10 text-rose-600 dark:bg-rose-500/20 dark:text-rose-400">
            <AlertTriangle className="h-5 w-5" aria-hidden="true" />
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg p-1 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-700 dark:hover:bg-neutral-800"
            aria-label="Tutup"
          >
            <X className="h-4 w-4" aria-hidden="true" />
          </button>
        </div>

        <div className="mt-4">
          <h3 className="text-base font-semibold text-neutral-900 dark:text-neutral-100">
            {t("inventory.deleteDialog.title")}
          </h3>
          <p className="mt-2 text-xs text-neutral-500 leading-relaxed dark:text-neutral-400">
            {t("inventory.deleteDialog.message", { name: product.name })}
          </p>
        </div>

        {errorMessage && (
          <div className="mt-3 rounded-xl border border-rose-500/20 bg-rose-500/10 p-2.5 text-xs text-rose-600 dark:text-rose-400">
            {errorMessage}
          </div>
        )}

        <div className="mt-6 flex items-center justify-end gap-3">
          <button
            type="button"
            onClick={onClose}
            className="h-9 rounded-xl px-4 text-xs font-medium text-neutral-600 transition hover:bg-neutral-100 dark:text-neutral-400 dark:hover:bg-neutral-800"
          >
            {t("inventory.deleteDialog.cancel")}
          </button>
          <button
            type="button"
            onClick={handleDelete}
            disabled={submitting}
            className="h-9 rounded-xl bg-rose-600 px-4 text-xs font-medium text-white shadow-sm transition hover:bg-rose-700 disabled:opacity-50 dark:bg-rose-600 dark:hover:bg-rose-500"
          >
            {submitting
              ? t("inventory.deleteDialog.deleting")
              : t("inventory.deleteDialog.confirm")}
          </button>
        </div>
      </div>
    </dialog>
  );
}
