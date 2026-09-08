"use client";

import * as React from "react";
import { apiClient } from "@/lib/api";
import type {
  Supplier,
  ComplianceCertificate,
  CreateSupplierInput,
  RegisterCertificateInput,
  ExpirySummary,
} from "../types";

export interface UseSuppliersReturn {
  suppliers: Supplier[];
  isLoading: boolean;
  error: string | null;
  searchQuery: string;
  setSearchQuery: (q: string) => void;
  statusFilter: "ALL" | "VALID" | "EXPIRING_SOON" | "EXPIRED" | "NO_CERT";
  setStatusFilter: (filter: "ALL" | "VALID" | "EXPIRING_SOON" | "EXPIRED" | "NO_CERT") => void;
  certTypeFilter: string;
  setCertTypeFilter: (type: string) => void;
  filteredSuppliers: Supplier[];
  expirySummary: ExpirySummary;
  refetch: () => Promise<void>;
  createSupplier: (data: CreateSupplierInput) => Promise<boolean>;
  registerCertificate: (supplierId: string, data: RegisterCertificateInput) => Promise<boolean>;
  revokeCertificate: (certId: string) => Promise<boolean>;
  getSupplierCertificates: (supplierId: string) => Promise<ComplianceCertificate[]>;
}

export function useSuppliers(): UseSuppliersReturn {
  const [suppliers, setSuppliers] = React.useState<Supplier[]>([]);
  const [isLoading, setIsLoading] = React.useState<boolean>(true);
  const [error, setError] = React.useState<string | null>(null);
  const [searchQuery, setSearchQuery] = React.useState<string>("");
  const [statusFilter, setStatusFilter] = React.useState<
    "ALL" | "VALID" | "EXPIRING_SOON" | "EXPIRED" | "NO_CERT"
  >("ALL");
  const [certTypeFilter, setCertTypeFilter] = React.useState<string>("ALL");

  const fetchSuppliers = React.useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const res = await apiClient.get<Supplier[]>("/supply-chain/suppliers");
      if (res.success && res.data) {
        setSuppliers(res.data);
      } else {
        setError(res.error?.message || "Gagal memuat data supplier");
      }
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Gagal memuat data supplier";
      setError(message);
    } finally {
      setIsLoading(false);
    }
  }, []);

  React.useEffect(() => {
    fetchSuppliers();
  }, [fetchSuppliers]);

  // Compute aggregated expiry summary
  const expirySummary = React.useMemo<ExpirySummary>(() => {
    let valid_count = 0;
    let expiring_soon_count = 0;
    let expired_count = 0;
    let missing_cert_count = 0;

    for (const s of suppliers) {
      const cert = s.compliance_certificate;
      if (!cert) {
        missing_cert_count++;
      } else if (cert.computed_status === "VALID") {
        valid_count++;
      } else if (cert.computed_status === "EXPIRING_SOON") {
        expiring_soon_count++;
      } else if (cert.computed_status === "EXPIRED" || cert.computed_status === "REVOKED") {
        expired_count++;
      }
    }

    return {
      total_active_suppliers: suppliers.length,
      valid_count,
      expiring_soon_count,
      expired_count,
      missing_cert_count,
    };
  }, [suppliers]);

  // Filter suppliers based on search query, statusFilter, and certTypeFilter
  const filteredSuppliers = React.useMemo(() => {
    return suppliers.filter((s) => {
      // 1. Status Filter
      if (statusFilter === "VALID" && s.compliance_certificate?.computed_status !== "VALID") {
        return false;
      }
      if (
        statusFilter === "EXPIRING_SOON" &&
        s.compliance_certificate?.computed_status !== "EXPIRING_SOON"
      ) {
        return false;
      }
      if (
        statusFilter === "EXPIRED" &&
        s.compliance_certificate?.computed_status !== "EXPIRED" &&
        s.compliance_certificate?.computed_status !== "REVOKED"
      ) {
        return false;
      }
      if (statusFilter === "NO_CERT" && !!s.compliance_certificate) {
        return false;
      }

      // 2. Certificate Type Filter (HALAL, BPOM, REGALKES, MICHELIN, etc.)
      if (certTypeFilter !== "ALL") {
        if (!s.compliance_certificate) return false;
        const currentType = s.compliance_certificate.cert_type?.toUpperCase() || "";
        if (certTypeFilter === "OTHER") {
          const standardPresets = ["HALAL", "BPOM", "REGALKES", "MICHELIN", "ISO_HACCP"];
          if (standardPresets.includes(currentType)) return false;
        } else if (currentType !== certTypeFilter.toUpperCase()) {
          return false;
        }
      }

      // 3. Search Query
      if (searchQuery.trim() !== "") {
        const q = searchQuery.toLowerCase();
        const matchName = s.company_name.toLowerCase().includes(q);
        const matchCode = s.code.toLowerCase().includes(q);
        const matchCertNum = s.compliance_certificate?.certificate_number
          ?.toLowerCase()
          .includes(q);
        const matchCertType = s.compliance_certificate?.cert_type
          ?.toLowerCase()
          .includes(q);
        const matchAuthority = s.compliance_certificate?.issuing_authority
          ?.toLowerCase()
          .includes(q);
        const matchScope = s.compliance_certificate?.scope
          ?.toLowerCase()
          .includes(q);
        return matchName || matchCode || matchCertNum || matchCertType || matchAuthority || matchScope;
      }

      return true;
    });
  }, [suppliers, searchQuery, statusFilter, certTypeFilter]);

  const createSupplier = React.useCallback(
    async (data: CreateSupplierInput): Promise<boolean> => {
      try {
        const res = await apiClient.post<Supplier>("/supply-chain/suppliers", data);
        if (res.success) {
          await fetchSuppliers();
          return true;
        }
        setError(res.error?.message || "Gagal mendaftarkan supplier");
        return false;
      } catch (err: unknown) {
        setError(err instanceof Error ? err.message : "Gagal mendaftarkan supplier");
        return false;
      }
    },
    [fetchSuppliers]
  );

  const registerCertificate = React.useCallback(
    async (supplierId: string, data: RegisterCertificateInput): Promise<boolean> => {
      try {
        const res = await apiClient.post<ComplianceCertificate>(
          `/supply-chain/suppliers/${supplierId}/certificates`,
          data
        );
        if (res.success) {
          await fetchSuppliers();
          return true;
        }
        setError(res.error?.message || "Gagal memperbarui sertifikat");
        return false;
      } catch (err: unknown) {
        setError(err instanceof Error ? err.message : "Gagal memperbarui sertifikat");
        return false;
      }
    },
    [fetchSuppliers]
  );

  const revokeCertificate = React.useCallback(
    async (certId: string): Promise<boolean> => {
      try {
        const res = await apiClient.put<void>(`/supply-chain/certificates/${certId}/revoke`);
        if (res.success) {
          await fetchSuppliers();
          return true;
        }
        setError(res.error?.message || "Gagal mencabut sertifikat");
        return false;
      } catch (err: unknown) {
        setError(err instanceof Error ? err.message : "Gagal mencabut sertifikat");
        return false;
      }
    },
    [fetchSuppliers]
  );

  const getSupplierCertificates = React.useCallback(
    async (supplierId: string): Promise<ComplianceCertificate[]> => {
      try {
        const res = await apiClient.get<ComplianceCertificate[]>(
          `/supply-chain/suppliers/${supplierId}/certificates`
        );
        if (res.success && res.data) {
          return res.data;
        }
        return [];
      } catch {
        return [];
      }
    },
    []
  );

  return {
    suppliers,
    isLoading,
    error,
    searchQuery,
    setSearchQuery,
    statusFilter,
    setStatusFilter,
    certTypeFilter,
    setCertTypeFilter,
    filteredSuppliers,
    expirySummary,
    refetch: fetchSuppliers,
    createSupplier,
    registerCertificate,
    revokeCertificate,
    getSupplierCertificates,
  };
}
