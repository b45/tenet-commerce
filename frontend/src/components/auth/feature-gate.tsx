"use client";

import * as React from "react";
import { useTranslation } from "@/lib/i18n";
import { useAuth } from "@/features/auth/hooks/use-auth";
import { useCapabilities } from "@/features/auth/hooks/use-capabilities";
import { Lock, ShieldAlert, ToggleLeft, Gauge, WifiOff, RotateCw, Sparkles } from "lucide-react";

export type GateState =
  | "loading"
  | "insufficient_subscription"
  | "insufficient_permission"
  | "owner_disabled"
  | "quota_exceeded"
  | "service_unavailable"
  | "allowed";

export interface FeatureGateProps {
  featureKey: string;
  requiredPermission?: string;
  requiredRole?: string | string[];
  children: React.ReactNode;
  fallback?: React.ReactNode | ((state: GateState) => React.ReactNode);
  hideIfDenied?: boolean;
  className?: string;
  onUpgradeClick?: () => void;
}

export function FeatureGate({
  featureKey,
  requiredPermission,
  requiredRole,
  children,
  fallback,
  hideIfDenied = false,
  className = "",
  onUpgradeClick,
}: FeatureGateProps) {
  const { t, direction } = useTranslation();
  const { hasPermission, hasRole, isLoading: isAuthLoading } = useAuth();
  const { isLoading: isCapLoading, isServiceUnavailable, checkFeature, refresh } = useCapabilities();

  const [isRetrying, setIsRetrying] = React.useState(false);

  const handleRetry = async () => {
    setIsRetrying(true);
    try {
      await refresh();
    } finally {
      setIsRetrying(false);
    }
  };

  // 1. Evaluate Current Gate State
  const state: GateState = React.useMemo(() => {
    if (isAuthLoading || isCapLoading) {
      return "loading";
    }

    if (isServiceUnavailable) {
      return "service_unavailable";
    }

    // RBAC Permission Guard
    if (requiredPermission && !hasPermission(requiredPermission)) {
      return "insufficient_permission";
    }

    // RBAC Role Guard
    if (requiredRole) {
      const allowed = Array.isArray(requiredRole)
        ? requiredRole.some((r) => hasRole(r))
        : hasRole(requiredRole);
      if (!allowed) {
        return "insufficient_permission";
      }
    }

    // Authoritative Feature Entitlement Guard
    const decision = checkFeature(featureKey);
    if (decision.allowed) {
      return "allowed";
    }

    if (decision.reason === "FEATURE_DISABLED") {
      return "owner_disabled";
    }

    if (decision.reason === "QUOTA_EXCEEDED") {
      return "quota_exceeded";
    }

    if (decision.reason === "SERVICE_UNAVAILABLE") {
      return "service_unavailable";
    }

    // Default entitlement denial: insufficient subscription tier / expired
    return "insufficient_subscription";
  }, [
    isAuthLoading,
    isCapLoading,
    isServiceUnavailable,
    requiredPermission,
    hasPermission,
    requiredRole,
    hasRole,
    checkFeature,
    featureKey,
  ]);

  // If state is allowed, render protected children directly
  if (state === "allowed") {
    return <>{children}</>;
  }

  // If user requested silent hiding on denial
  if (hideIfDenied && state !== "loading") {
    return null;
  }

  // If a custom fallback is provided
  if (fallback !== undefined) {
    if (typeof fallback === "function") {
      return <>{fallback(state)}</>;
    }
    return <>{fallback}</>;
  }

  const baseContainerStyles =
    "relative overflow-hidden rounded-2xl border p-6 transition-all duration-300";
  const glassCardStyles =
    "bg-white/80 dark:bg-slate-900/80 backdrop-blur-xl border-slate-200/80 dark:border-slate-800 shadow-sm";

  // State 1: Loading Skeleton (Cupertino Pulse)
  if (state === "loading") {
    return (
      <div
        className={`${baseContainerStyles} ${glassCardStyles} animate-pulse ${className}`}
        dir={direction}
        aria-busy="true"
        aria-label={t("entitlements.loading")}
      >
        <div className="flex items-center gap-4">
          <div className="h-12 w-12 rounded-xl bg-slate-200 dark:bg-slate-800" />
          <div className="flex-1 space-y-2">
            <div className="h-4 w-1/3 rounded-lg bg-slate-200 dark:bg-slate-800" />
            <div className="h-3 w-2/3 rounded-lg bg-slate-100 dark:bg-slate-800/60" />
          </div>
        </div>
      </div>
    );
  }

  // State 2: Insufficient Subscription (Upgrade Required)
  if (state === "insufficient_subscription") {
    return (
      <div
        className={`${baseContainerStyles} ${glassCardStyles} border-amber-500/20 bg-gradient-to-br from-amber-500/5 via-transparent to-purple-500/5 ${className}`}
        dir={direction}
        role="alert"
      >
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
          <div className="flex items-start gap-4">
            <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-gradient-to-tr from-amber-500/20 to-purple-500/20 border border-amber-500/30 text-amber-600 dark:text-amber-400">
              <Lock className="h-6 w-6" aria-hidden="true" />
            </div>
            <div>
              <h4 className="text-base font-semibold text-slate-900 dark:text-slate-100">
                {t("entitlements.insufficientSubscription.title")}
              </h4>
              <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
                {t("entitlements.insufficientSubscription.description")}
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={onUpgradeClick}
            className="inline-flex items-center gap-2 rounded-xl bg-gradient-to-r from-amber-600 to-purple-600 hover:from-amber-700 hover:to-purple-700 px-4 py-2.5 text-sm font-medium text-white shadow-sm transition-all duration-200 hover:shadow-md focus:outline-none focus:ring-2 focus:ring-amber-500/50"
          >
            <Sparkles className="h-4 w-4" aria-hidden="true" />
            <span>{t("entitlements.insufficientSubscription.upgradeAction")}</span>
          </button>
        </div>
      </div>
    );
  }

  // State 3: Insufficient Permission (User RBAC Denied)
  if (state === "insufficient_permission") {
    return (
      <div
        className={`${baseContainerStyles} ${glassCardStyles} border-rose-500/20 bg-rose-500/5 ${className}`}
        dir={direction}
        role="alert"
      >
        <div className="flex items-start gap-4">
          <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-600 dark:text-rose-400">
            <ShieldAlert className="h-6 w-6" aria-hidden="true" />
          </div>
          <div>
            <h4 className="text-base font-semibold text-slate-900 dark:text-slate-100">
              {t("entitlements.insufficientPermission.title")}
            </h4>
            <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
              {t("entitlements.insufficientPermission.description")}
            </p>
          </div>
        </div>
      </div>
    );
  }

  // State 4: Disabled by Store Owner
  if (state === "owner_disabled") {
    return (
      <div
        className={`${baseContainerStyles} ${glassCardStyles} border-slate-300 dark:border-slate-800 ${className}`}
        dir={direction}
        role="alert"
      >
        <div className="flex items-start gap-4">
          <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-slate-100 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 text-slate-500">
            <ToggleLeft className="h-6 w-6" aria-hidden="true" />
          </div>
          <div>
            <h4 className="text-base font-semibold text-slate-900 dark:text-slate-100">
              {t("entitlements.disabledByOwner.title")}
            </h4>
            <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
              {t("entitlements.disabledByOwner.description")}
            </p>
          </div>
        </div>
      </div>
    );
  }

  // State 5: Quota Exceeded
  if (state === "quota_exceeded") {
    const decision = checkFeature(featureKey);
    return (
      <div
        className={`${baseContainerStyles} ${glassCardStyles} border-orange-500/20 bg-orange-500/5 ${className}`}
        dir={direction}
        role="alert"
      >
        <div className="flex items-start gap-4">
          <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-orange-500/10 border border-orange-500/20 text-orange-600 dark:text-orange-400">
            <Gauge className="h-6 w-6" aria-hidden="true" />
          </div>
          <div>
            <h4 className="text-base font-semibold text-slate-900 dark:text-slate-100">
              {t("entitlements.quotaExceeded.title")}
            </h4>
            <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
              {t("entitlements.quotaExceeded.description")}
            </p>
            {decision.quota_limit != null && (
              <p className="mt-2 text-xs font-medium text-orange-700 dark:text-orange-300">
                {t("entitlements.quotaExceeded.usageNotice", {
                  current: decision.current_usage ?? 0,
                  limit: decision.quota_limit,
                })}
              </p>
            )}
          </div>
        </div>
      </div>
    );
  }

  // State 6: Service Unavailable (Degraded with Retry)
  return (
    <div
      className={`${baseContainerStyles} ${glassCardStyles} border-amber-500/20 bg-amber-500/5 ${className}`}
      dir={direction}
      role="alert"
    >
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
        <div className="flex items-start gap-4">
          <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-amber-500/10 border border-amber-500/20 text-amber-600 dark:text-amber-400">
            <WifiOff className="h-6 w-6" aria-hidden="true" />
          </div>
          <div>
            <h4 className="text-base font-semibold text-slate-900 dark:text-slate-100">
              {t("entitlements.serviceUnavailable.title")}
            </h4>
            <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
              {t("entitlements.serviceUnavailable.description")}
            </p>
          </div>
        </div>
        <button
          type="button"
          onClick={handleRetry}
          disabled={isRetrying}
          className="inline-flex items-center gap-2 rounded-xl bg-slate-900 hover:bg-slate-800 dark:bg-slate-100 dark:hover:bg-white dark:text-slate-900 px-4 py-2 text-sm font-medium text-white shadow-sm transition-all duration-200 disabled:opacity-50"
        >
          <RotateCw className={`h-4 w-4 ${isRetrying ? "animate-spin" : ""}`} aria-hidden="true" />
          <span>{t("entitlements.serviceUnavailable.retryAction")}</span>
        </button>
      </div>
    </div>
  );
}
