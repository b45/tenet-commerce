"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Header } from "@/components/layout/header";
import { Sidebar } from "@/components/layout/sidebar";
import { authApi, type UserProfile } from "@/lib/api";
import { CapabilitiesProvider } from "@/features/auth/hooks/use-capabilities";
import { AlertTriangle, Loader2, RefreshCw } from "lucide-react";
import { useTranslation } from "@/lib/i18n";
import { clearAllOfflineState } from "@/lib/offline/db";

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const { t } = useTranslation();
  const [user, setUser] = React.useState<UserProfile | null>(null);
  const [isLoading, setIsLoading] = React.useState(true);
  const [isSidebarOpen, setIsSidebarOpen] = React.useState(false);
  const [maintenance, setMaintenance] = React.useState(false);
  const closeSidebar = React.useCallback(() => setIsSidebarOpen(false), []);

  React.useEffect(() => {
    const onMaintenance = () => setMaintenance(true);
    window.addEventListener("tenet:maintenance", onMaintenance);
    return () => window.removeEventListener("tenet:maintenance", onMaintenance);
  }, []);

  React.useEffect(() => {
    let isMounted = true;

    async function loadSession() {
      try {
        const profile = await authApi.me();
        if (isMounted) {
          setUser(profile);
          setIsLoading(false);
        }
      } catch {
        if (isMounted) {
          await clearAllOfflineState();
          router.push("/login");
        }
      }
    }

    loadSession();
    return () => {
      isMounted = false;
    };
  }, [router]);

  if (isLoading) {
    return (
      <div className="min-h-screen flex flex-col items-center justify-center bg-[var(--color-surface-muted)] gap-3">
        <Loader2 className="h-8 w-8 animate-spin text-[var(--color-action-primary)]" />
        <p className="text-sm text-[var(--color-text-secondary)] font-medium">
          {t("nav.sessionLoading")}
        </p>
      </div>
    );
  }

  return (
    <CapabilitiesProvider initialUser={user}>
      <div className="flex min-h-screen bg-[var(--color-surface-muted)]">
        <Sidebar
          user={user}
          isOpen={isSidebarOpen}
          onClose={closeSidebar}
        />

        <div className="flex flex-1 flex-col min-w-0">
          <Header
            user={user}
            isSidebarOpen={isSidebarOpen}
            onToggleSidebar={() => setIsSidebarOpen((prev) => !prev)}
          />

          {maintenance && (
            <div className="border-b border-amber-200 bg-amber-50 px-4 py-3 text-amber-950 sm:px-6 xl:px-8" role="status">
              <div className="mx-auto flex max-w-7xl items-start gap-3">
                <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0 text-amber-600" aria-hidden="true" />
                <div className="min-w-0 flex-1">
                  <p className="font-semibold">{t("common.maintenance.title")}</p>
                  <p className="text-sm text-amber-800">{t("common.maintenance.message")}</p>
                </div>
                <button type="button" onClick={() => window.location.reload()} className="inline-flex shrink-0 items-center gap-2 rounded-md border border-amber-300 px-3 py-1.5 text-sm font-medium hover:bg-amber-100">
                  <RefreshCw className="h-4 w-4" aria-hidden="true" />
                  {t("common.maintenance.retry")}
                </button>
              </div>
            </div>
          )}

          <main className="min-w-0 flex-1 p-4 sm:p-6 xl:p-8">
            {children}
          </main>
        </div>
      </div>
    </CapabilitiesProvider>
  );
}
