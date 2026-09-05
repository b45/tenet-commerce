"use client";

import * as React from "react";
import { ArrowLeft, ArrowRight } from "lucide-react";
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
import { formatIDR } from "@/lib/money";
import { cn } from "@/lib/utils";
import { useTranslation } from "@/lib/i18n";

export default function POSPage() {
  const { t } = useTranslation();
  const [viewMode, setViewMode] = React.useState<POSViewMode>("register");
  const [mobileTab, setMobileTab] = React.useState<"catalog" | "cart">("catalog");

  // Domain Hooks
  const catalog = useCatalog();
  const cart = useCart();
  const checkout = useCheckout();

  // Advisory refresh only: cart reconciliation / server quote remains G-01.
  const handleOpenTender = () => {
    catalog.refetch();
    checkout.startReview(cart.totals.total);
  };

  const handleCheckoutSubmit = () => {
    checkout.submitCheckout(cart.items, cart.totals.total);
  };

  const handleNewTransaction = () => {
    checkout.finishCompleted(() => {
      cart.clearCart();
      setMobileTab("catalog");
    });
  };

  return (
    <div className="space-y-4">
      <div inert={checkout.step !== "idle"} className="space-y-4">
      {/* Page Header Bar */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pb-1 border-b border-[var(--color-border-hairline)]">
        <div>
          <h1 className="text-xl sm:text-2xl font-bold tracking-tight text-[var(--color-text-primary)]">
            {t("pos.title")}
          </h1>
          <p className="text-xs text-[var(--color-text-secondary)]">
            {t("pos.subtitle")}
          </p>
        </div>

        <div className="flex items-center gap-3">
          <SegmentedControl
            options={[
              { value: "register", label: t("pos.tabs.register") },
              { value: "history", label: t("pos.tabs.history") },
            ]}
            value={viewMode}
            onChange={(val) => setViewMode(val as POSViewMode)}
            size="sm"
          />

          <Badge variant="success" className="text-[11px] px-2.5 py-1 hidden sm:inline-flex">
            {t("common.status.online")}
          </Badge>
        </div>
      </div>

      {/* Main Content Workspace */}
      {viewMode === "register" ? (
        <div className="relative flex flex-col lg:flex-row gap-5 lg:h-[calc(100vh-190px)] lg:min-h-[560px]">
          {/* Mobile sub-tab toggle (Catalog vs Cart) */}
          <div className="flex lg:hidden items-center p-1 bg-[var(--color-surface-muted)] rounded-[12px] border border-[var(--color-border-hairline)] w-full">
            <button
              type="button"
              onClick={() => setMobileTab("catalog")}
              className={cn(
                "flex-1 py-1.5 px-3 text-xs font-semibold rounded-[8px] transition-all",
                mobileTab === "catalog"
                  ? "bg-[var(--color-surface-base)] text-[var(--color-text-primary)] shadow-xs"
                  : "text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
              )}
            >
              {t("pos.catalog.title") || "Katalog"}
            </button>
            <button
              type="button"
              onClick={() => setMobileTab("cart")}
              className={cn(
                "flex-1 py-1.5 px-3 text-xs font-semibold rounded-[8px] transition-all flex items-center justify-center gap-1.5",
                mobileTab === "cart"
                  ? "bg-[var(--color-surface-base)] text-[var(--color-text-primary)] shadow-xs"
                  : "text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
              )}
            >
              <span>{t("pos.cart.title")}</span>
              {cart.totals.totalItems > 0 && (
                <span className="px-1.5 py-0.2 rounded-full text-[10px] font-bold bg-[var(--color-action-primary)] text-white">
                  {cart.totals.totalItems}
                </span>
              )}
            </button>
          </div>

          {/* Left Panel: Catalog & Search Grid */}
          <div
            className={cn(
              "flex-1 min-w-0 lg:h-full lg:overflow-hidden",
              mobileTab === "catalog" ? "block" : "hidden lg:block",
              cart.totals.totalItems > 0 ? "pb-20 lg:pb-0" : ""
            )}
          >
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

          {/* Right Panel: Cart & Checkout Summary (Full width on mobile cart tab, fixed side column on desktop) */}
          <div
            className={cn(
              "w-full lg:w-[380px] xl:w-[420px] shrink-0 lg:h-full",
              mobileTab === "cart" ? "block h-[calc(100vh-230px)] min-h-[480px]" : "hidden lg:block"
            )}
          >
            {mobileTab === "cart" && (
              <div className="lg:hidden mb-2">
                <button
                  type="button"
                  onClick={() => setMobileTab("catalog")}
                  className="flex items-center gap-1.5 text-xs font-semibold text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] px-2.5 py-1.5 rounded-lg bg-[var(--color-surface-base)] border border-[var(--color-border-hairline)] shadow-2xs"
                >
                  <ArrowLeft className="w-3.5 h-3.5" />
                  <span>{t("pos.cart.backToCatalog")}</span>
                </button>
              </div>
            )}
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

          {/* Mobile Bottom Sticky Bar for Quick Cart Access from Catalog */}
          {mobileTab === "catalog" && cart.totals.totalItems > 0 && (
            <div className="lg:hidden fixed bottom-4 left-4 right-4 z-20">
              <button
                type="button"
                onClick={() => setMobileTab("cart")}
                className="w-full h-14 bg-[var(--color-action-primary)] text-white rounded-[18px] px-4 flex items-center justify-between shadow-xl active:scale-[0.98] transition-transform"
              >
                <div className="flex items-center gap-2.5">
                  <div className="w-8 h-8 rounded-full bg-white/20 flex items-center justify-center font-bold text-xs">
                    {cart.totals.totalItems}
                  </div>
                  <div className="text-left">
                    <p className="text-[10px] uppercase font-semibold tracking-wider text-white/80 leading-tight">
                      {t("pos.cart.title")}
                    </p>
                    <p className="text-sm font-bold leading-tight font-mono">
                      {formatIDR(cart.totals.total)}
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-1.5 text-xs font-bold bg-white/10 px-3 py-1.5 rounded-full">
                  <span>{t("pos.cart.viewCart")}</span>
                  <ArrowRight className="w-3.5 h-3.5" />
                </div>
              </button>
            </div>
          )}
        </div>
      ) : (
        <OrderHistory />
      )}

      </div>

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
        commandReference={checkout.idempotencyKey}
      />

      {/* Receipt Modal Dialog */}
      <ReceiptModal
        isOpen={checkout.step === "completed" && checkout.receipt !== null}
        onClose={handleNewTransaction}
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
