import * as React from "react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { ShieldAlert } from "lucide-react";

export default function ForbiddenPage() {
  return (
    <div className="min-h-screen flex flex-col items-center justify-center bg-[var(--color-surface-muted)] px-4 text-center">
      <div className="w-16 h-16 rounded-2xl bg-red-100 text-[var(--color-status-danger-text)] flex items-center justify-center mb-4 shadow-xs">
        <ShieldAlert className="w-8 h-8" />
      </div>
      <h1 className="text-3xl font-bold tracking-tight text-[var(--color-text-primary)]">
        403 — Akses Ditolak
      </h1>
      <p className="mt-2 text-sm text-[var(--color-text-secondary)] max-w-md">
        Akun Anda tidak memiliki izin RBAC yang diperlukan untuk mengakses modul ini. Silakan hubungi Store Manager atau Administrator toko Anda.
      </p>
      <div className="mt-6 flex gap-3">
        <Link href="/dashboard">
          <Button variant="primary">Kembali ke Dashboard</Button>
        </Link>
      </div>
    </div>
  );
}
