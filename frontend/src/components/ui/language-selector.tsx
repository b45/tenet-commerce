"use client";

import * as React from "react";
import { useTranslation, LOCALES, type Locale } from "@/lib/i18n";
import { cn } from "@/lib/utils";

interface LanguageSelectorProps {
  className?: string;
  variant?: "segmented" | "compact";
}

const AVAILABLE_LOCALES: Locale[] = ["id", "en", "ar"];

export function LanguageSelector({ className, variant = "segmented" }: LanguageSelectorProps) {
  const { locale, setLocale } = useTranslation();

  if (variant === "compact") {
    return (
      <div className={cn("inline-flex items-center gap-1", className)}>
        {AVAILABLE_LOCALES.map((code) => {
          const cfg = LOCALES[code];
          const isActive = locale === code;
          return (
            <button
              key={code}
              type="button"
              onClick={() => setLocale(code)}
              aria-label={`Switch to ${cfg.name}`}
              className={cn(
                "px-2 py-1 text-xs rounded-lg transition-all font-medium flex items-center gap-1.5",
                isActive
                  ? "bg-[var(--color-bg-primary)] text-[var(--color-text-primary)] shadow-sm font-semibold border border-[var(--color-border-hairline)]"
                  : "text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-bg-subtle)]"
              )}
            >
              <span>{cfg.flag}</span>
              <span className="uppercase text-[10px] tracking-wider">{cfg.code}</span>
            </button>
          );
        })}
      </div>
    );
  }

  return (
    <div
      role="radiogroup"
      aria-label="Select interface language"
      className={cn(
        "inline-flex items-center p-0.5 rounded-xl bg-[var(--color-bg-tertiary)] border border-[var(--color-border-hairline)]",
        className
      )}
    >
      {AVAILABLE_LOCALES.map((code) => {
        const cfg = LOCALES[code];
        const isActive = locale === code;
        return (
          <button
            key={code}
            type="button"
            role="radio"
            aria-checked={isActive}
            onClick={() => setLocale(code)}
            className={cn(
              "px-2.5 py-1 text-xs rounded-lg transition-all flex items-center gap-1.5 cursor-pointer select-none",
              isActive
                ? "bg-[var(--color-bg-primary)] text-[var(--color-text-primary)] shadow-sm font-semibold border border-[var(--color-border-subtle)]"
                : "text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)]"
            )}
          >
            <span className="text-xs leading-none">{cfg.flag}</span>
            <span className="uppercase text-[11px] font-mono tracking-tight font-medium">
              {cfg.code}
            </span>
          </button>
        );
      })}
    </div>
  );
}
