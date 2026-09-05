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

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setFieldErrors({});

    try {
      await login({
        tenant_slug: tenantSlug.trim(),
        email: email.trim(),
        password,
      });
    } catch {
      // Handled via useAuth error state
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4" noValidate>
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
          autoComplete="organization"
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
          name="email"
          type="email"
          placeholder={t("auth.emailPlaceholder")}
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          disabled={isLoading}
          autoComplete="email"
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
