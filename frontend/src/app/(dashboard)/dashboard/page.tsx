"use client";

import * as React from "react";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { SegmentedControl } from "@/components/ui/segmented-control";
import { Store, ShieldCheck, ShoppingCart, TrendingUp, ArrowUpRight } from "lucide-react";

export default function DashboardPage() {
  const [viewFilter, setViewFilter] = React.useState("today");

  return (
    <div className="space-y-6 max-w-7xl mx-auto">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-[24px] font-bold tracking-tight text-[#0B0F19]">
            Ringkasan Operasional Toko
          </h1>
          <p className="text-[13px] text-[#555D6E] mt-0.5">
            Monitoring performa penjualan kasir, stok produk, dan kepatuhan syariah.
          </p>
        </div>

        <SegmentedControl
          size="sm"
          value={viewFilter}
          onChange={setViewFilter}
          options={[
            { value: "today", label: "Hari Ini" },
            { value: "week", label: "7 Hari" },
            { value: "month", label: "Bulan Ini" },
          ]}
        />
      </div>

      {/* Metric Cards with Cupertino Hairline Surfaces */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card className="rounded-[16px]">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-[12px] font-medium uppercase tracking-wider text-[#555D6E]">
              Kepatuhan Syariah
            </CardTitle>
            <ShieldCheck className="h-4 w-4 text-emerald-600" />
          </CardHeader>
          <CardContent>
            <div className="text-[26px] font-bold tracking-tight text-[#0B0F19] font-tabular">
              100%
            </div>
            <div className="flex items-center gap-2 mt-1.5">
              <Badge variant="success" dot className="text-[10px]">
                Sertifikat Halal Aktif
              </Badge>
            </div>
          </CardContent>
        </Card>

        <Card className="rounded-[16px]">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-[12px] font-medium uppercase tracking-wider text-[#555D6E]">
              Status Register POS
            </CardTitle>
            <ShoppingCart className="h-4 w-4 text-[#0066CC]" />
          </CardHeader>
          <CardContent>
            <div className="text-[26px] font-bold tracking-tight text-[#0B0F19]">
              Online & Siap
            </div>
            <p className="text-[11px] text-[#8B95A5] mt-1.5">
              Idempotency Engine Terkoneksi
            </p>
          </CardContent>
        </Card>

        <Card className="rounded-[16px]">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-[12px] font-medium uppercase tracking-wider text-[#555D6E]">
              Isolasi Tenant
            </CardTitle>
            <Store className="h-4 w-4 text-slate-700" />
          </CardHeader>
          <CardContent>
            <div className="text-[26px] font-bold tracking-tight text-[#0B0F19]">
              Aktif
            </div>
            <p className="text-[11px] text-[#8B95A5] mt-1.5">
              PostgreSQL Schema Scoped
            </p>
          </CardContent>
        </Card>

        <Card className="rounded-[16px]">
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-[12px] font-medium uppercase tracking-wider text-[#555D6E]">
              Integritas Ledger
            </CardTitle>
            <TrendingUp className="h-4 w-4 text-amber-600" />
          </CardHeader>
          <CardContent>
            <div className="text-[26px] font-bold tracking-tight text-[#0B0F19]">
              Seimbang
            </div>
            <div className="flex items-center gap-1 text-[11px] text-emerald-700 font-medium mt-1.5">
              <ArrowUpRight className="w-3.5 h-3.5" />
              <span>Total Debit = Total Kredit</span>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Operational Highlights Card */}
      <Card className="rounded-[18px]">
        <CardHeader>
          <CardTitle>Arsitektur Desain Timeless Industrial</CardTitle>
          <CardDescription>
            Sistem antarmuka ini dibangun menggunakan prinsip desain Cupertino: minimalis, presisi tinggi, bebas dari distraksi visual, dan siap mendukung produktivitas operasional kasir serta manajemen retail modern.
          </CardDescription>
        </CardHeader>
      </Card>
    </div>
  );
}
