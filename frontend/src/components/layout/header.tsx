"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { authApi, type UserProfile } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { LanguageSelector } from "@/components/ui/language-selector";
import { useTranslation } from "@/lib/i18n";
import { LogOut, Menu } from "lucide-react";

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
    <header className="relative z-30 flex w-full flex-wrap items-center justify-between gap-x-4 gap-y-2 border-b border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] px-4 py-2 sm:px-6">
      <div className="flex min-w-0 basis-full items-center gap-2 min-[480px]:basis-auto min-[480px]:flex-1">
        <button
          type="button"
          onClick={onToggleSidebar}
          aria-label={t("nav.openMenu")}
          aria-expanded={isSidebarOpen}
          aria-controls="mobile-navigation"
          className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl text-[var(--color-text-secondary)] focus-visible:outline focus-visible:outline-2 focus-visible:outline-[var(--color-action-primary)] xl:hidden"
        >
          <Menu className="h-5 w-5" aria-hidden="true" />
        </button>
        <div className="min-w-0">
          <p className="text-sm font-semibold text-[var(--color-text-primary)]">{t("nav.brand")}</p>
          <p className="break-all text-sm text-[var(--color-text-secondary)]">
            <bdi>{user?.tenant_slug || t("nav.unknownTenant")}</bdi>
          </p>
        </div>
      </div>
      <div className="flex max-w-full flex-wrap items-center gap-2">
        <LanguageSelector variant="select" />
        <Button
          variant="ghost"
          onClick={handleLogout}
          isLoading={isLoggingOut}
          aria-label={t("nav.logout")}
          className="h-12 min-w-12 text-sm"
        >
          <LogOut className="h-4 w-4" aria-hidden="true" />
          <span className="hidden sm:inline">{t("nav.logout")}</span>
        </Button>
      </div>
    </header>
  );
}
