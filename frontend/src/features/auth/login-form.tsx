"use client";

import * as React from "react";
import { useAuth } from "./hooks/use-auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Alert } from "@/components/ui/alert";
import { ShieldCheck, Lock, Mail, Building2, Server } from "lucide-react";

import { HelpdeskDiagnostic } from "@/components/ui/helpdesk-diagnostic";
import { useTranslation } from "@/lib/i18n";

export function LoginForm() {
  const { t } = useTranslation();
  const { login, isLoading, error, traceId } = useAuth();
  const [tenantSlug, setTenantSlug] = React.useState("");
  const [email, setEmail] = React.useState("");
  const [password, setPassword] = React.useState("");
  const [fieldErrors, setFieldErrors] = React.useState<{ [key: string]: string }>({});

  // Prefill tenant_slug from cookie / localStorage or fallback
  React.useEffect(() => {
    if (typeof window !== "undefined") {
      // 1. Check cookies for tenet_tenant_slug
      const cookieMatch = document.cookie
        .split("; ")
        .find((row) => row.startsWith("tenet_tenant_slug="));
      const savedCookieTenant = cookieMatch ? decodeURIComponent(cookieMatch.split("=")[1]) : null;

      // 2. Check localStorage
      const localTenant = localStorage.getItem("tenet_last_tenant_slug");

      const tenantToUse = savedCookieTenant || localTenant || "";
      if (tenantToUse) {
        setTenantSlug(tenantToUse);
      }
    }
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setFieldErrors({});

    const trimmedTenant = tenantSlug.trim();
    if (typeof window !== "undefined" && trimmedTenant) {
      localStorage.setItem("tenet_last_tenant_slug", trimmedTenant);
    }

    try {
      await login({
        tenant_slug: trimmedTenant,
        email: email.trim(),
        password,
      });
    } catch {
      // Handled via useAuth error state
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4" noValidate method="POST">
      {error && (
        <Alert variant="destructive" title={t("auth.errorTitle")}>
          <div>{error}</div>
          <HelpdeskDiagnostic traceId={traceId} errorMessage={error} errorCode="AUTH_FAILED" />
        </Alert>
      )}

      <div className="space-y-1.5">
        <label
          htmlFor="tenant_slug"
          className="text-[11px] font-semibold uppercase tracking-wider text-[#555D6E] flex items-center justify-between"
        >
          <span className="flex items-center gap-1.5">
            <Building2 className="w-3.5 h-3.5 text-[#8B95A5]" />
            {t("auth.tenantSlug")}
          </span>
          <span className="text-[10px] text-[#8B95A5] normal-case font-normal">Identitas Schema</span>
        </label>
        <Input
          id="tenant_slug"
          name="tenant_slug"
          type="text"
          placeholder={t("auth.tenantSlugPlaceholder")}
          value={tenantSlug}
          onChange={(e) => setTenantSlug(e.target.value)}
          disabled={isLoading}
          autoComplete="off"
          autoCorrect="off"
          autoCapitalize="none"
          spellCheck={false}
          data-1p-ignore="true"
          data-lpignore="true"
          required
          error={!!fieldErrors.tenant_slug}
        />
      </div>

      <div className="space-y-1.5">
        <label
          htmlFor="email"
          className="text-[11px] font-semibold uppercase tracking-wider text-[#555D6E] flex items-center gap-1.5"
        >
          <Mail className="w-3.5 h-3.5 text-[#8B95A5]" />
          {t("auth.email")}
        </label>
        <Input
          id="email"
          name="username"
          type="email"
          placeholder={t("auth.emailPlaceholder")}
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          disabled={isLoading}
          autoComplete="username"
          autoCapitalize="none"
          autoCorrect="off"
          spellCheck={false}
          required
          error={!!fieldErrors.email}
        />
      </div>

      <div className="space-y-1.5">
        <label
          htmlFor="password"
          className="text-[11px] font-semibold uppercase tracking-wider text-[#555D6E] flex items-center gap-1.5"
        >
          <Lock className="w-3.5 h-3.5 text-[#8B95A5]" />
          {t("auth.password")}
        </label>
        <Input
          id="password"
          name="password"
          type="password"
          placeholder={t("auth.passwordPlaceholder")}
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          disabled={isLoading}
          autoComplete="current-password"
          required
          error={!!fieldErrors.password}
        />
      </div>

      <div className="pt-2">
        <Button
          type="submit"
          variant="primary"
          size="lg"
          className="w-full text-[14px]"
          isLoading={isLoading}
        >
          {isLoading ? t("auth.submitting") : t("auth.submit")}
        </Button>
      </div>

      {/* Trust & Security Metadata */}
      <div className="pt-4 border-t border-black/[0.06] flex items-center justify-center gap-4 text-[11px] text-[#8B95A5] select-none">
        <span className="inline-flex items-center gap-1">
          <ShieldCheck className="w-3.5 h-3.5 text-emerald-600" />
          httpOnly Secure Session
        </span>
        <span className="inline-flex items-center gap-1">
          <Server className="w-3.5 h-3.5 text-[#0066CC]" />
          Multi-Tenant Isolated
        </span>
      </div>
    </form>
  );
}
