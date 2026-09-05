"use client";

import * as React from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import type { UserProfile } from "@/lib/api";
import {
  ShoppingCart,
  Receipt,
  Package,
  ShieldCheck,
  Truck,
  BookOpen,
  LayoutDashboard,
  X,
} from "lucide-react";

interface SidebarProps {
  user: UserProfile | null;
  isOpen: boolean;
  onClose: () => void;
}

interface NavItem {
  label: string;
  href: string;
  icon: React.ComponentType<{ className?: string }>;
  permission?: string;
  roles?: string[];
}

const NAV_ITEMS: NavItem[] = [
  {
    label: "POS Kasir",
    href: "/pos",
    icon: ShoppingCart,
    permission: "pos:read",
  },
  {
    label: "Riwayat Transaksi",
    href: "/pos/orders",
    icon: Receipt,
    permission: "pos:read",
  },
  {
    label: "Inventori Produk",
    href: "/inventory",
    icon: Package,
    permission: "inventory:read",
  },
  {
    label: "Sertifikasi Halal",
    href: "/supply-chain/certificates",
    icon: ShieldCheck,
    permission: "supply_chain:manage",
  },
  {
    label: "Pengadaan (PO & GR)",
    href: "/supply-chain/po",
    icon: Truck,
    permission: "supply_chain:manage",
  },
  {
    label: "Buku Besar (Ledger)",
    href: "/ledger/entries",
    icon: BookOpen,
    permission: "ledger:read",
  },
  {
    label: "Manager Dashboard",
    href: "/dashboard",
    icon: LayoutDashboard,
    roles: ["MANAGER", "SUPER_ADMIN"],
  },
];

export function Sidebar({ user, isOpen, onClose }: SidebarProps) {
  const pathname = usePathname();

  const filteredNav = NAV_ITEMS.filter((item) => {
    if (!user) return true;
    if (item.roles && !item.roles.includes(user.role)) {
      return false;
    }
    if (item.permission && !user.permissions?.includes(item.permission)) {
      return false;
    }
    return true;
  });

  return (
    <>
      {/* Mobile Backdrop */}
      {isOpen && (
        <div
          className="fixed inset-0 z-40 bg-slate-900/30 backdrop-blur-xs md:hidden"
          onClick={onClose}
          aria-hidden="true"
        />
      )}

      {/* Sidebar Navigation */}
      <aside
        className={cn(
          "fixed inset-y-0 left-0 z-50 flex w-60 flex-col border-r border-black/[0.06] bg-[#F5F5F7] transition-transform duration-200 ease-[cubic-bezier(0.16,1,0.3,1)] md:static md:translate-x-0",
          isOpen ? "translate-x-0" : "-translate-x-full"
        )}
      >
        <div className="flex h-14 items-center justify-between px-5 border-b border-black/[0.06]">
          <Link href="/dashboard" className="flex items-center gap-2.5">
            <div className="w-7 h-7 rounded-[8px] bg-[#0B0F19] text-white flex items-center justify-center font-bold text-xs tracking-tighter shadow-xs">
              TC
            </div>
            <span className="font-bold tracking-tight text-[15px] text-[#0B0F19]">
              Tenet Commerce
            </span>
          </Link>

          <button
            type="button"
            onClick={onClose}
            aria-label="Tutup Menu"
            className="md:hidden p-1.5 rounded-[8px] text-[#555D6E] hover:bg-black/[0.05]"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <nav className="flex-1 space-y-1 px-3 py-3.5 overflow-y-auto" aria-label="Navigasi Utama">
          {filteredNav.map((item) => {
            const isActive = pathname === item.href || pathname.startsWith(`${item.href}/`);
            const Icon = item.icon;

            return (
              <Link
                key={item.href}
                href={item.href}
                onClick={onClose}
                className={cn(
                  "flex items-center gap-2.5 rounded-[10px] px-3 py-2 text-[13px] font-medium tracking-tight transition-all duration-150 select-none",
                  isActive
                    ? "bg-white text-[#0B0F19] font-semibold shadow-[0_1px_3px_rgba(0,0,0,0.06),0_1px_1px_rgba(0,0,0,0.04)] border border-black/[0.04]"
                    : "text-[#555D6E] hover:text-[#0B0F19] hover:bg-black/[0.04]"
                )}
              >
                <Icon
                  className={cn(
                    "h-4 w-4",
                    isActive ? "text-[#0066CC]" : "text-[#8B95A5]"
                  )}
                />
                <span>{item.label}</span>
              </Link>
            );
          })}
        </nav>

        <div className="p-3.5 border-t border-black/[0.06] text-[11px] text-[#8B95A5] text-center font-medium">
          v0.3.0 · Cupertino Industrial UI
        </div>
      </aside>
    </>
  );
}
