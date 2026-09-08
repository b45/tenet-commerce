/**
 * Tenet Commerce — Multi-Standard Compliance & Certification Types
 * Supports: Halal (BPJPH/MUI), BPOM, Regalkes (Kemenkes), Michelin Guide, ISO/HACCP, and Custom Standards.
 * Aligned with backend models in backend/internal/supplychain/models.go
 */

export type CertificateStatus = "VALID" | "EXPIRING_SOON" | "EXPIRED" | "REVOKED";

export type StandardCertType =
  | "HALAL"
  | "BPOM"
  | "REGALKES"
  | "MICHELIN"
  | "ISO_HACCP"
  | "OTHER";

export interface CertTypeMetadata {
  id: StandardCertType;
  labelKey: string;
  defaultAuthority: string;
  defaultScope: string;
  badgeBg: string;
  badgeText: string;
  badgeBorder: string;
}

export const CERT_TYPE_PRESETS: CertTypeMetadata[] = [
  {
    id: "HALAL",
    labelKey: "supplyChain.certTypes.halal",
    defaultAuthority: "BPJPH Kemenag / LPPOM-MUI",
    defaultScope: "Bahan Baku & Produk Halal",
    badgeBg: "bg-emerald-50 dark:bg-emerald-950/40",
    badgeText: "text-emerald-700 dark:text-emerald-300",
    badgeBorder: "border-emerald-200 dark:border-emerald-800",
  },
  {
    id: "BPOM",
    labelKey: "supplyChain.certTypes.bpom",
    defaultAuthority: "Badan Pengawas Obat dan Makanan (BPOM RI)",
    defaultScope: "Izin Edar Makanan & Minuman Olahan (MD/ML)",
    badgeBg: "bg-sky-50 dark:bg-sky-950/40",
    badgeText: "text-sky-700 dark:text-sky-300",
    badgeBorder: "border-sky-200 dark:border-sky-800",
  },
  {
    id: "REGALKES",
    labelKey: "supplyChain.certTypes.regalkes",
    defaultAuthority: "Kementerian Kesehatan RI (Kemenkes)",
    defaultScope: "Izin Distribusi Alat Kesehatan & PKRT",
    badgeBg: "bg-purple-50 dark:bg-purple-950/40",
    badgeText: "text-purple-700 dark:text-purple-300",
    badgeBorder: "border-purple-200 dark:border-purple-800",
  },
  {
    id: "MICHELIN",
    labelKey: "supplyChain.certTypes.michelin",
    defaultAuthority: "Michelin Guide & Culinary Quality Board",
    defaultScope: "Standar Keunggulan Gastronomi & Kualitas Bahan",
    badgeBg: "bg-rose-50 dark:bg-rose-950/40",
    badgeText: "text-rose-700 dark:text-rose-300",
    badgeBorder: "border-rose-200 dark:border-rose-800",
  },
  {
    id: "ISO_HACCP",
    labelKey: "supplyChain.certTypes.isoHaccp",
    defaultAuthority: "ISO / KAN / Badan Akreditasi Mutu",
    defaultScope: "Manajemen Mutu ISO 9001 & Keamanan Pangan HACCP",
    badgeBg: "bg-amber-50 dark:bg-amber-950/40",
    badgeText: "text-amber-700 dark:text-amber-300",
    badgeBorder: "border-amber-200 dark:border-amber-800",
  },
  {
    id: "OTHER",
    labelKey: "supplyChain.certTypes.other",
    defaultAuthority: "",
    defaultScope: "",
    badgeBg: "bg-slate-50 dark:bg-slate-800/60",
    badgeText: "text-slate-700 dark:text-slate-300",
    badgeBorder: "border-slate-200 dark:border-slate-700",
  },
];

export interface ComplianceCertificate {
  id: string;
  supplier_id: string;
  cert_type: string;
  certificate_number: string;
  issuing_authority: string;
  scope: string;
  valid_from: string;
  expiry_date: string;
  document_url?: string | null;
  computed_status: CertificateStatus;
  created_at: string;
}

export interface Supplier {
  id: string;
  code: string;
  company_name: string;
  contact_person: string;
  contact_email: string;
  contact_phone: string;
  is_active: boolean;
  created_at: string;
  compliance_certificate?: ComplianceCertificate | null;
}

export interface CreateSupplierInput {
  code: string;
  company_name: string;
  contact_person: string;
  contact_email: string;
  contact_phone: string;
  compliance_certificate?: {
    cert_type: string;
    certificate_number: string;
    issuing_authority: string;
    scope: string;
    valid_from: string;
    expiry_date: string;
    document_url?: string;
  };
}

export interface RegisterCertificateInput {
  cert_type: string;
  certificate_number: string;
  issuing_authority: string;
  scope: string;
  valid_from: string;
  expiry_date: string;
  document_url?: string;
}

export interface ExpirySummary {
  total_active_suppliers: number;
  valid_count: number;
  expiring_soon_count: number;
  expired_count: number;
  missing_cert_count: number;
}
