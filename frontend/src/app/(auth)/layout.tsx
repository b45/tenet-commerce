import * as React from "react";
import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Masuk — Tenet Commerce",
  description: "Autentikasi POS dan Sistem Operasional Retail Tenet Commerce",
};

export default function AuthLayout({ children }: { children: React.ReactNode }) {
  return (
    <main className="min-h-screen flex items-center justify-center bg-slate-50 px-4 py-12 sm:px-6 lg:px-8">
      <div className="w-full max-w-md space-y-8">
        {children}
      </div>
    </main>
  );
}
