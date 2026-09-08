/**
 * Tenet Commerce — i18n Core Registry & String Interpolator
 */

import { id } from "./locales/id";
import { en } from "./locales/en";
import { ar } from "./locales/ar";
import type { Locale, LocaleConfig, TranslationSchema } from "./types";

export * from "./types";
export * from "./context";

export const LOCALES: Record<Locale, LocaleConfig> = {
  id: {
    code: "id",
    name: "Indonesian",
    nativeName: "Bahasa Indonesia",
    flag: "🇮🇩",
    direction: "ltr",
  },
  en: {
    code: "en",
    name: "English",
    nativeName: "English",
    flag: "🇬🇧",
    direction: "ltr",
  },
  ar: {
    code: "ar",
    name: "Arabic",
    nativeName: "العربية",
    flag: "🇸🇦",
    direction: "rtl",
  },
};

export const DEFAULT_LOCALE: Locale = "id";

export const DICTIONARIES: Record<Locale, TranslationSchema> = {
  id,
  en,
  ar,
};

/**
 * Resolves a dotted translation path (e.g. "pos.cart.title") and interpolates variables ({number}, {amount})
 */
export function translate(
  locale: Locale,
  path: string,
  params?: Record<string, string | number>
): string {
  const dict = DICTIONARIES[locale] || DICTIONARIES[DEFAULT_LOCALE];
  const keys = path.split(".");

  let current: unknown = dict;
  for (const k of keys) {
    if (current && typeof current === "object" && k in current) {
      current = (current as Record<string, unknown>)[k];
    } else {
      current = undefined;
      break;
    }
  }

  // Fallback to default locale (id) if missing in current locale
  if (typeof current !== "string") {
    let fallback: unknown = DICTIONARIES[DEFAULT_LOCALE];
    for (const k of keys) {
      if (fallback && typeof fallback === "object" && k in fallback) {
        fallback = (fallback as Record<string, unknown>)[k];
      } else {
        fallback = undefined;
        break;
      }
    }
    current = typeof fallback === "string" ? fallback : path;
  }

  let text = String(current);
  if (params) {
    for (const [key, value] of Object.entries(params)) {
      text = text.replace(new RegExp(`\\{${key}\\}`, "g"), String(value));
    }
  }

  return text;
}
