"use client";

import * as React from "react";
import { ShieldCheck, Plus, Search, Building2, AlertTriangle, XCircle, CheckCircle2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useTranslation } from "@/lib/i18n";
import { useSuppliers } from "@/features/supply-chain/hooks/use-suppliers";
import { SupplierTable } from "@/features/supply-chain/components/supplier-table";
import { ExpiryAlertBanner } from "@/features/supply-chain/components/expiry-alert-banner";
import { SupplierModal } from "@/features/supply-chain/components/supplier-modal";
import { RegisterCertModal } from "@/features/supply-chain/components/register-cert-modal";
import { CertHistoryModal } from "@/features/supply-chain/components/cert-history-modal";
import type { Supplier } from "@/features/supply-chain/types";

export default function CertificatesPage() {
  const { t } = useTranslation();
  const {
    isLoading,
    searchQuery,
    setSearchQuery,
    statusFilter,
    setStatusFilter,
    certTypeFilter,
    setCertTypeFilter,
    filteredSuppliers,
    expirySummary,
    createSupplier,
    registerCertificate,
    revokeCertificate,
    getSupplierCertificates,
  } = useSuppliers();

  // Modal States
  const [isAddSupplierOpen, setIsAddSupplierOpen] = React.useState(false);
  const [renewSupplier, setRenewSupplier] = React.useState<Supplier | null>(null);
  const [historySupplier, setHistorySupplier] = React.useState<Supplier | null>(null);

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 pb-2 border-b border-[var(--color-border-hairline)]">
        <div>
          <div className="flex items-center gap-2">
            <h1 className="text-xl sm:text-2xl font-bold tracking-tight text-[var(--color-text-primary)]">
              {t("supplyChain.title")}
            </h1>
            <span className="hidden sm:inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-[11px] font-bold bg-[var(--color-status-success-bg)] text-[var(--color-status-success-text)] border border-[var(--color-status-success-border)]">
              <ShieldCheck className="h-3 w-3" />
              <span>{t("supplyChain.badge")}</span>
            </span>
          </div>
          <p className="text-xs text-[var(--color-text-secondary)] mt-0.5">
            {t("supplyChain.subtitle")}
          </p>
        </div>

        <Button
          type="button"
          onClick={() => setIsAddSupplierOpen(true)}
          className="gap-2 shrink-0 h-10 px-4 text-xs font-semibold shadow-xs"
        >
          <Plus className="h-4 w-4" />
          <span>{t("supplyChain.addSupplier")}</span>
        </Button>
      </div>

      {/* Compliance Metrics Summary Cards */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-3.5">
        <div className="p-4 rounded-xl border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] shadow-2xs space-y-1">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-[var(--color-text-secondary)]">
              {t("supplyChain.stats.totalSuppliers")}
            </span>
            <Building2 className="h-4 w-4 text-[var(--color-text-tertiary)]" />
          </div>
          <div className="text-2xl font-extrabold tracking-tight text-[var(--color-text-primary)] font-mono">
            {expirySummary.total_active_suppliers}
          </div>
        </div>

        <div className="p-4 rounded-xl border border-[var(--color-status-success-border)] bg-[var(--color-status-success-bg)]/20 shadow-2xs space-y-1">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-[var(--color-status-success-text)]">
              {t("supplyChain.stats.validCertificates")}
            </span>
            <CheckCircle2 className="h-4 w-4 text-[var(--color-status-success-text)]" />
          </div>
          <div className="text-2xl font-extrabold tracking-tight text-[var(--color-status-success-text)] font-mono">
            {expirySummary.valid_count}
          </div>
        </div>

        <div className="p-4 rounded-xl border border-[var(--color-status-warning-border)] bg-[var(--color-status-warning-bg)]/30 shadow-2xs space-y-1">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-[var(--color-status-warning-text)]">
              {t("supplyChain.stats.expiringSoon")}
            </span>
            <AlertTriangle className="h-4 w-4 text-[var(--color-status-warning-text)]" />
          </div>
          <div className="text-2xl font-extrabold tracking-tight text-[var(--color-status-warning-text)] font-mono">
            {expirySummary.expiring_soon_count}
          </div>
        </div>

        <div className="p-4 rounded-xl border border-[var(--color-status-error-border)] bg-[var(--color-status-error-bg)]/20 shadow-2xs space-y-1">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-[var(--color-status-error-text)]">
              {t("supplyChain.stats.expiredBlocked")}
            </span>
            <XCircle className="h-4 w-4 text-[var(--color-status-error-text)]" />
          </div>
          <div className="text-2xl font-extrabold tracking-tight text-[var(--color-status-error-text)] font-mono">
            {expirySummary.expired_count}
          </div>
        </div>
      </div>

      {/* Proactive Invariant Expiry Alert Banner */}
      <ExpiryAlertBanner
        summary={expirySummary}
        onFilterExpiring={() => setStatusFilter("EXPIRING_SOON")}
      />

      {/* Search & Status Filter Bar */}
      <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-3">
        {/* Status Filter Pills & Cert Type Selector */}
        <div className="flex flex-wrap items-center gap-2">
          <div className="flex flex-wrap items-center gap-1.5 p-1 bg-[var(--color-surface-muted)] rounded-xl border border-[var(--color-border-hairline)]">
            <button
              type="button"
              onClick={() => setStatusFilter("ALL")}
              className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all ${
                statusFilter === "ALL"
                  ? "bg-[var(--color-surface-base)] text-[var(--color-text-primary)] shadow-2xs"
                  : "text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
              }`}
            >
              {t("supplyChain.filterAll")}
            </button>
            <button
              type="button"
              onClick={() => setStatusFilter("VALID")}
              className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all ${
                statusFilter === "VALID"
                  ? "bg-[var(--color-surface-base)] text-[var(--color-text-primary)] shadow-2xs"
                  : "text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
              }`}
            >
              {t("supplyChain.filterValid")}
            </button>
            <button
              type="button"
              onClick={() => setStatusFilter("EXPIRING_SOON")}
              className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all ${
                statusFilter === "EXPIRING_SOON"
                  ? "bg-[var(--color-surface-base)] text-[var(--color-text-primary)] shadow-2xs"
                  : "text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
              }`}
            >
              {t("supplyChain.filterExpiringSoon")}
            </button>
            <button
              type="button"
              onClick={() => setStatusFilter("EXPIRED")}
              className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all ${
                statusFilter === "EXPIRED"
                  ? "bg-[var(--color-surface-base)] text-[var(--color-text-primary)] shadow-2xs"
                  : "text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
              }`}
            >
              {t("supplyChain.filterExpired")}
            </button>
            <button
              type="button"
              onClick={() => setStatusFilter("NO_CERT")}
              className={`px-3 py-1.5 text-xs font-semibold rounded-lg transition-all ${
                statusFilter === "NO_CERT"
                  ? "bg-[var(--color-surface-base)] text-[var(--color-text-primary)] shadow-2xs"
                  : "text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
              }`}
            >
              {t("supplyChain.filterNoCert")}
            </button>
          </div>

          {/* Standard Type Dropdown Filter */}
          <div className="flex items-center gap-1.5">
            <select
              value={certTypeFilter}
              onChange={(e) => setCertTypeFilter(e.target.value)}
              className="h-9 px-3 text-xs font-semibold rounded-xl border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] text-[var(--color-text-primary)] focus:outline-hidden focus:ring-2 focus:ring-[var(--color-action-primary)] shadow-2xs"
              aria-label={t("supplyChain.certTypes.allTypes")}
            >
              <option value="ALL">{t("supplyChain.certTypes.allTypes")}</option>
              <option value="HALAL">{t("supplyChain.certTypes.halal")}</option>
              <option value="BPOM">{t("supplyChain.certTypes.bpom")}</option>
              <option value="REGALKES">{t("supplyChain.certTypes.regalkes")}</option>
              <option value="MICHELIN">{t("supplyChain.certTypes.michelin")}</option>
              <option value="ISO_HACCP">{t("supplyChain.certTypes.isoHaccp")}</option>
              <option value="OTHER">{t("supplyChain.certTypes.other")}</option>
            </select>
          </div>
        </div>

        {/* Search Bar */}
        <div className="relative w-full lg:w-72">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-[var(--color-text-tertiary)] rtl:left-auto rtl:right-3" />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder={t("supplyChain.searchPlaceholder")}
            className="w-full h-9 pl-9 pr-3 rtl:pl-3 rtl:pr-9 text-xs rounded-xl border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] text-[var(--color-text-primary)] placeholder:text-[var(--color-text-tertiary)] focus:outline-hidden focus:ring-2 focus:ring-[var(--color-action-primary)] shadow-2xs"
          />
        </div>
      </div>

      {/* Main Table */}
      {isLoading ? (
        <div className="py-20 text-center text-sm font-semibold text-[var(--color-text-secondary)] animate-pulse">
          {t("supplyChain.loading")}
        </div>
      ) : (
        <SupplierTable
          suppliers={filteredSuppliers}
          onOpenRenewModal={(s) => setRenewSupplier(s)}
          onOpenHistoryModal={(s) => setHistorySupplier(s)}
        />
      )}

      {/* Modals */}
      <SupplierModal
        isOpen={isAddSupplierOpen}
        onClose={() => setIsAddSupplierOpen(false)}
        onSubmit={createSupplier}
      />

      <RegisterCertModal
        isOpen={renewSupplier !== null}
        supplier={renewSupplier}
        onClose={() => setRenewSupplier(null)}
        onSubmit={registerCertificate}
      />

      <CertHistoryModal
        isOpen={historySupplier !== null}
        supplier={historySupplier}
        onClose={() => setHistorySupplier(null)}
        onFetchCertificates={getSupplierCertificates}
        onRevokeCertificate={revokeCertificate}
      />
    </div>
  );
}
