"use client";

import * as React from "react";
import { apiClient } from "@/lib/api";
import { logger } from "@/lib/logger";
import type { DashboardSummary } from "../types";

export function useDashboard() {
  const [summary, setSummary] = React.useState<DashboardSummary | null>(null);
  const [loading, setLoading] = React.useState<boolean>(true);
  const [error, setError] = React.useState<string | null>(null);

  const fetchDashboard = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await apiClient.get<DashboardSummary>("/manager/dashboard");
      if (res.success && res.data) {
        setSummary(res.data);
      } else {
        const msg = res.error?.message || "Failed to load dashboard metrics";
        setError(msg);
        logger.error("Manager dashboard fetch error", undefined, { error: res.error });
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Network error";
      setError(msg);
      logger.error("Unexpected error fetching manager dashboard", err);
    } finally {
      setLoading(false);
    }
  }, []);

  React.useEffect(() => {
    fetchDashboard();
  }, [fetchDashboard]);

  return {
    summary,
    loading,
    error,
    refetch: fetchDashboard,
  };
}
