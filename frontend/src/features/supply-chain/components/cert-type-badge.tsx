"use client";

import * as React from "react";
import { Award, ShieldCheck, Sparkles, Stethoscope, FileText, CheckCircle } from "lucide-react";
import { useTranslation } from "@/lib/i18n";

export interface CertTypeBadgeProps {
  certType?: string | null;
  className?: string;
}

export function CertTypeBadge({ certType, className = "" }: CertTypeBadgeProps) {
  const { t } = useTranslation();

  if (!certType) return null;

  const normalized = certType.toUpperCase().trim();

  switch (normalized) {
    case "HALAL":
      return (
        <span
          className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-[11px] font-bold bg-emerald-50 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300 border border-emerald-200 dark:border-emerald-800 ${className}`}
        >
          <ShieldCheck className="h-3 w-3 text-emerald-600 dark:text-emerald-400" />
          <span>{t("supplyChain.certTypes.halal")}</span>
        </span>
      );
    case "BPOM":
      return (
        <span
          className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-[11px] font-bold bg-sky-50 text-sky-700 dark:bg-sky-950/40 dark:text-sky-300 border border-sky-200 dark:border-sky-800 ${className}`}
        >
          <CheckCircle className="h-3 w-3 text-sky-600 dark:text-sky-400" />
          <span>{t("supplyChain.certTypes.bpom")}</span>
        </span>
      );
    case "REGALKES":
      return (
        <span
          className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-[11px] font-bold bg-purple-50 text-purple-700 dark:bg-purple-950/40 dark:text-purple-300 border border-purple-200 dark:border-purple-800 ${className}`}
        >
          <Stethoscope className="h-3 w-3 text-purple-600 dark:text-purple-400" />
          <span>{t("supplyChain.certTypes.regalkes")}</span>
        </span>
      );
    case "MICHELIN":
      return (
        <span
          className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-[11px] font-bold bg-rose-50 text-rose-700 dark:bg-rose-950/40 dark:text-rose-300 border border-rose-200 dark:border-rose-800 ${className}`}
        >
          <Sparkles className="h-3 w-3 text-rose-500" />
          <span>{t("supplyChain.certTypes.michelin")}</span>
        </span>
      );
    case "ISO_HACCP":
    case "ISO":
    case "HACCP":
      return (
        <span
          className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-[11px] font-bold bg-amber-50 text-amber-700 dark:bg-amber-950/40 dark:text-amber-300 border border-amber-200 dark:border-amber-800 ${className}`}
        >
          <Award className="h-3 w-3 text-amber-600 dark:text-amber-400" />
          <span>{t("supplyChain.certTypes.isoHaccp")}</span>
        </span>
      );
    default:
      return (
        <span
          className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-[11px] font-bold bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-300 border border-slate-200 dark:border-slate-700 ${className}`}
        >
          <FileText className="h-3 w-3 text-slate-500" />
          <span>{certType}</span>
        </span>
      );
  }
}
