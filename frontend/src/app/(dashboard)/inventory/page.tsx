"use client";

import * as React from "react";
import { useAuth } from "@/features/auth/hooks/use-auth";
import { useInventory } from "@/features/inventory/hooks/use-inventory";
import { LowStockBanner } from "@/features/inventory/components/low-stock-banner";
import { InventoryHeader } from "@/features/inventory/components/inventory-header";
import { InventoryTable } from "@/features/inventory/components/inventory-table";
import { ProductModal } from "@/features/inventory/components/product-modal";
import { StockAdjustModal } from "@/features/inventory/components/stock-adjust-modal";
import { ProductDeleteDialog } from "@/features/inventory/components/product-delete-dialog";
import type { InventoryProduct } from "@/features/inventory/types";
import { useTranslation } from "@/lib/i18n";

export default function InventoryPage() {
  const { t } = useTranslation();
  const { hasPermission } = useAuth();
  const canWrite = hasPermission("inventory:write");

  // Inventory state and mutations
  const {
    filteredProducts,
    categories,
    lowStockItems,
    loading,
    error,
    filters,
    setSearch,
    setCategory,
    setStockStatus,
    refresh,
    adjustStock,
    createProduct,
    updateProduct,
    deleteProduct,
  } = useInventory();

  // Modals state
  const [productModalOpen, setProductModalOpen] = React.useState(false);
  const [editingProduct, setEditingProduct] = React.useState<InventoryProduct | null>(null);

  const [adjustModalOpen, setAdjustModalOpen] = React.useState(false);
  const [adjustingProduct, setAdjustingProduct] = React.useState<InventoryProduct | null>(null);

  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false);
  const [deletingProduct, setDeletingProduct] = React.useState<InventoryProduct | null>(null);

  // Success notification message banner
  const [feedbackMessage, setFeedbackMessage] = React.useState<string | null>(null);

  const showFeedback = (msg: string) => {
    setFeedbackMessage(msg);
    setTimeout(() => {
      setFeedbackMessage(null);
    }, 4000);
  };

  // Handlers
  const handleOpenAddProduct = () => {
    setEditingProduct(null);
    setProductModalOpen(true);
  };

  const handleOpenEditProduct = (prod: InventoryProduct) => {
    setEditingProduct(prod);
    setProductModalOpen(true);
  };

  const handleOpenAdjustStock = (prod: InventoryProduct) => {
    setAdjustingProduct(prod);
    setAdjustModalOpen(true);
  };

  const handleOpenDeleteProduct = (prod: InventoryProduct) => {
    setDeletingProduct(prod);
    setDeleteDialogOpen(true);
  };

  return (
    <div className="mx-auto max-w-7xl space-y-6">
      {/* Temporary Notification Banner */}
      {feedbackMessage && (
        <div className="flex items-center justify-between rounded-xl border border-emerald-500/20 bg-emerald-500/10 px-4 py-3 text-xs font-medium text-emerald-800 backdrop-blur-sm dark:text-emerald-200 animate-in fade-in duration-200">
          <span>{feedbackMessage}</span>
          <button
            type="button"
            onClick={() => setFeedbackMessage(null)}
            className="text-emerald-600 hover:text-emerald-800 dark:text-emerald-400"
          >
            ✕
          </button>
        </div>
      )}

      {/* Global Error Banner */}
      {error && (
        <div className="rounded-xl border border-rose-500/20 bg-rose-500/10 p-4 text-xs font-medium text-rose-700 dark:text-rose-300">
          {error}
        </div>
      )}

      {/* Low Stock Alert Banner */}
      <LowStockBanner
        count={lowStockItems.length}
        onFilterLowStock={() =>
          setStockStatus(filters.stock_status === "low_stock" ? "all" : "low_stock")
        }
        isFilterActive={filters.stock_status === "low_stock"}
      />

      {/* Header & Controls */}
      <InventoryHeader
        search={filters.search}
        onSearchChange={setSearch}
        selectedCategory={filters.category_id}
        onCategoryChange={setCategory}
        categories={categories}
        stockStatus={filters.stock_status}
        onStockStatusChange={setStockStatus}
        onAddProduct={handleOpenAddProduct}
        onRefresh={refresh}
        isLoading={loading}
        canWrite={canWrite}
      />

      {/* Main Table / Mobile Cards */}
      <InventoryTable
        products={filteredProducts}
        onAdjustStock={handleOpenAdjustStock}
        onEditProduct={handleOpenEditProduct}
        onDeleteProduct={handleOpenDeleteProduct}
        canWrite={canWrite}
        isLoading={loading}
      />

      {/* Product Create / Edit Modal */}
      <ProductModal
        isOpen={productModalOpen}
        onClose={() => setProductModalOpen(false)}
        product={editingProduct}
        categories={categories}
        onSubmitCreate={async (payload) => {
          const res = await createProduct(payload);
          if (res.success) {
            showFeedback(`Produk ${payload.name} berhasil ditambahkan.`);
          }
          return res;
        }}
        onSubmitUpdate={async (id, payload) => {
          const res = await updateProduct(id, payload);
          if (res.success) {
            showFeedback(`Produk ${payload.name} berhasil diperbarui.`);
          }
          return res;
        }}
      />

      {/* Stock Adjustment / Opname Modal */}
      <StockAdjustModal
        isOpen={adjustModalOpen}
        onClose={() => setAdjustModalOpen(false)}
        product={adjustingProduct}
        onSubmitAdjust={async (payload) => {
          const res = await adjustStock(payload);
          if (res.success && adjustingProduct) {
            showFeedback(
              t("inventory.adjustModal.successMessage", {
                name: adjustingProduct.name,
                quantity: res.data?.new_quantity ?? payload.quantity,
              })
            );
          }
          return res;
        }}
      />

      {/* Product Deactivate Dialog */}
      <ProductDeleteDialog
        isOpen={deleteDialogOpen}
        onClose={() => setDeleteDialogOpen(false)}
        product={deletingProduct}
        onConfirmDelete={async (id) => {
          const res = await deleteProduct(id);
          if (res.success && deletingProduct) {
            showFeedback(
              t("inventory.deleteDialog.successMessage", {
                name: deletingProduct.name,
              })
            );
          }
          return res;
        }}
      />
    </div>
  );
}
