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
  onToggleSidebar?: () => void;
}

export function Header({ user, onToggleSidebar }: HeaderProps) {
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
    <header className="sticky top-0 z-30 flex h-14 w-full items-center justify-between border-b border-black/[0.06] bg-white/80 backdrop-blur-xl px-4 sm:px-6 shadow-[0_1px_2px_rgba(0,0,0,0.02)] transition-all">
      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={onToggleSidebar}
          aria-label="Buka Menu Navigasi"
          className="md:hidden p-1.5 rounded-[9px] text-[#555D6E] hover:bg-black/[0.05] focus:outline-none focus:ring-2 focus:ring-[#0066CC]/20"
        >
          <Menu className="h-5 w-5" />
        </button>

        {/* Tenant Identity Pill matching Cupertino sample */}
        <div className="flex items-center gap-1.5 sm:gap-2 text-xs sm:text-[13px] font-semibold tracking-tight text-[#0B0F19] min-w-0">
          <div className="w-6 h-6 rounded-[7px] bg-[#0B0F19] text-white flex items-center justify-center font-bold text-xs tracking-tighter shrink-0 shadow-xs">
            TC
          </div>
          <span className="hidden sm:inline">Tenet Commerce</span>
          <span className="text-black/30 font-normal hidden sm:inline">·</span>
          <span className="text-[#555D6E] font-medium truncate max-w-[110px] sm:max-w-none">{user?.tenant_slug || "al-barakah-mart"}</span>
        </div>
      </div>

      <div className="flex items-center gap-2 sm:gap-3">
        {/* Language Switcher */}
        <LanguageSelector variant="segmented" />

        {/* Network Online Status Pill (Apple Style) */}
        <div className="hidden sm:inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-black/[0.04] text-[11px] font-medium text-[#555D6E] border border-black/[0.04]">
          <Wifi className="w-3 h-3 text-emerald-600" />
          <span>{t("common.network.connected")}</span>
          <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse" />
        </div>

        {user && (
          <div className="hidden sm:flex items-center gap-2">
            <Badge variant="outline" className="text-[11px] font-medium border-black/[0.08] text-[#555D6E] gap-1.5">
              <User className="w-3 h-3 text-[#8B95A5]" />
              <span>{user.role}</span>
            </Badge>
          </div>
        )}

        <Button
          variant="ghost"
          size="sm"
          onClick={handleLogout}
          isLoading={isLoggingOut}
          className="text-xs h-8 text-[#555D6E] hover:text-[#0B0F19] gap-1.5"
        >
          <LogOut className="h-3.5 w-3.5" />
          <span className="hidden sm:inline font-medium">{t("nav.logout")}</span>
        </Button>
      </div>
    </header>
  );
}
