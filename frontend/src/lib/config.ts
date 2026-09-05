import { z } from "zod";

/**
 * Enterprise Frontend Configuration
 * Strictly validates environment variables at runtime using Zod.
 */
const envSchema = z.object({
  BACKEND_API_URL: z.string().url().default("http://localhost:8081"),
  NEXT_PUBLIC_APP_URL: z.string().url().default("http://localhost:3000"),
  NEXT_PUBLIC_APP_NAME: z.string().default("Tenet Commerce"),
  NEXT_PUBLIC_DEFAULT_TENANT_SLUG: z.string().default("b45-bakery"),
  NODE_ENV: z.enum(["development", "test", "production"]).default("development"),
});

const parsed = envSchema.safeParse({
  BACKEND_API_URL: process.env.BACKEND_API_URL,
  NEXT_PUBLIC_APP_URL: process.env.NEXT_PUBLIC_APP_URL,
  NEXT_PUBLIC_APP_NAME: process.env.NEXT_PUBLIC_APP_NAME,
  NEXT_PUBLIC_DEFAULT_TENANT_SLUG: process.env.NEXT_PUBLIC_DEFAULT_TENANT_SLUG,
  NODE_ENV: process.env.NODE_ENV,
});

if (!parsed.success) {
  // eslint-disable-next-line no-console
  console.error("❌ Invalid environment variables configuration:", parsed.error.format());
  throw new Error("Invalid frontend environment configuration");
}

export const env = parsed.data;

/**
 * Returns clean backend API URL without trailing slash
 */
export function getBackendUrl(): string {
  return env.BACKEND_API_URL.replace(/\/$/, "");
}
