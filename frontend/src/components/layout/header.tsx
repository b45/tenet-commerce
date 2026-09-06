"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { authApi, type UserProfile } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { LanguageSelector } from "@/components/ui/language-selector";
import { useTranslation } from "@/lib/i18n";
import { LogOut, User, Menu, Wifi } from "lucide-react";

interface HeaderProps {
  user: UserProfile | null;
  isSidebarOpen: boolean;
  onToggleSidebar: () => void;
}

export function Header({ user, isSidebarOpen, onToggleSidebar }: HeaderProps) {
  const router = useRouter();
  const { t } = useTranslation();
  const [isLoggingOut, setIsLoggingOut] = React.useState(false);

  const handleLogout = async () => {
    setIsLoggingOut(true);
    try {
      await authApi.logout();
      router.push("/login");
      router.refresh();
    } catch {
      router.push("/login");
    } finally {
      setIsLoggingOut(false);
    }
  };

  return (
    <header className="sticky top-0 z-30 flex h-14 w-full items-center justify-between border-b border-[var(--color-border-hairline)] bg-white/80 backdrop-blur-xl px-4 sm:px-6 shadow-[0_1px_2px_rgba(0,0,0,0.02)] transition-all">
      {/* Left: Menu toggle & Brand / Tenant identity */}
      <div className="flex items-center gap-2.5 sm:gap-3 min-w-0">
        <button
          type="button"
          onClick={onToggleSidebar}
          aria-label={t("nav.openMenu")}
          aria-expanded={isSidebarOpen}
          aria-controls="mobile-navigation"
          className="xl:hidden p-1.5 rounded-lg text-[var(--color-text-secondary)] hover:bg-black/[0.05] focus:outline-none focus:ring-2 focus:ring-[var(--color-action-primary)]/20"
        >
          <Menu className="h-5 w-5" aria-hidden="true" />
        </button>

        {/* Tenant Identity Pill matching Apple Cupertino standard */}
        <div className="flex items-center gap-1.5 sm:gap-2 text-xs sm:text-[13px] font-semibold tracking-tight text-[var(--color-text-primary)] min-w-0">
          <div className="w-6 h-6 rounded-[7px] bg-[#0B0F19] text-white flex items-center justify-center font-bold text-xs tracking-tighter shrink-0 shadow-xs">
            TC
          </div>
          <span className="hidden sm:inline font-semibold">{t("nav.brand")}</span>
          <span className="text-black/30 font-normal hidden sm:inline">·</span>
          <span className="text-[var(--color-text-secondary)] font-medium truncate max-w-[120px] sm:max-w-none">
            <bdi>{user?.tenant_slug || "al-barakah-mart"}</bdi>
          </span>
        </div>
      </div>

      {/* Right: Language Segmented Control, Network Status, Role Badge, Logout */}
      <div className="flex items-center gap-2 sm:gap-3">
        {/* Apple Segmented Language Switcher (ID | EN | AR) */}
        <LanguageSelector variant="segmented" />

        {/* Network Online Status Pill */}
        <div className="hidden md:inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-black/[0.04] text-[11px] font-medium text-[var(--color-text-secondary)] border border-black/[0.04]">
          <Wifi className="w-3 h-3 text-emerald-600" aria-hidden="true" />
          <span>{t("common.network.connected")}</span>
          <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse" />
        </div>

        {/* User Role Badge */}
        {user && (
          <div className="hidden sm:flex items-center gap-2">
            <Badge variant="outline" className="text-[11px] font-medium border-black/[0.08] text-[var(--color-text-secondary)] gap-1.5">
              <User className="w-3 h-3 text-[var(--color-text-muted)]" />
              <span>{user.role}</span>
            </Badge>
          </div>
        )}

        {/* Logout Button */}
        <Button
          variant="ghost"
          size="sm"
          onClick={handleLogout}
          isLoading={isLoggingOut}
          aria-label={t("nav.logout")}
          className="text-xs h-8 text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-black/[0.04] gap-1.5 px-2.5 rounded-lg"
        >
          <LogOut className="h-3.5 w-3.5" aria-hidden="true" />
          <span className="hidden sm:inline font-medium">{t("nav.logout")}</span>
        </Button>
      </div>
    </header>
  );
}
