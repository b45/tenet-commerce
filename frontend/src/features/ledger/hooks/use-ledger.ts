"use client";

import * as React from "react";
import { apiClient } from "@/lib/api";
import { logger } from "@/lib/logger";
import type { LedgerAccount, LedgerEntry, TrialBalanceSummary } from "../types";

export function useLedger() {
  const [accounts, setAccounts] = React.useState<LedgerAccount[]>([]);
  const [entries, setEntries] = React.useState<LedgerEntry[]>([]);
  const [trialBalance, setTrialBalance] = React.useState<TrialBalanceSummary | null>(null);
  const [loading, setLoading] = React.useState<boolean>(true);
  const [error, setError] = React.useState<string | null>(null);

  const fetchLedgerData = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [accRes, entriesRes, tbRes] = await Promise.all([
        apiClient.get<LedgerAccount[]>("/ledger/accounts"),
        apiClient.get<LedgerEntry[]>("/ledger/entries?limit=50"),
        apiClient.get<TrialBalanceSummary>("/ledger/trial-balance"),
      ]);

      if (accRes.success && accRes.data) {
        setAccounts(Array.isArray(accRes.data) ? accRes.data : []);
      }
      if (entriesRes.success && entriesRes.data) {
        setEntries(Array.isArray(entriesRes.data) ? entriesRes.data : []);
      }
      if (tbRes.success && tbRes.data) {
        setTrialBalance(tbRes.data);
      }

      if (!accRes.success && accRes.error) {
        setError(accRes.error.message);
      } else if (!entriesRes.success && entriesRes.error) {
        setError(entriesRes.error.message);
      } else if (!tbRes.success && tbRes.error) {
        setError(tbRes.error.message);
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Failed to load ledger data";
      setError(msg);
      logger.error("Ledger fetch error", err);
    } finally {
      setLoading(false);
    }
  }, []);

  React.useEffect(() => {
    fetchLedgerData();
  }, [fetchLedgerData]);

  return {
    accounts,
    entries,
    trialBalance,
    loading,
    error,
    refetch: fetchLedgerData,
  };
}
