"use client";

import * as React from "react";
import Link from "next/link";
import { ArrowLeft, ShoppingCart } from "lucide-react";
import { OrderHistory } from "@/features/pos/components/order-history";
import { useTranslation } from "@/lib/i18n";

export default function POSOrdersPage() {
  const { t } = useTranslation();

  return (
    <div className="mx-auto max-w-7xl space-y-4">
      {/* Top Header Bar */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pb-1 border-b border-[var(--color-border-hairline)]">
        <div>
          <h1 className="text-xl sm:text-2xl font-bold tracking-tight text-[var(--color-text-primary)]">
            {t("history.title")}
          </h1>
          <p className="text-xs text-[var(--color-text-secondary)]">
            {t("history.subtitle")}
          </p>
        </div>

        <div className="flex items-center gap-2">
          <Link
            href="/pos"
            className="inline-flex min-h-10 items-center gap-2 px-3.5 py-1.5 rounded-xl text-xs font-semibold bg-[var(--color-surface-base)] border border-[var(--color-border-hairline)] text-[var(--color-text-primary)] hover:bg-[var(--color-surface-muted)] shadow-2xs transition-all focus-visible:outline focus-visible:outline-2 focus-visible:outline-[var(--color-action-primary)] active:scale-95"
          >
            <ArrowLeft className="w-4 h-4 rtl:rotate-180" />
            <ShoppingCart className="w-3.5 h-3.5" />
            <span>{t("pos.title")}</span>
          </Link>
        </div>
      </div>

      {/* Main Order History Component */}
      <OrderHistory hideTitle />
    </div>
  );
}
