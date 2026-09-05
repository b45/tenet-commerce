"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { authApi, type UserProfile, ApiError } from "@/lib/api";
import { loginSchema, type LoginInput } from "../schemas/login.schema";

interface UseAuthReturn {
  user: UserProfile | null;
  isLoading: boolean;
  error: string | null;
  traceId: string | null;
  login: (input: LoginInput) => Promise<void>;
  logout: () => Promise<void>;
  hasPermission: (permission: string) => boolean;
  hasRole: (role: string) => boolean;
}

export function useAuth(): UseAuthReturn {
  const router = useRouter();
  const [user, setUser] = React.useState<UserProfile | null>(null);
  const [isLoading, setIsLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);
  const [traceId, setTraceId] = React.useState<string | null>(null);

  React.useEffect(() => {
    let isMounted = true;
    async function init() {
      try {
        const profile = await authApi.me();
        if (isMounted) {
          setUser(profile);
          setError(null);
        }
      } catch {
        if (isMounted) {
          setUser(null);
        }
      } finally {
        if (isMounted) setIsLoading(false);
      }
    }
    init();
    return () => {
      isMounted = false;
    };
  }, []);

  const login = async (input: LoginInput) => {
    setError(null);
    setIsLoading(true);

    // 1. Runtime schema validation using Zod
    const validation = loginSchema.safeParse(input);
    if (!validation.success) {
      const firstError = validation.error.issues[0]?.message || "Validasi gagal";
      setError(firstError);
      setIsLoading(false);
      throw new Error(firstError);
    }

    // 2. Execute login request via BFF
    try {
      const profile = await authApi.login(validation.data);
      setUser(profile);

      // 3. Deterministic role redirection
      if (profile.role === "CASHIER") {
        router.push("/pos");
      } else if (profile.role === "FINANCIAL_ADMIN") {
        router.push("/ledger/entries");
      } else if (profile.role === "COMPLIANCE_OFFICER") {
        router.push("/supply-chain/certificates");
      } else {
        router.push("/dashboard");
      }
      router.refresh();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Autentikasi gagal. Periksa kredensial Anda.";
      setError(msg);
      if (err instanceof ApiError && err.traceId) {
        setTraceId(err.traceId);
      }
      throw err;
    } finally {
      setIsLoading(false);
    }
  };

  const logout = async () => {
    setIsLoading(true);
    try {
      await authApi.logout();
      setUser(null);
      setTraceId(null);
      router.push("/login");
      router.refresh();
    } finally {
      setIsLoading(false);
    }
  };

  const hasPermission = (permission: string): boolean => {
    if (!user) return false;
    if (user.role === "SUPER_ADMIN") return true;
    return user.permissions?.includes(permission) ?? false;
  };

  const hasRole = (role: string): boolean => {
    if (!user) return false;
    return user.role === role;
  };

  return {
    user,
    isLoading,
    error,
    traceId,
    login,
    logout,
    hasPermission,
    hasRole,
  };
}
