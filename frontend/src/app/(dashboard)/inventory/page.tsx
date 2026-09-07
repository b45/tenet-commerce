"use client";

import * as React from "react";
import { useAuth } from "@/features/auth/hooks/use-auth";
import { useInventory } from "@/features/inventory/hooks/use-inventory";
import { StockOverviewBanner } from "@/features/inventory/components/stock-overview-banner";
import { LowStockBanner } from "@/features/inventory/components/low-stock-banner";
import { InventoryHeader } from "@/features/inventory/components/inventory-header";
import { InventoryTable } from "@/features/inventory/components/inventory-table";
import { ProductModal } from "@/features/inventory/components/product-modal";
import { StockAdjustModal } from "@/features/inventory/components/stock-adjust-modal";
import { StockCardModal } from "@/features/inventory/components/stock-card-modal";
import { ProcurementDraftModal } from "@/features/inventory/components/procurement-draft-modal";
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
    stockOverview,
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
    getStockCard,
  } = useInventory();

  // Modals state
  const [productModalOpen, setProductModalOpen] = React.useState(false);
  const [editingProduct, setEditingProduct] = React.useState<InventoryProduct | null>(null);

  const [adjustModalOpen, setAdjustModalOpen] = React.useState(false);
  const [adjustingProduct, setAdjustingProduct] = React.useState<InventoryProduct | null>(null);

  const [stockCardModalOpen, setStockCardModalOpen] = React.useState(false);
  const [stockCardProduct, setStockCardProduct] = React.useState<InventoryProduct | null>(null);

  const [procureModalOpen, setProcureModalOpen] = React.useState(false);
  const [procureProduct, setProcureProduct] = React.useState<InventoryProduct | null>(null);

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

  const handleOpenStockCard = (prod: InventoryProduct) => {
    setStockCardProduct(prod);
    setStockCardModalOpen(true);
  };

  const handleOpenProcureDraft = (prod: InventoryProduct) => {
    setProcureProduct(prod);
    setProcureModalOpen(true);
  };

  const handleOpenDeleteProduct = (prod: InventoryProduct) => {
    setDeletingProduct(prod);
    setDeleteDialogOpen(true);
  };

  return (
    <div className="mx-auto max-w-7xl space-y-6">
      {/* Temporary Notification Banner */}
      {feedbackMessage && (
        <div className="flex items-center justify-between rounded-xl border border-emerald-500/20 bg-emerald-50 px-4 py-3 text-xs font-medium text-emerald-800 shadow-sm animate-in fade-in duration-200">
          <span>{feedbackMessage}</span>
          <button
            type="button"
            onClick={() => setFeedbackMessage(null)}
            aria-label={t("inventory.close")}
            className="text-emerald-600 hover:text-emerald-800"
          >
            ✕
          </button>
        </div>
      )}

      {/* Global Error Banner */}
      {error && (
        <div className="rounded-xl border border-rose-500/20 bg-rose-50 p-4 text-xs font-medium text-rose-700 shadow-sm">
          {error}
        </div>
      )}

      {/* Warehouse Inventory Overview Card */}
      <StockOverviewBanner overview={stockOverview} isLoading={loading} />

      {/* Low Stock Alert Banner */}
      <LowStockBanner
        count={lowStockItems.length}
        onFilterLowStock={() =>
          setStockStatus(filters.stock_status === "low_stock" ? "all" : "low_stock")
        }
        isFilterActive={filters.stock_status === "low_stock"}
        onOpenProcureAction={
          lowStockItems.length > 0 ? () => handleOpenProcureDraft(lowStockItems[0]) : undefined
        }
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
        onOpenStockCard={handleOpenStockCard}
        onOpenProcureDraft={handleOpenProcureDraft}
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
            showFeedback(t("inventory.productModal.createSuccess", { name: payload.name }));
          }
          return res;
        }}
        onSubmitUpdate={async (id, payload) => {
          const res = await updateProduct(id, payload);
          if (res.success) {
            showFeedback(t("inventory.productModal.updateSuccess", { name: payload.name }));
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

      {/* Stock Card Ledger Modal */}
      <StockCardModal
        isOpen={stockCardModalOpen}
        onClose={() => setStockCardModalOpen(false)}
        product={stockCardProduct}
        onFetchStockCard={getStockCard}
      />

      {/* Low-Stock Procurement Reorder Draft Modal */}
      <ProcurementDraftModal
        isOpen={procureModalOpen}
        onClose={() => setProcureModalOpen(false)}
        product={procureProduct}
        onConfirmDraft={(prod) => {
          showFeedback(
            t("inventory.procurementDraft.successMessage", { product: prod.name })
          );
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
