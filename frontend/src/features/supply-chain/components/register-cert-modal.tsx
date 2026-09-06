"use client";

import * as React from "react";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { useTranslation } from "@/lib/i18n";
import { CERT_TYPE_PRESETS, type Supplier, type RegisterCertificateInput } from "../types";

export interface RegisterCertModalProps {
  isOpen: boolean;
  supplier: Supplier | null;
  onClose: () => void;
  onSubmit: (supplierId: string, data: RegisterCertificateInput) => Promise<boolean>;
}

export function RegisterCertModal({
  isOpen,
  supplier,
  onClose,
  onSubmit,
}: RegisterCertModalProps) {
  const { t } = useTranslation();
  const [isSubmitting, setIsSubmitting] = React.useState(false);

  // Certificate Form State
  const [selectedPreset, setSelectedPreset] = React.useState("HALAL");
  const [customCertType, setCustomCertType] = React.useState("");
  const [certNumber, setCertNumber] = React.useState("");
  const [issuingAuthority, setIssuingAuthority] = React.useState("BPJPH Kemenag / LPPOM-MUI");
  const [scope, setScope] = React.useState("");
  const [validFrom, setValidFrom] = React.useState("");
  const [expiryDate, setExpiryDate] = React.useState("");
  const [docUrl, setDocUrl] = React.useState("");

  React.useEffect(() => {
    if (supplier?.compliance_certificate) {
      const c = supplier.compliance_certificate;
      const matchedPreset = CERT_TYPE_PRESETS.find(
        (p) => p.id === c.cert_type?.toUpperCase()
      );
      if (matchedPreset) {
        setSelectedPreset(matchedPreset.id);
        setCustomCertType("");
      } else {
        setSelectedPreset("OTHER");
        setCustomCertType(c.cert_type || "");
      }
      setCertNumber(c.certificate_number || "");
      setIssuingAuthority(c.issuing_authority || "");
      setScope(c.scope || "");
      setValidFrom(c.valid_from ? c.valid_from.split("T")[0] : "");
      setExpiryDate(c.expiry_date ? c.expiry_date.split("T")[0] : "");
      setDocUrl(c.document_url || "");
    } else {
      setSelectedPreset("HALAL");
      setCustomCertType("");
      setCertNumber("");
      setIssuingAuthority("BPJPH Kemenag / LPPOM-MUI");
      setScope("Bahan Baku & Produk Halal");
      setValidFrom("");
      setExpiryDate("");
      setDocUrl("");
    }
  }, [supplier]);

  const handlePresetChange = (presetId: string) => {
    setSelectedPreset(presetId);
    const preset = CERT_TYPE_PRESETS.find((p) => p.id === presetId);
    if (preset && presetId !== "OTHER") {
      setIssuingAuthority(preset.defaultAuthority);
      setScope(preset.defaultScope);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!supplier) return;
    setIsSubmitting(true);

    const finalCertType =
      selectedPreset === "OTHER"
        ? customCertType.trim() || "OTHER"
        : selectedPreset;

    const payload: RegisterCertificateInput = {
      cert_type: finalCertType,
      certificate_number: certNumber.trim(),
      issuing_authority: issuingAuthority.trim(),
      scope: scope.trim() || "Kepatuhan Mutu & Bahan Baku",
      valid_from: validFrom || new Date().toISOString().split("T")[0],
      expiry_date: expiryDate,
      document_url: docUrl.trim() || undefined,
    };

    const success = await onSubmit(supplier.id, payload);
    setIsSubmitting(false);
    if (success) {
      onClose();
    }
  };

  if (!supplier) return null;

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={t("supplyChain.modal.renewCertTitle")}
      description={t("supplyChain.modal.renewCertDesc").replace("{name}", supplier.company_name)}
      maxWidth="md"
    >
      <form onSubmit={handleSubmit} className="space-y-4">
        {/* Certificate Standard Selection */}
        <div className="space-y-1">
          <label className="text-xs font-semibold text-[var(--color-text-secondary)]">
            {t("supplyChain.modal.certTypeLabel")} *
          </label>
          <select
            value={selectedPreset}
            onChange={(e) => handlePresetChange(e.target.value)}
            className="w-full h-10 px-3 text-sm rounded-lg border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] text-[var(--color-text-primary)] focus:outline-hidden focus:ring-2 focus:ring-[var(--color-action-primary)]"
          >
            {CERT_TYPE_PRESETS.map((preset) => (
              <option key={preset.id} value={preset.id}>
                {t(preset.labelKey)}
              </option>
            ))}
          </select>
        </div>

        {/* Custom Cert Type input if OTHER selected */}
        {selectedPreset === "OTHER" && (
          <div className="space-y-1">
            <label className="text-xs font-semibold text-[var(--color-text-secondary)]">
              {t("supplyChain.modal.customCertTypePlaceholder")} *
            </label>
            <input
              type="text"
              required
              value={customCertType}
              onChange={(e) => setCustomCertType(e.target.value)}
              placeholder="misal: FSSC 22000 / Fair Trade / Rainforest"
              className="w-full h-10 px-3 text-sm rounded-lg border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] text-[var(--color-text-primary)] focus:outline-hidden focus:ring-2 focus:ring-[var(--color-action-primary)]"
            />
          </div>
        )}

        <div className="space-y-1">
          <label className="text-xs font-semibold text-[var(--color-text-secondary)]">
            {t("supplyChain.modal.certNumberLabel")} *
          </label>
          <input
            type="text"
            required
            value={certNumber}
            onChange={(e) => setCertNumber(e.target.value)}
            placeholder="ID3211000012345 / MD 224310001"
            className="w-full h-10 px-3 text-sm font-mono rounded-lg border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] text-[var(--color-text-primary)] focus:outline-hidden focus:ring-2 focus:ring-[var(--color-action-primary)]"
          />
        </div>

        <div className="space-y-1">
          <label className="text-xs font-semibold text-[var(--color-text-secondary)]">
            {t("supplyChain.modal.authorityLabel")} *
          </label>
          <input
            type="text"
            required
            value={issuingAuthority}
            onChange={(e) => setIssuingAuthority(e.target.value)}
            placeholder="BPJPH Kemenag / BPOM RI / Kemenkes"
            className="w-full h-10 px-3 text-sm rounded-lg border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] text-[var(--color-text-primary)] focus:outline-hidden focus:ring-2 focus:ring-[var(--color-action-primary)]"
          />
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <div className="space-y-1">
            <label className="text-xs font-semibold text-[var(--color-text-secondary)]">
              {t("supplyChain.modal.validFromLabel")}
            </label>
            <input
              type="date"
              value={validFrom}
              onChange={(e) => setValidFrom(e.target.value)}
              className="w-full h-10 px-3 text-xs rounded-lg border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] text-[var(--color-text-primary)] focus:outline-hidden focus:ring-2 focus:ring-[var(--color-action-primary)]"
            />
          </div>

          <div className="space-y-1">
            <label className="text-xs font-semibold text-[var(--color-text-secondary)]">
              {t("supplyChain.modal.expiryDateLabel")} *
            </label>
            <input
              type="date"
              required
              value={expiryDate}
              onChange={(e) => setExpiryDate(e.target.value)}
              className="w-full h-10 px-3 text-xs rounded-lg border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] text-[var(--color-text-primary)] focus:outline-hidden focus:ring-2 focus:ring-[var(--color-action-primary)]"
            />
          </div>
        </div>

        <div className="space-y-1">
          <label className="text-xs font-semibold text-[var(--color-text-secondary)]">
            {t("supplyChain.modal.scopeLabel")}
          </label>
          <input
            type="text"
            value={scope}
            onChange={(e) => setScope(e.target.value)}
            placeholder="Bahan Baku, Daging Segar, Gandum"
            className="w-full h-10 px-3 text-sm rounded-lg border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] text-[var(--color-text-primary)] focus:outline-hidden focus:ring-2 focus:ring-[var(--color-action-primary)]"
          />
        </div>

        <div className="space-y-1">
          <label className="text-xs font-semibold text-[var(--color-text-secondary)]">
            {t("supplyChain.modal.docUrlLabel")}
          </label>
          <input
            type="url"
            value={docUrl}
            onChange={(e) => setDocUrl(e.target.value)}
            placeholder="https://bpjph.halal.go.id/cert/..."
            className="w-full h-10 px-3 text-sm font-mono rounded-lg border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] text-[var(--color-text-primary)] focus:outline-hidden focus:ring-2 focus:ring-[var(--color-action-primary)]"
          />
        </div>

        <div className="flex items-center justify-end gap-2.5 pt-3 border-t border-[var(--color-border-hairline)]">
          <Button type="button" variant="secondary" onClick={onClose} disabled={isSubmitting}>
            {t("supplyChain.modal.cancel")}
          </Button>
          <Button type="submit" disabled={isSubmitting}>
            {isSubmitting ? t("supplyChain.modal.saving") : t("supplyChain.modal.submitRenew")}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
