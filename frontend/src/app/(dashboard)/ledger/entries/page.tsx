"use client";

import * as React from "react";
import { FeatureGate } from "@/components/auth/feature-gate";
import { SegmentedControl } from "@/components/ui/segmented-control";
import { Button } from "@/components/ui/button";
import { useTranslation } from "@/lib/i18n";
import { useLedger } from "@/features/ledger/hooks/use-ledger";
import { JournalEntriesTable } from "@/features/ledger/components/journal-entries-table";
import { ChartOfAccountsTable } from "@/features/ledger/components/chart-of-accounts-table";
import { TrialBalanceView } from "@/features/ledger/components/trial-balance-view";
import { RefreshCw, AlertCircle } from "lucide-react";

type LedgerTab = "entries" | "accounts" | "trial_balance";

export default function LedgerEntriesPage() {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = React.useState<LedgerTab>("entries");
  const { accounts, entries, trialBalance, loading, error, refetch } = useLedger();

  return (
    <FeatureGate featureKey="pos.daily_summary" requiredPermission="ledger:read">
      <div className="space-y-6 max-w-7xl mx-auto">
        {/* Header Bar */}
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
          <div>
            <h1 className="text-[24px] font-bold tracking-tight text-[#0B0F19]">
              {t("ledger.title")}
            </h1>
            <p className="text-[13px] text-[#555D6E] mt-0.5">
              {t("ledger.subtitle")}
            </p>
          </div>

          <div className="flex items-center gap-3">
            <SegmentedControl
              size="sm"
              value={activeTab}
              onChange={(val) => setActiveTab(val as LedgerTab)}
              options={[
                { value: "entries", label: t("ledger.tabs.entries") },
                { value: "accounts", label: t("ledger.tabs.accounts") },
                { value: "trial_balance", label: t("ledger.tabs.trialBalance") },
              ]}
            />
            <Button
              variant="outline"
              size="sm"
              onClick={refetch}
              disabled={loading}
              className="gap-1.5 h-8 text-xs font-medium"
            >
              <RefreshCw className={`h-3.5 w-3.5 ${loading ? "animate-spin" : ""}`} aria-hidden="true" />
              <span>{t("inventory.refresh")}</span>
            </Button>
          </div>
        </div>

        {/* Error Banner */}
        {error && (
          <div className="rounded-2xl border border-rose-200 bg-rose-50/60 p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3 text-rose-900">
            <div className="flex items-center gap-2.5">
              <AlertCircle className="h-5 w-5 text-rose-600 shrink-0" aria-hidden="true" />
              <div>
                <p className="text-sm font-semibold">{t("dashboard.errorTitle")}</p>
                <p className="text-xs text-rose-700/90">{error}</p>
              </div>
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={refetch}
              className="bg-white border-rose-300 text-rose-800 hover:bg-rose-100/50 self-start sm:self-auto text-xs"
            >
              {t("inventory.refresh")}
            </Button>
          </div>
        )}

        {/* Tab Content */}
        {activeTab === "entries" && (
          <JournalEntriesTable entries={entries} isLoading={loading} />
        )}
        {activeTab === "accounts" && (
          <ChartOfAccountsTable accounts={accounts} isLoading={loading} />
        )}
        {activeTab === "trial_balance" && (
          <TrialBalanceView trialBalance={trialBalance} isLoading={loading} />
        )}
      </div>
    </FeatureGate>
  );
}
