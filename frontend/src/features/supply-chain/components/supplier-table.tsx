"use client";

import * as React from "react";
import { History, PlusCircle, Building2, Phone, Mail } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useTranslation } from "@/lib/i18n";
import { CertificateStatusBadge } from "./certificate-status-badge";
import { CertTypeBadge } from "./cert-type-badge";
import type { Supplier } from "../types";

export interface SupplierTableProps {
  suppliers: Supplier[];
  onOpenRenewModal: (supplier: Supplier) => void;
  onOpenHistoryModal: (supplier: Supplier) => void;
}

export function SupplierTable({
  suppliers,
  onOpenRenewModal,
  onOpenHistoryModal,
}: SupplierTableProps) {
  const { t } = useTranslation();

  if (suppliers.length === 0) {
    return (
      <div className="py-16 text-center rounded-2xl border border-dashed border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] space-y-2">
        <Building2 className="h-10 w-10 text-[var(--color-text-tertiary)] mx-auto opacity-50" />
        <p className="text-sm font-semibold text-[var(--color-text-secondary)]">
          {t("supplyChain.emptyState")}
        </p>
      </div>
    );
  }

  return (
    <div className="overflow-hidden rounded-2xl border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] shadow-2xs">
      <div className="overflow-x-auto">
        <table className="w-full text-left text-sm border-collapse">
          <thead>
            <tr className="border-b border-[var(--color-border-hairline)] bg-[var(--color-surface-muted)] text-[11px] font-bold uppercase tracking-wider text-[var(--color-text-secondary)]">
              <th scope="col" className="px-5 py-3.5">{t("supplyChain.table.supplier")}</th>
              <th scope="col" className="px-5 py-3.5">{t("supplyChain.table.contact")}</th>
              <th scope="col" className="px-5 py-3.5">{t("supplyChain.table.certificate")}</th>
              <th scope="col" className="px-5 py-3.5">{t("supplyChain.table.authority")}</th>
              <th scope="col" className="px-5 py-3.5">{t("supplyChain.table.validUntil")}</th>
              <th scope="col" className="px-5 py-3.5">{t("supplyChain.table.status")}</th>
              <th scope="col" className="px-5 py-3.5 text-right">{t("supplyChain.table.actions")}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-[var(--color-border-hairline)]">
            {suppliers.map((supplier) => {
              const cert = supplier.compliance_certificate;
              return (
                <tr
                  key={supplier.id}
                  className="hover:bg-[var(--color-surface-muted)]/50 transition-colors"
                >
                  {/* Supplier Info */}
                  <td className="px-5 py-4">
                    <div className="space-y-0.5">
                      <div className="font-bold text-[var(--color-text-primary)]">
                        {supplier.company_name}
                      </div>
                      <div className="font-mono text-xs text-[var(--color-text-tertiary)]">
                        {supplier.code}
                      </div>
                    </div>
                  </td>

                  {/* Contact Info */}
                  <td className="px-5 py-4">
                    <div className="space-y-1 text-xs text-[var(--color-text-secondary)]">
                      <div className="font-medium text-[var(--color-text-primary)]">
                        {supplier.contact_person || "-"}
                      </div>
                      {supplier.contact_email && (
                        <div className="flex items-center gap-1 text-[11px]">
                          <Mail className="h-3 w-3" />
                          <span>{supplier.contact_email}</span>
                        </div>
                      )}
                      {supplier.contact_phone && (
                        <div className="flex items-center gap-1">
                          <Phone className="h-3 w-3" />
                          <span>{supplier.contact_phone}</span>
                        </div>
                      )}
                    </div>
                  </td>

                  {/* Compliance Certificate Details */}
                  <td className="px-5 py-4">
                    {cert ? (
                      <div className="space-y-1">
                        <div className="flex flex-wrap items-center gap-1.5">
                          <CertTypeBadge certType={cert.cert_type} />
                          <span className="font-mono text-xs font-bold text-[var(--color-text-primary)]">
                            {cert.certificate_number}
                          </span>
                        </div>
                        <div className="text-[11px] text-[var(--color-text-secondary)] truncate max-w-[200px]">
                          {cert.scope}
                        </div>
                      </div>
                    ) : (
                      <span className="text-xs text-[var(--color-text-tertiary)] italic">
                        {t("supplyChain.table.noCertificate")}
                      </span>
                    )}
                  </td>

                  {/* Issuing Authority */}
                  <td className="px-5 py-4">
                    {cert ? (
                      <span className="text-xs font-semibold text-[var(--color-text-primary)]">
                        {cert.issuing_authority}
                      </span>
                    ) : (
                      <span className="text-xs text-[var(--color-text-tertiary)]">-</span>
                    )}
                  </td>

                  {/* Expiry Date */}
                  <td className="px-5 py-4">
                    {cert ? (
                      <div className="font-mono text-xs space-y-0.5">
                        <div className="font-semibold text-[var(--color-text-primary)]">
                          {cert.expiry_date ? cert.expiry_date.split("T")[0] : "-"}
                        </div>
                        <div className="text-[10px] text-[var(--color-text-secondary)]">
                          Mulai: {cert.valid_from ? cert.valid_from.split("T")[0] : "-"}
                        </div>
                      </div>
                    ) : (
                      <span className="text-xs text-[var(--color-text-tertiary)]">-</span>
                    )}
                  </td>

                  {/* Status Badge */}
                  <td className="px-5 py-4">
                    <CertificateStatusBadge status={cert?.computed_status} />
                  </td>

                  {/* Action Buttons */}
                  <td className="px-5 py-4 text-right">
                    <div className="flex items-center justify-end gap-1.5">
                      <Button
                        type="button"
                        variant="ghost"
                        size="sm"
                        onClick={() => onOpenHistoryModal(supplier)}
                        title={t("supplyChain.table.viewHistory")}
                        className="h-8 px-2.5 text-xs gap-1 text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
                      >
                        <History className="h-3.5 w-3.5" />
                        <span className="hidden xl:inline">{t("supplyChain.table.viewHistory")}</span>
                      </Button>

                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={() => onOpenRenewModal(supplier)}
                        title={t("supplyChain.table.renewCert")}
                        className="h-8 px-2.5 text-xs gap-1 font-medium border-[var(--color-border-hairline)]"
                      >
                        <PlusCircle className="h-3.5 w-3.5 text-[var(--color-action-primary)]" />
                        <span>{t("supplyChain.table.renewCert")}</span>
                      </Button>
                    </div>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}
