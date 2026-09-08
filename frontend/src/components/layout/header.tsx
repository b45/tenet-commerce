"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { authApi, type UserProfile } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { LanguageSelector } from "@/components/ui/language-selector";
import { useTranslation } from "@/lib/i18n";
import { useCapabilities } from "@/features/auth/hooks/use-capabilities";
import { LogOut, Menu } from "lucide-react";

interface HeaderProps {
  user: UserProfile | null;
  isSidebarOpen: boolean;
  onToggleSidebar: () => void;
}

export function Header({ user, isSidebarOpen, onToggleSidebar }: HeaderProps) {
  const router = useRouter();
  const { t } = useTranslation();
  const { capabilities } = useCapabilities();
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
    <header className="sticky top-0 z-30 grid w-full grid-cols-[minmax(0,1fr)_auto] items-center gap-2 border-b border-[var(--color-border-hairline)] bg-white/80 backdrop-blur-xl px-4 py-2 sm:px-6 shadow-[0_1px_2px_rgba(0,0,0,0.02)]">
      {/* Left: Menu toggle & Brand / Tenant identity */}
      <div className="flex items-center gap-2.5 sm:gap-3 min-w-0">
        <button
          type="button"
          onClick={onToggleSidebar}
          aria-label={t("nav.openMenu")}
          aria-expanded={isSidebarOpen}
          aria-controls="mobile-navigation"
          className="xl:hidden flex h-12 w-12 shrink-0 items-center justify-center rounded-lg text-[var(--color-text-secondary)] hover:bg-black/[0.05] focus-visible:outline focus-visible:outline-2 focus-visible:outline-[var(--color-action-primary)]"
        >
          <Menu className="h-5 w-5" aria-hidden="true" />
        </button>

        {/* Tenant Identity Pill matching Apple Cupertino standard */}
        <div className="flex items-center gap-1.5 sm:gap-2 text-xs sm:text-[13px] font-semibold tracking-tight text-[var(--color-text-primary)] min-w-0">
          <div className="w-6 h-6 rounded-[7px] bg-[#0B0F19] text-white flex items-center justify-center font-bold text-xs tracking-tighter shrink-0 shadow-xs">
            TC
          </div>
          <span title={user?.tenant_slug} className="min-w-0 truncate text-[var(--color-text-secondary)] font-medium">
            <bdi>{user?.tenant_slug || t("nav.sessionLoading")}</bdi>
          </span>
          {capabilities?.plan && (
            <Badge variant="outline" dot className="hidden sm:inline-flex text-[10px] py-0 px-2 h-5 font-medium border-emerald-600/25 text-emerald-700 bg-emerald-50/50">
              {capabilities.plan.name}
            </Badge>
          )}
        </div>
      </div>

      {/* Compact secondary controls leave the flexible column to tenant identity. */}
      <div className="flex min-w-0 items-center gap-1">
        <LanguageSelector />

        {/* Logout Button */}
        <Button
          variant="ghost"
          size="icon"
          onClick={handleLogout}
          isLoading={isLoggingOut}
          aria-label={t("nav.logout")}
          className="h-12 w-12 shrink-0 text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-black/[0.04] rounded-lg"
        >
          <LogOut className="h-3.5 w-3.5" aria-hidden="true" />
        </Button>
      </div>
    </header>
  );
}
