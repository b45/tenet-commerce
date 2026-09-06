"use client";

import { useTranslation, LOCALES, type Locale } from "@/lib/i18n";
import { cn } from "@/lib/utils";

const AVAILABLE_LOCALES: Locale[] = ["id", "en", "ar"];

export function LanguageSelector({ className }: { className?: string }) {
  const { locale, setLocale, t } = useTranslation();
  return (
    <select
      aria-label={t("nav.language")}
      value={locale}
      onChange={event => {
        const next = event.target.value;
        if (AVAILABLE_LOCALES.includes(next as Locale)) setLocale(next as Locale);
      }}
      className={cn("h-11 w-20 min-w-0 shrink-0 rounded-lg border border-transparent bg-transparent px-2 text-base text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-muted)] focus-visible:outline focus-visible:outline-2 focus-visible:outline-[var(--color-action-primary)]", className)}
    >
      {AVAILABLE_LOCALES.map(code => (
        <option key={code} value={code} lang={code} dir="ltr" aria-label={LOCALES[code].nativeName}>
          {code.toUpperCase()}
        </option>
      ))}
    </select>
  );
}
