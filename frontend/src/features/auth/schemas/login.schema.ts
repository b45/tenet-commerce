import { z } from "zod";

export const loginSchema = z.object({
  tenant_slug: z
    .string()
    .trim()
    .min(1, "Tenant slug toko wajib diisi")
    .regex(/^[a-zA-Z0-9_-]+$/, "Tenant slug hanya boleh berisi huruf, angka, strip (-), atau underscore (_)"),
  email: z
    .string()
    .trim()
    .min(1, "Email wajib diisi")
    .email("Format email tidak valid"),
  password: z
    .string()
    .min(1, "Kata sandi wajib diisi")
    .min(6, "Kata sandi minimal 6 karakter"),
});

export type LoginInput = z.infer<typeof loginSchema>;
