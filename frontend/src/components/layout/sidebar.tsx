"use client";

import * as React from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import type { UserProfile } from "@/lib/api";
import { useTranslation } from "@/lib/i18n";
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
  labelKey: string;
  label: string;
  href: string;
  icon: React.ComponentType<{ className?: string }>;
  permission?: string;
  roles?: string[];
}

const NAV_ITEMS: NavItem[] = [
  {
    labelKey: "nav.pos",
    label: "POS Kasir",
    href: "/pos",
    icon: ShoppingCart,
    permission: "pos:read",
  },
  {
    labelKey: "nav.orderHistory",
    label: "Riwayat Transaksi",
    href: "/pos/orders",
    icon: Receipt,
    permission: "pos:read",
  },
  {
    labelKey: "nav.inventory",
    label: "Inventori Produk",
    href: "/inventory",
    icon: Package,
    permission: "inventory:read",
  },
  {
    labelKey: "nav.certificates",
    label: "Sertifikasi Halal",
    href: "/supply-chain/certificates",
    icon: ShieldCheck,
    permission: "supply_chain:manage",
  },
  {
    labelKey: "nav.procurement",
    label: "Pengadaan (PO & GR)",
    href: "/supply-chain/po",
    icon: Truck,
    permission: "supply_chain:manage",
  },
  {
    labelKey: "nav.ledger",
    label: "Buku Besar (Ledger)",
    href: "/ledger/entries",
    icon: BookOpen,
    permission: "ledger:read",
  },
  {
    labelKey: "nav.dashboard",
    label: "Manager Dashboard",
    href: "/dashboard",
    icon: LayoutDashboard,
    roles: ["MANAGER", "SUPER_ADMIN"],
  },
];

export function Sidebar({ user, isOpen, onClose }: SidebarProps) {
  const pathname = usePathname();
  const { t } = useTranslation();
  const dialogRef = React.useRef<HTMLDialogElement>(null);
  const desktopRef = React.useRef<HTMLElement>(null);

  React.useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;
    // Keep this query aligned with the shell's Tailwind xl breakpoint.
    const wide = window.matchMedia("(min-width: 1280px)");
    const previousOverflow = document.body.style.overflow;
    const closeAtDesktop = () => {
      if (wide.matches && dialog.open) {
        dialog.close();
        onClose();
        desktopRef.current?.focus();
      }
    };
    if (isOpen && !wide.matches) {
      dialog.showModal();
      document.body.style.overflow = "hidden";
    } else if (dialog.open) {
      dialog.close();
    }
    wide.addEventListener("change", closeAtDesktop);
    return () => {
      wide.removeEventListener("change", closeAtDesktop);
      document.body.style.overflow = previousOverflow;
    };
  }, [isOpen, onClose]);

  const filteredNav = NAV_ITEMS.filter(item => {
    if (!user) return false;
    if (item.roles && !item.roles.includes(user.role)) return false;
    return !item.permission || !!user.permissions?.includes(item.permission);
  });

  const content = (mobile: boolean) => (
    <>
      <div className="flex min-h-16 shrink-0 items-center justify-between gap-2 border-b border-[var(--color-border-hairline)] px-4 py-2">
        <span className="text-base font-semibold">{t("nav.brand")}</span>
        {mobile && (
          <button type="button" onClick={onClose} aria-label={t("nav.closeMenu")}
            className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl focus-visible:outline focus-visible:outline-2 focus-visible:outline-[var(--color-action-primary)]">
            <X className="h-5 w-5" aria-hidden="true" />
          </button>
        )}
      </div>
      <div className="px-4 py-3 text-sm text-[var(--color-text-secondary)]">
        <p className="break-all"><bdi>{user?.tenant_slug || t("nav.unknownTenant")}</bdi></p>
        {user && <p className="mt-1 break-words"><bdi>{user.role}</bdi></p>}
      </div>
      <nav className="min-h-0 flex-1 space-y-1 overflow-y-auto px-3 pb-4" aria-label={t("nav.navigation")}>
        {filteredNav.map(item => {
          const active = pathname === item.href || pathname.startsWith(`${item.href}/`);
          const Icon = item.icon;
          return (
            <Link key={item.href} href={item.href} onClick={onClose}
              aria-current={active ? "page" : undefined}
              className={cn(
                "flex min-h-12 items-center gap-3 rounded-xl px-3 py-3 text-sm font-medium focus-visible:outline focus-visible:outline-2 focus-visible:outline-[var(--color-action-primary)]",
                active ? "bg-[var(--color-surface-base)] text-[var(--color-action-primary)]" : "text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-base)]"
              )}>
              <Icon className="h-5 w-5 shrink-0" />
              <span className="min-w-0 break-words">{t(item.labelKey)}</span>
            </Link>
          );
        })}
      </nav>
    </>
  );

  return (
    <>
      <aside ref={desktopRef} tabIndex={-1} aria-label={t("nav.navigation")}
        className="sticky top-0 hidden h-dvh w-60 shrink-0 flex-col border-e border-[var(--color-border-hairline)] bg-[var(--color-surface-muted)] text-[var(--color-text-primary)] xl:flex">
        {content(false)}
      </aside>
      <dialog ref={dialogRef} id="mobile-navigation" aria-label={t("nav.navigation")}
        onCancel={event => { event.preventDefault(); onClose(); }}
        onClose={onClose}
        onClick={event => { if (event.target === event.currentTarget) onClose(); }}
        className="fixed inset-y-0 start-0 end-auto m-0 h-dvh max-h-none w-[min(20rem,100%)] max-w-full border-0 bg-[var(--color-surface-muted)] p-0 text-[var(--color-text-primary)] backdrop:bg-black/40 xl:hidden">
        <div className="flex h-full flex-col">{content(true)}</div>
      </dialog>
    </>
  );
}
