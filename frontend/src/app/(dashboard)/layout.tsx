"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Header } from "@/components/layout/header";
import { Sidebar } from "@/components/layout/sidebar";
import { authApi, type UserProfile } from "@/lib/api";
import { Loader2 } from "lucide-react";
import { useTranslation } from "@/lib/i18n";

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const { t } = useTranslation();
  const [user, setUser] = React.useState<UserProfile | null>(null);
  const [isLoading, setIsLoading] = React.useState(true);
  const [isSidebarOpen, setIsSidebarOpen] = React.useState(false);
  const closeSidebar = React.useCallback(() => setIsSidebarOpen(false), []);

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

        <main className="min-w-0 flex-1 p-4 sm:p-6 xl:p-8">
          {children}
        </main>
      </div>
    </div>
  );
}
