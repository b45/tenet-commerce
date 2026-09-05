"use client";

import * as React from "react";
import { ArrowLeft } from "lucide-react";
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
  const catalogHeading = React.useRef<HTMLHeadingElement>(null);
  const cartHeading = React.useRef<HTMLHeadingElement>(null);
  const compactNavigation = React.useRef<HTMLDivElement>(null);
  const catalogScroll = React.useRef(0);
  const pendingPanel = React.useRef<"catalog" | "cart" | null>(null);

  const switchPanel = (panel: "catalog" | "cart") => {
    if (checkout.step !== "idle") return;
    if (mobileTab === "catalog" && panel !== mobileTab) catalogScroll.current = window.scrollY;
    if (panel === mobileTab) {
      (panel === "catalog" ? catalogHeading : cartHeading).current?.focus();
      return;
    }
    pendingPanel.current = panel;
    setMobileTab(panel);
  };

  React.useEffect(() => {
    const panel = pendingPanel.current;
    if (!panel || checkout.step !== "idle" || viewMode !== "register") return;
    const heading = (panel === "catalog" ? catalogHeading : cartHeading).current;
    if (!heading) return;
    pendingPanel.current = null;
    heading.style.scrollMarginTop = `${(compactNavigation.current?.offsetHeight ?? 0) + 16}px`;
    heading.focus({ preventScroll: true });
    if (panel === "catalog") {
      window.scrollTo({ top: catalogScroll.current, behavior: "instant" });
      const navigationBottom = Math.max(0, compactNavigation.current?.getBoundingClientRect().bottom ?? 0);
      const controls = heading.parentElement?.querySelectorAll<HTMLElement>("button:not([disabled]), [tabindex='0']");
      const visible = Array.from(controls ?? []).find(control => {
        const rect = control.getBoundingClientRect();
        return rect.height > 0 && rect.top >= navigationBottom && rect.bottom <= window.innerHeight;
      });
      // Do not restore scroll while leaving keyboard focus on an offscreen heading.
      if (visible) visible.focus({ preventScroll: true });
      else heading.scrollIntoView({ block: "start", behavior: "instant" });
    } else heading.scrollIntoView({ block: "start", behavior: "instant" });
  }, [mobileTab, checkout.step, viewMode]);

  React.useEffect(() => {
    const wide = window.matchMedia("(min-width: 1280px)");
    const restoreVisibleFocus = () => {
      if (checkout.step !== "idle") return;
      const active = document.activeElement;
      const hiddenHeading = mobileTab === "catalog" ? cartHeading : catalogHeading;
      const selectedHeading = mobileTab === "catalog" ? catalogHeading : cartHeading;
      if ((!wide.matches && hiddenHeading.current?.parentElement?.contains(active)) ||
          (wide.matches && compactNavigation.current?.contains(active))) {
        selectedHeading.current?.focus({ preventScroll: true });
      }
    };
    wide.addEventListener("change", restoreVisibleFocus);
    return () => wide.removeEventListener("change", restoreVisibleFocus);
  }, [mobileTab, checkout.step]);

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
      catalogScroll.current = 0;
      pendingPanel.current = "catalog";
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
        </div>
      </div>

      {/* Main Content Workspace */}
      {viewMode === "register" ? (
        <div className="relative flex flex-col xl:flex-row gap-5">
          {/* Mobile sub-tab toggle (Catalog vs Cart) */}
          <div ref={compactNavigation} className="pos-compact-navigation flex flex-wrap xl:hidden items-center p-1 bg-[var(--color-surface-muted)] rounded-[12px] border border-[var(--color-border-hairline)] w-full">
            <button
              type="button"
              onClick={() => switchPanel("catalog")}
              aria-pressed={mobileTab === "catalog"}
              aria-controls="pos-catalog-region"
              className={cn(
                "min-w-0 flex-1 min-h-12 py-2 px-3 text-sm font-semibold rounded-[8px] transition-all",
                mobileTab === "catalog"
                  ? "bg-[var(--color-surface-base)] text-[var(--color-text-primary)] shadow-xs"
                  : "text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
              )}
            >
              {t("pos.catalog.title")}
            </button>
            <button
              type="button"
              onClick={() => switchPanel("cart")}
              aria-pressed={mobileTab === "cart"}
              aria-controls="pos-cart-region"
              className={cn(
                "min-w-0 flex-1 min-h-12 py-2 px-3 text-sm font-semibold rounded-[8px] transition-all flex items-center justify-center gap-1.5",
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
            <p className="w-full break-all px-3 py-2 text-sm text-[var(--color-text-primary)]" aria-live="polite" aria-atomic="true">
              <span>{t("pos.cart.total")}: </span><bdi className="font-mono font-semibold">{formatIDR(cart.totals.total)}</bdi>
            </p>
          </div>

          {/* Left Panel: Catalog & Search Grid */}
          <div
            id="pos-catalog-region" role="region" aria-labelledby="pos-catalog-heading"
            className={cn(
              "flex-1 min-w-0",
              mobileTab === "catalog" ? "block" : "hidden xl:block"
            )}
          >
            <h2 id="pos-catalog-heading" ref={catalogHeading} tabIndex={-1} className="mb-3 scroll-mt-32 text-lg font-semibold focus-visible:outline focus-visible:outline-2">{t("pos.catalog.title")}</h2>
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
            id="pos-cart-region" role="region" aria-labelledby="pos-cart-heading"
            className={cn(
              "w-full xl:w-[380px] 2xl:w-[420px] shrink-0",
              mobileTab === "cart" ? "block" : "hidden xl:block"
            )}
          >
            <h2 id="pos-cart-heading" ref={cartHeading} tabIndex={-1} className="mb-3 scroll-mt-32 text-lg font-semibold focus-visible:outline focus-visible:outline-2">{t("pos.cart.title")}</h2>
            {mobileTab === "cart" && (
              <div className="xl:hidden mb-2">
                <button
                  type="button"
                  onClick={() => switchPanel("catalog")}
                  className="flex min-h-12 items-center gap-2 text-sm font-semibold text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] px-2.5 py-1.5 rounded-lg bg-[var(--color-surface-base)] border border-[var(--color-border-hairline)] shadow-2xs"
                >
                  <ArrowLeft className="w-3.5 h-3.5 rtl:rotate-180" />
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
