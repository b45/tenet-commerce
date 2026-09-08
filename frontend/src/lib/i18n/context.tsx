"use client";

import * as React from "react";
import {
  DEFAULT_LOCALE,
  LOCALES,
  translate,
  type Direction,
  type Locale,
  type LocaleConfig,
} from "./index";

interface I18nContextValue {
  locale: Locale;
  direction: Direction;
  config: LocaleConfig;
  setLocale: (nextLocale: Locale) => void;
  t: (path: string, params?: Record<string, string | number>) => string;
}

const I18nContext = React.createContext<I18nContextValue | null>(null);

const STORAGE_KEY = "tenet_locale";
const COOKIE_NAME = "tenet_locale";

function getInitialLocale(): Locale {
  if (typeof window === "undefined") return DEFAULT_LOCALE;

  // 1. Try cookie first (synced with SSR)
  const match = document.cookie.match(new RegExp(`(?:^|; )${COOKIE_NAME}=([^;]*)`));
  if (match && match[1] && (match[1] === "id" || match[1] === "en" || match[1] === "ar")) {
    return match[1] as Locale;
  }

  // 2. Try localStorage
  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved && (saved === "id" || saved === "en" || saved === "ar")) {
      return saved as Locale;
    }
  } catch {
    // ignore
  }

  return DEFAULT_LOCALE;
}

export function I18nProvider({ children }: { children: React.ReactNode }) {
  const [locale, setLocaleState] = React.useState<Locale>(DEFAULT_LOCALE);
  const [isInitialized, setIsInitialized] = React.useState<boolean>(false);

  React.useEffect(() => {
    const initial = getInitialLocale();
    setLocaleState(initial);
    setIsInitialized(true);
  }, []);

  const direction = LOCALES[locale]?.direction || "ltr";

  // Sync HTML tag attributes when locale or direction changes
  React.useEffect(() => {
    if (!isInitialized) return;

    document.documentElement.lang = locale;
    document.documentElement.dir = direction;

    // Set cookie for Next.js SSR / BFF requests
    document.cookie = `${COOKIE_NAME}=${locale}; path=/; max-age=31536000; SameSite=Lax`;

    try {
      localStorage.setItem(STORAGE_KEY, locale);
    } catch {
      // ignore
    }
  }, [locale, direction, isInitialized]);

  const setLocale = React.useCallback((next: Locale) => {
    if (next !== "id" && next !== "en" && next !== "ar") return;
    setLocaleState(next);
  }, []);

  const t = React.useCallback(
    (path: string, params?: Record<string, string | number>) => {
      return translate(locale, path, params);
    },
    [locale]
  );

  const value = React.useMemo<I18nContextValue>(() => ({
    locale,
    direction,
    config: LOCALES[locale],
    setLocale,
    t,
  }), [locale, direction, setLocale, t]);

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useTranslation() {
  const ctx = React.useContext(I18nContext);
  if (!ctx) {
    // Graceful fallback for components rendered outside provider
    return {
      locale: DEFAULT_LOCALE,
      direction: "ltr" as Direction,
      config: LOCALES[DEFAULT_LOCALE],
      setLocale: () => {},
      t: (path: string, params?: Record<string, string | number>) => translate(DEFAULT_LOCALE, path, params),
    };
  }
  return ctx;
}
