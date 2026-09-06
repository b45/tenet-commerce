"use client";

import * as React from "react";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { useTranslation } from "@/lib/i18n";
import { CertificateStatusBadge } from "./certificate-status-badge";
import { CertTypeBadge } from "./cert-type-badge";
import type { Supplier, ComplianceCertificate } from "../types";

export interface CertHistoryModalProps {
  isOpen: boolean;
  supplier: Supplier | null;
  onClose: () => void;
  onFetchCertificates: (supplierId: string) => Promise<ComplianceCertificate[]>;
  onRevokeCertificate: (certId: string) => Promise<boolean>;
}

export function CertHistoryModal({
  isOpen,
  supplier,
  onClose,
  onFetchCertificates,
  onRevokeCertificate,
}: CertHistoryModalProps) {
  const { t } = useTranslation();
  const [certs, setCerts] = React.useState<ComplianceCertificate[]>([]);
  const [isLoading, setIsLoading] = React.useState(false);
  const [revokingId, setRevokingId] = React.useState<string | null>(null);

  React.useEffect(() => {
    if (supplier && isOpen) {
      setIsLoading(true);
      onFetchCertificates(supplier.id)
        .then((data) => setCerts(data))
        .finally(() => setIsLoading(false));
    } else {
      setCerts([]);
    }
  }, [supplier, isOpen, onFetchCertificates]);

  const handleRevoke = async (certId: string) => {
    if (!confirm(t("supplyChain.modal.revokeConfirm"))) {
      return;
    }
    setRevokingId(certId);
    const success = await onRevokeCertificate(certId);
    setRevokingId(null);
    if (success && supplier) {
      const refreshed = await onFetchCertificates(supplier.id);
      setCerts(refreshed);
    }
  };

  if (!supplier) return null;

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={t("supplyChain.modal.historyTitle")}
      description={t("supplyChain.modal.historyDesc").replace("{name}", supplier.company_name)}
      maxWidth="lg"
    >
      <div className="space-y-4">
        {isLoading ? (
          <div className="py-10 text-center text-sm text-[var(--color-text-secondary)] animate-pulse">
            {t("supplyChain.loading")}
          </div>
        ) : certs.length === 0 ? (
          <div className="py-8 text-center text-sm text-[var(--color-text-tertiary)] italic">
            {t("supplyChain.modal.noHistory")}
          </div>
        ) : (
          <div className="divide-y divide-[var(--color-border-hairline)] border border-[var(--color-border-hairline)] rounded-xl overflow-hidden bg-[var(--color-surface-base)]">
            {certs.map((c) => (
              <div
                key={c.id}
                className="p-4 flex flex-col sm:flex-row sm:items-center justify-between gap-3 hover:bg-[var(--color-surface-muted)]/40 transition-colors"
              >
                <div className="space-y-1.5">
                  <div className="flex flex-wrap items-center gap-2">
                    <CertTypeBadge certType={c.cert_type} />
                    <span className="font-mono text-sm font-bold text-[var(--color-text-primary)]">
                      {c.certificate_number}
                    </span>
                    <CertificateStatusBadge status={c.computed_status} />
                  </div>
                  <div className="text-xs text-[var(--color-text-secondary)]">
                    {c.issuing_authority} • {c.scope || "Semua Bahan Baku"}
                  </div>
                  <div className="font-mono text-[11px] text-[var(--color-text-tertiary)]">
                    Masa Berlaku: {c.valid_from ? c.valid_from.split("T")[0] : "-"} s/d{" "}
                    <span className="font-bold text-[var(--color-text-primary)]">
                      {c.expiry_date ? c.expiry_date.split("T")[0] : "-"}
                    </span>
                  </div>
                </div>

                {c.computed_status === "VALID" && (
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    disabled={revokingId === c.id}
                    onClick={() => handleRevoke(c.id)}
                    className="h-8 text-xs text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-950/50 border-red-200 dark:border-red-900 shrink-0"
                  >
                    {revokingId === c.id
                      ? t("supplyChain.modal.revoking")
                      : t("supplyChain.modal.revokeAction")}
                  </Button>
                )}
              </div>
            ))}
          </div>
        )}

        <div className="flex justify-end pt-3 border-t border-[var(--color-border-hairline)]">
          <Button type="button" variant="secondary" onClick={onClose}>
            {t("supplyChain.modal.cancel")}
          </Button>
        </div>
      </div>
    </Modal>
  );
}
