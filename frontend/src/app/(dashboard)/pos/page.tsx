"use client";

import * as React from "react";
import { Badge } from "@/components/ui/badge";
import { SegmentedControl } from "@/components/ui/segmented-control";
import { useCatalog } from "@/features/pos/hooks/use-catalog";
import { useCart } from "@/features/pos/hooks/use-cart";
import { useCheckout } from "@/features/pos/hooks/use-checkout";
import { CatalogGrid } from "@/features/pos/components/catalog-grid";
import { CartPanel } from "@/features/pos/components/cart-panel";
import { TenderModal } from "@/features/pos/components/tender-modal";
import { ReceiptModal } from "@/features/pos/components/receipt-modal";
import { ThermalReceipt } from "@/features/pos/components/thermal-receipt";
import { OrderHistory } from "@/features/pos/components/order-history";
import type { POSViewMode } from "@/features/pos/types";

export default function POSPage() {
  const [viewMode, setViewMode] = React.useState<POSViewMode>("register");

  // Domain Hooks
  const catalog = useCatalog();
  const cart = useCart();
  const checkout = useCheckout();

  // Mitigation G-01: Re-fetch catalog when opening tender review to ensure latest prices/stock
  const handleOpenTender = () => {
    catalog.refetch();
    checkout.startReview(cart.totals.total);
  };

  const handleCheckoutSubmit = () => {
    checkout.submitCheckout(cart.items, cart.totals.total);
  };

  const handleNewTransaction = () => {
    cart.clearCart();
    checkout.resetCheckout();
  };

  return (
    <div className="space-y-4">
      {/* Page Header Bar */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pb-1 border-b border-[var(--color-border-hairline)]">
        <div>
          <h1 className="text-xl sm:text-2xl font-bold tracking-tight text-[var(--color-text-primary)]">
            Terminal Kasir POS
          </h1>
          <p className="text-xs text-[var(--color-text-secondary)]">
            Pencarian produk, barcode scanner, manajemen keranjang, dan cetak struk thermal.
          </p>
        </div>

        <div className="flex items-center gap-3">
          <SegmentedControl
            options={[
              { value: "register", label: "Register Kasir" },
              { value: "history", label: "Riwayat Transaksi" },
            ]}
            value={viewMode}
            onChange={(val) => setViewMode(val as POSViewMode)}
            size="sm"
          />

          <Badge variant="success" className="text-[11px] px-2.5 py-1 hidden sm:inline-flex">
            Online
          </Badge>
        </div>
      </div>

      {/* Main Content Workspace */}
      {viewMode === "register" ? (
        <div className="flex flex-col lg:flex-row gap-5 h-[calc(100vh-190px)] min-h-[560px]">
          {/* Left Panel: Catalog & Search Grid */}
          <div className="flex-1 min-w-0 h-full overflow-hidden">
            <CatalogGrid
              products={catalog.filteredProducts}
              categories={catalog.categories}
              selectedCategory={catalog.selectedCategory}
              onSelectCategory={catalog.setSelectedCategory}
              searchQuery={catalog.searchQuery}
              onSearchChange={catalog.setSearchQuery}
              onAddToCart={cart.addItem}
              isLoading={catalog.isLoading}
              error={catalog.error}
              onRetry={catalog.refetch}
            />
          </div>

          {/* Right Panel: Cart & Checkout Summary (Fixed Width on Desktop) */}
          <div className="w-full lg:w-[380px] xl:w-[420px] shrink-0 h-full">
            <CartPanel
              items={cart.items}
              totals={cart.totals}
              onUpdateQuantity={cart.updateQuantity}
              onDecrementItem={cart.decrementItem}
              onRemoveItem={cart.removeItem}
              onClearCart={cart.clearCart}
              onOpenTender={handleOpenTender}
            />
          </div>
        </div>
      ) : (
        <OrderHistory />
      )}

      {/* Tender Modal Dialog */}
      <TenderModal
        isOpen={checkout.step === "review" || checkout.step === "submitting" || checkout.step === "rejected" || checkout.step === "unknown_error"}
        onClose={checkout.closeReview}
        totalAmount={cart.totals.total}
        cashTendered={checkout.cashTendered}
        onCashTenderedChange={checkout.setCashTendered}
        onSubmit={handleCheckoutSubmit}
        isSubmitting={checkout.isSubmitting}
        errorMessage={checkout.errorMessage}
        step={checkout.step}
      />

      {/* Receipt Modal Dialog */}
      <ReceiptModal
        isOpen={checkout.step === "completed" && checkout.receipt !== null}
        onClose={checkout.resetCheckout}
        receipt={checkout.receipt}
        onNewTransaction={handleNewTransaction}
      />

      {/* Hidden 80mm Thermal Receipt Print Structure */}
      {checkout.receipt && (
        <ThermalReceipt receipt={checkout.receipt} />
      )}
    </div>
  );
}
