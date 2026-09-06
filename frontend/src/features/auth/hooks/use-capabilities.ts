"use client";

import * as React from "react";
import { apiClient, ApiError } from "@/lib/api";
import { useAuth } from "./use-auth";

export type SubscriptionStatus = "ACTIVE" | "TRIALING" | "PAST_DUE" | "CANCELED";

export type DecisionReason =
  | "ENTITLED"
  | "FEATURE_DISABLED"
  | "FEATURE_NOT_ENTITLED"
  | "SUBSCRIPTION_INACTIVE"
  | "NO_SUBSCRIPTION"
  | "QUOTA_EXCEEDED"
  | "SERVICE_UNAVAILABLE";

export interface FeatureDecision {
  allowed: boolean;
  reason: DecisionReason;
  grant_type?: "BOOLEAN" | "QUOTA";
  quota_limit?: number | null;
  current_usage?: number;
}

export interface PlanSummary {
  code: string;
  name: string;
  status: SubscriptionStatus;
}

export interface TenantCapabilities {
  tenant_id: string;
  plan: PlanSummary;
  capabilities: Record<string, FeatureDecision>;
  policy_version: number;
}

export interface UseCapabilitiesReturn {
  capabilities: TenantCapabilities | null;
  isLoading: boolean;
  error: string | null;
  isServiceUnavailable: boolean;
  refresh: () => Promise<void>;
  checkFeature: (featureKey: string) => FeatureDecision;
}

export function useCapabilities(): UseCapabilitiesReturn {
  const { user, isLoading: isAuthLoading } = useAuth();
  const [capabilities, setCapabilities] = React.useState<TenantCapabilities | null>(null);
  const [isLoading, setIsLoading] = React.useState<boolean>(true);
  const [error, setError] = React.useState<string | null>(null);
  const [isServiceUnavailable, setIsServiceUnavailable] = React.useState<boolean>(false);

  const fetchCapabilities = React.useCallback(async () => {
    if (!user) {
      setCapabilities(null);
      setIsLoading(false);
      return;
    }

    setIsLoading(true);
    setError(null);
    setIsServiceUnavailable(false);

    try {
      const res = await apiClient.get<TenantCapabilities>("/me/capabilities");
      if (res.success && res.data) {
        setCapabilities(res.data);
      } else {
        const errCode = res.error?.code || "FETCH_FAILED";
        setError(res.error?.message || "Failed to load capabilities");
        if (errCode === "NETWORK_ERROR" || (res.error?.status && res.error.status >= 500)) {
          setIsServiceUnavailable(true);
        }
      }
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Capability evaluation failed";
      setError(message);
      if (err instanceof ApiError && err.status >= 500) {
        setIsServiceUnavailable(true);
      }
    } finally {
      setIsLoading(false);
    }
  }, [user]);

  React.useEffect(() => {
    if (isAuthLoading) return;
    if (!user) {
      setCapabilities(null);
      setIsLoading(false);
      return;
    }
    fetchCapabilities();
  }, [user, isAuthLoading, fetchCapabilities]);

  const checkFeature = React.useCallback(
    (featureKey: string): FeatureDecision => {
      if (!capabilities || !capabilities.capabilities) {
        return {
          allowed: false,
          reason: isServiceUnavailable ? "SERVICE_UNAVAILABLE" : "FEATURE_NOT_ENTITLED",
        };
      }

      const decision = capabilities.capabilities[featureKey];
      if (!decision) {
        return {
          allowed: false,
          reason: "FEATURE_NOT_ENTITLED",
        };
      }

      return decision;
    },
    [capabilities, isServiceUnavailable]
  );

  return {
    capabilities,
    isLoading: isAuthLoading || isLoading,
    error,
    isServiceUnavailable,
    refresh: fetchCapabilities,
    checkFeature,
  };
}
