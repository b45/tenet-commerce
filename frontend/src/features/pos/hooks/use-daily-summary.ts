"use client";

import * as React from "react";
import { apiClient } from "@/lib/api";
import type { DailySummaryResponse } from "../types";

export interface UseDailySummaryReturn {
  summary: DailySummaryResponse | null;
  isLoading: boolean;
  error: string | null;
  selectedDate: string;
  setSelectedDate: (date: string) => void;
  refetch: () => Promise<void>;
}

export function useDailySummary(initialDate?: string, enabled: boolean = true): UseDailySummaryReturn {
  const [selectedDate, setSelectedDate] = React.useState<string>(() => {
    if (initialDate) return initialDate;
    const now = new Date();
    const year = now.getFullYear();
    const month = String(now.getMonth() + 1).padStart(2, "0");
    const day = String(now.getDate()).padStart(2, "0");
    return `${year}-${month}-${day}`;
  });
  const [summary, setSummary] = React.useState<DailySummaryResponse | null>(null);
  const [isLoading, setIsLoading] = React.useState<boolean>(false);
  const [error, setError] = React.useState<string | null>(null);

  const fetchSummary = React.useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const endpoint = selectedDate
        ? `/pos/daily-summary?date=${encodeURIComponent(selectedDate)}`
        : "/pos/daily-summary";
      const res = await apiClient.get<DailySummaryResponse>(endpoint);
      if (res.success && res.data) {
        setSummary(res.data);
      } else {
        setError(res.error?.message || "Gagal memuat ringkasan harian");
      }
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Gagal memuat ringkasan harian";
      setError(message);
    } finally {
      setIsLoading(false);
    }
  }, [selectedDate]);

  React.useEffect(() => {
    if (!enabled) return;
    fetchSummary();
  }, [fetchSummary, enabled]);

  return {
    summary,
    isLoading,
    error,
    selectedDate,
    setSelectedDate,
    refetch: fetchSummary,
  };
}
