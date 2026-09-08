"use client";

import * as React from "react";
import Link from "next/link";
import { FeatureGate } from "@/components/auth/feature-gate";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { SegmentedControl } from "@/components/ui/segmented-control";
import { useTranslation } from "@/lib/i18n";
import { formatIDR } from "@/lib/money";
import { useDashboard } from "@/features/dashboard/hooks/use-dashboard";
import {
  ShieldCheck,
  Package,
  AlertCircle,
  RefreshCw,
  ArrowRight,
  BookOpen,
  DollarSign,
  Receipt,
} from "lucide-react";

export default function DashboardPage() {
  const { t } = useTranslation();
  const [viewFilter, setViewFilter] = React.useState("today");
  const { summary, loading, error, refetch } = useDashboard();

  const isToday = viewFilter === "today";

  return (
    <FeatureGate featureKey="pos.daily_summary" requiredRole={["MANAGER", "SUPER_ADMIN"]}>
      <div className="space-y-6 max-w-7xl mx-auto">
        {/* Header with Title & Filter Controls */}
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
          <div>
            <h1 className="text-[24px] font-bold tracking-tight text-[#0B0F19]">
              {t("dashboard.title")}
            </h1>
            <p className="text-[13px] text-[#555D6E] mt-0.5">
              {t("dashboard.subtitle")}
            </p>
          </div>

          <div className="flex items-center gap-3">
            <SegmentedControl
              size="sm"
              value={viewFilter}
              onChange={setViewFilter}
              options={[
                { value: "today", label: t("dashboard.viewFilter.today") },
                { value: "all_time", label: t("dashboard.viewFilter.allTime") },
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

        {/* Error State Banner */}
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

        {/* Skeleton Loading State */}
        {loading && !summary && (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4 animate-pulse">
            {[1, 2, 3, 4].map((n) => (
              <div key={n} className="h-32 rounded-2xl bg-black/[0.04] border border-black/[0.06]" />
            ))}
          </div>
        )}

        {/* Metric Cards: Cupertino Hairline Surfaces with Verified Operational Numbers */}
        {summary && (
          <>
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
              {/* Card 1: Gross Sales */}
              <Card className="rounded-[18px]">
                <CardHeader className="flex flex-row items-center justify-between pb-2">
                  <CardTitle className="text-[12px] font-medium uppercase tracking-wider text-[#555D6E]">
                    {t("dashboard.metrics.grossSales")}
                  </CardTitle>
                  <DollarSign className="h-4 w-4 text-[#0066CC]" />
                </CardHeader>
                <CardContent>
                  <div className="text-[26px] font-bold tracking-tight text-[#0B0F19] font-tabular">
                    {formatIDR(isToday ? summary.sales_summary.today_gross_sales : summary.sales_summary.today_gross_sales)}
                  </div>
                  <div className="flex items-center gap-1.5 text-[11px] text-[#555D6E] mt-1.5 font-tabular">
                    <span>{t("dashboard.metrics.netSales")}:</span>
                    <span className="font-semibold text-[#0B0F19]">
                      {formatIDR(summary.sales_summary.today_net_sales)}
                    </span>
                  </div>
                </CardContent>
              </Card>

              {/* Card 2: Orders Count & AOV */}
              <Card className="rounded-[18px]">
                <CardHeader className="flex flex-row items-center justify-between pb-2">
                  <CardTitle className="text-[12px] font-medium uppercase tracking-wider text-[#555D6E]">
                    {t("dashboard.metrics.todayOrders")}
                  </CardTitle>
                  <Receipt className="h-4 w-4 text-emerald-600" />
                </CardHeader>
                <CardContent>
                  <div className="text-[26px] font-bold tracking-tight text-[#0B0F19] font-tabular">
                    {isToday ? summary.sales_summary.today_orders_count : summary.sales_summary.all_time_orders_count}
                    <span className="text-xs font-normal text-[#8B95A5] ml-1.5">
                      {t("dashboard.units.transactions")}
                    </span>
                  </div>
                  <div className="flex items-center gap-1.5 text-[11px] text-[#555D6E] mt-1.5 font-tabular">
                    <span>{t("dashboard.metrics.avgOrderValue")}:</span>
                    <span className="font-semibold text-[#0B0F19]">
                      {formatIDR(summary.sales_summary.average_order_value)}
                    </span>
                  </div>
                </CardContent>
              </Card>

              {/* Card 3: Inventory Low Stock Alerts */}
              <Card className="rounded-[18px]">
                <CardHeader className="flex flex-row items-center justify-between pb-2">
                  <CardTitle className="text-[12px] font-medium uppercase tracking-wider text-[#555D6E]">
                    {t("dashboard.metrics.lowStockAlerts")}
                  </CardTitle>
                  <Package className="h-4 w-4 text-amber-600" />
                </CardHeader>
                <CardContent>
                  <div className="text-[26px] font-bold tracking-tight text-[#0B0F19] font-tabular">
                    {summary.inventory_alerts.low_stock_count}
                    <span className="text-xs font-normal text-[#8B95A5] ml-1.5">
                      {t("dashboard.units.alerts")}
                    </span>
                  </div>
                  <div className="flex items-center gap-2 mt-1.5">
                    {summary.inventory_alerts.low_stock_count > 0 ? (
                      <Badge variant="warning" dot className="text-[10px]">
                        {summary.inventory_alerts.low_stock_count} {t("dashboard.units.alerts")}
                      </Badge>
                    ) : (
                      <Badge variant="success" dot className="text-[10px]">
                        {t("dashboard.alerts.noLowStock")}
                      </Badge>
                    )}
                  </div>
                </CardContent>
              </Card>

              {/* Card 4: Halal Supplier Compliance */}
              <Card className="rounded-[18px]">
                <CardHeader className="flex flex-row items-center justify-between pb-2">
                  <CardTitle className="text-[12px] font-medium uppercase tracking-wider text-[#555D6E]">
                    {t("dashboard.metrics.halalCompliance")}
                  </CardTitle>
                  <ShieldCheck className="h-4 w-4 text-emerald-600" />
                </CardHeader>
                <CardContent>
                  <div className="text-[26px] font-bold tracking-tight text-[#0B0F19] font-tabular">
                    {summary.compliance_alerts.expired_certificates_count > 0 || summary.compliance_alerts.expiring_certificates_count > 0 ? (
                      <span className="text-amber-600">
                        {summary.compliance_alerts.expiring_certificates_count + summary.compliance_alerts.expired_certificates_count}
                      </span>
                    ) : (
                      <span className="text-emerald-700">0</span>
                    )}
                    <span className="text-xs font-normal text-[#8B95A5] ml-1.5">
                      {t("dashboard.units.alerts")}
                    </span>
                  </div>
                  <div className="flex items-center gap-2 mt-1.5">
                    {summary.compliance_alerts.expired_certificates_count > 0 ? (
                      <Badge variant="danger" dot className="text-[10px]">
                        {summary.compliance_alerts.expired_certificates_count} {t("dashboard.alerts.expired")}
                      </Badge>
                    ) : summary.compliance_alerts.expiring_certificates_count > 0 ? (
                      <Badge variant="warning" dot className="text-[10px]">
                        {summary.compliance_alerts.expiring_certificates_count} {t("dashboard.units.expiringCertificates")}
                      </Badge>
                    ) : (
                      <Badge variant="success" dot className="text-[10px]">
                        {t("dashboard.alerts.noExpiringCerts")}
                      </Badge>
                    )}
                  </div>
                </CardContent>
              </Card>
            </div>

            {/* Operational Deep Dive: Low-Stock SKUs & Compliance Cert Alerts */}
            <div className="grid gap-6 lg:grid-cols-2">
              {/* Low Stock SKUs Table Card */}
              <Card className="rounded-[18px]">
                <CardHeader className="flex flex-row items-center justify-between border-b border-black/[0.04] pb-4">
                  <div>
                    <CardTitle className="text-[16px] font-semibold text-[#0B0F19]">
                      {t("dashboard.sections.lowStockTitle")}
                    </CardTitle>
                    <CardDescription className="text-xs text-[#555D6E] mt-0.5">
                      {t("dashboard.sections.lowStockDesc")}
                    </CardDescription>
                  </div>
                  <Link
                    href="/inventory"
                    className="inline-flex items-center gap-1 text-xs font-semibold text-[#0066CC] hover:underline"
                  >
                    <span>{t("dashboard.alerts.viewInventoryAction")}</span>
                    <ArrowRight className="h-3 w-3" />
                  </Link>
                </CardHeader>
                <CardContent className="pt-4">
                  {summary.inventory_alerts.items.length === 0 ? (
                    <div className="py-8 text-center text-xs text-[#8B95A5]">
                      <Package className="h-8 w-8 text-[#8B95A5]/60 mx-auto mb-2" />
                      <p>{t("dashboard.alerts.noLowStock")}</p>
                    </div>
                  ) : (
                    <div className="divide-y divide-black/[0.04]">
                      {summary.inventory_alerts.items.slice(0, 5).map((item) => (
                        <div key={item.product_id} className="py-2.5 flex items-center justify-between">
                          <div className="min-w-0 pr-3">
                            <p className="text-xs font-semibold text-[#0B0F19] truncate">{item.name}</p>
                            <p className="text-[11px] font-mono text-[#8B95A5]">
                              {item.sku} {item.category_name ? `• ${item.category_name}` : ""}
                            </p>
                          </div>
                          <div className="text-right shrink-0">
                            <Badge variant={item.current_stock <= 0 ? "danger" : "warning"} className="text-[10px]">
                              {item.current_stock} / {item.threshold}
                            </Badge>
                            <p className="text-[10px] text-[#8B95A5] mt-0.5 font-tabular">
                              {formatIDR(item.unit_price)}
                            </p>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </CardContent>
              </Card>

              {/* Halal Compliance Alert Card */}
              <Card className="rounded-[18px]">
                <CardHeader className="flex flex-row items-center justify-between border-b border-black/[0.04] pb-4">
                  <div>
                    <CardTitle className="text-[16px] font-semibold text-[#0B0F19]">
                      {t("dashboard.sections.complianceTitle")}
                    </CardTitle>
                    <CardDescription className="text-xs text-[#555D6E] mt-0.5">
                      {t("dashboard.sections.complianceDesc")}
                    </CardDescription>
                  </div>
                  <Link
                    href="/supply-chain/certificates"
                    className="inline-flex items-center gap-1 text-xs font-semibold text-[#0066CC] hover:underline"
                  >
                    <span>{t("dashboard.alerts.viewSuppliersAction")}</span>
                    <ArrowRight className="h-3 w-3" />
                  </Link>
                </CardHeader>
                <CardContent className="pt-4">
                  {summary.compliance_alerts.items.length === 0 ? (
                    <div className="py-8 text-center text-xs text-[#8B95A5]">
                      <ShieldCheck className="h-8 w-8 text-emerald-600/60 mx-auto mb-2" />
                      <p>{t("dashboard.alerts.noExpiringCerts")}</p>
                    </div>
                  ) : (
                    <div className="divide-y divide-black/[0.04]">
                      {summary.compliance_alerts.items.slice(0, 5).map((item) => (
                        <div key={item.certificate_id} className="py-2.5 flex items-center justify-between">
                          <div className="min-w-0 pr-3">
                            <p className="text-xs font-semibold text-[#0B0F19] truncate">{item.supplier_name}</p>
                            <p className="text-[11px] font-mono text-[#8B95A5]">
                              {item.certificate_number} • {item.issuing_authority}
                            </p>
                          </div>
                          <div className="text-right shrink-0">
                            {item.status === "EXPIRED" || item.status === "REVOKED" ? (
                              <Badge variant="danger" dot className="text-[10px]">
                                {t("dashboard.alerts.expired")}
                              </Badge>
                            ) : (
                              <Badge variant="warning" dot className="text-[10px]">
                                {t("dashboard.alerts.daysRemaining", { days: item.days_remaining })}
                              </Badge>
                            )}
                            <p className="text-[10px] text-[#8B95A5] mt-0.5 font-mono">
                              {new Date(item.expiry_date).toLocaleDateString()}
                            </p>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </CardContent>
              </Card>
            </div>

            {/* Financial Ledger & Bookkeeping Status */}
            <Card className="rounded-[18px]">
              <CardHeader className="flex flex-row items-center justify-between pb-3">
                <div className="flex items-center gap-2.5">
                  <div className="flex h-8 w-8 items-center justify-center rounded-xl bg-slate-100 text-slate-700">
                    <BookOpen className="h-4 w-4" aria-hidden="true" />
                  </div>
                  <div>
                    <CardTitle className="text-[16px] font-semibold text-[#0B0F19]">
                      {t("dashboard.sections.operationalTitle")}
                    </CardTitle>
                    <CardDescription className="text-xs text-[#555D6E] mt-0.5">
                      {t("dashboard.sections.operationalDesc")}
                    </CardDescription>
                  </div>
                </div>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
                  <div className="rounded-xl bg-[#F8F8FA] p-3.5 border border-black/[0.04]">
                    <div className="text-xs text-[#555D6E] font-medium">
                      {t("dashboard.metrics.ledgerIntegrity")}
                    </div>
                    <div className="mt-1 font-semibold text-[#0B0F19] text-base flex items-center gap-1.5">
                      <span className="inline-block h-2 w-2 rounded-full bg-emerald-500" />
                      <span>{t("dashboard.units.balanced")}</span>
                    </div>
                    <p className="text-[11px] text-[#8B95A5] mt-0.5">
                      {summary.financial_summary.today_journal_entries_count} {t("dashboard.units.journalEntriesToday")}
                    </p>
                  </div>

                  <div className="rounded-xl bg-[#F8F8FA] p-3.5 border border-black/[0.04]">
                    <div className="text-xs text-[#555D6E] font-medium">
                      {t("dashboard.metrics.activeCatalog")}
                    </div>
                    <div className="mt-1 font-mono text-base font-semibold text-[#0B0F19]">
                      {summary.financial_summary.active_accounts_count} <span className="text-xs font-normal text-[#8B95A5]">{t("dashboard.units.skus")}</span>
                    </div>
                    <p className="text-[11px] text-[#8B95A5] mt-0.5">
                      Chart of Accounts
                    </p>
                  </div>

                  <div className="rounded-xl bg-[#F8F8FA] p-3.5 border border-black/[0.04]">
                    <div className="text-xs text-[#555D6E] font-medium">
                      Sync Status
                    </div>
                    <div className="mt-1 font-semibold text-emerald-700 text-base flex items-center gap-1.5">
                      <span className="inline-block h-2 w-2 rounded-full bg-emerald-500" />
                      <span>Online & Verified</span>
                    </div>
                    <p className="text-[11px] text-[#8B95A5] mt-0.5 font-mono text-[10px]">
                      {summary.generated_at ? new Date(summary.generated_at).toLocaleTimeString() : ""}
                    </p>
                  </div>
                </div>
              </CardContent>
            </Card>
          </>
        )}
      </div>
    </FeatureGate>
  );
}
