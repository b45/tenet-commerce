"use client";

import * as React from "react";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { useTranslation } from "@/lib/i18n";
import { CERT_TYPE_PRESETS, type CreateSupplierInput } from "../types";

export interface SupplierModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (data: CreateSupplierInput) => Promise<boolean>;
}

export function SupplierModal({ isOpen, onClose, onSubmit }: SupplierModalProps) {
  const { t } = useTranslation();
  const [isSubmitting, setIsSubmitting] = React.useState(false);
  const [includeCert, setIncludeCert] = React.useState(false);

  // Form State
  const [code, setCode] = React.useState("");
  const [companyName, setCompanyName] = React.useState("");
  const [contactPerson, setContactPerson] = React.useState("");
  const [contactEmail, setContactEmail] = React.useState("");
  const [contactPhone, setContactPhone] = React.useState("");

  // Certificate State
  const [selectedPreset, setSelectedPreset] = React.useState("HALAL");
  const [customCertType, setCustomCertType] = React.useState("");
  const [certNumber, setCertNumber] = React.useState("");
  const [issuingAuthority, setIssuingAuthority] = React.useState("BPJPH Kemenag / LPPOM-MUI");
  const [scope, setScope] = React.useState("Bahan Baku & Produk Halal");
  const [validFrom, setValidFrom] = React.useState("");
  const [expiryDate, setExpiryDate] = React.useState("");
  const [docUrl, setDocUrl] = React.useState("");

  const handlePresetChange = (presetId: string) => {
    setSelectedPreset(presetId);
    const preset = CERT_TYPE_PRESETS.find((p) => p.id === presetId);
    if (preset && presetId !== "OTHER") {
      setIssuingAuthority(preset.defaultAuthority);
      setScope(preset.defaultScope);
    }
  };

  const resetForm = () => {
    setCode("");
    setCompanyName("");
    setContactPerson("");
    setContactEmail("");
    setContactPhone("");
    setIncludeCert(false);
    setSelectedPreset("HALAL");
    setCustomCertType("");
    setCertNumber("");
    setIssuingAuthority("BPJPH Kemenag / LPPOM-MUI");
    setScope("Bahan Baku & Produk Halal");
    setValidFrom("");
    setExpiryDate("");
    setDocUrl("");
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);

    const payload: CreateSupplierInput = {
      code: code.trim(),
      company_name: companyName.trim(),
      contact_person: contactPerson.trim(),
      contact_email: contactEmail.trim(),
      contact_phone: contactPhone.trim(),
    };

    if (includeCert && certNumber.trim() && expiryDate) {
      const finalCertType =
        selectedPreset === "OTHER"
          ? customCertType.trim() || "OTHER"
          : selectedPreset;

      payload.compliance_certificate = {
        cert_type: finalCertType,
        certificate_number: certNumber.trim(),
        issuing_authority: issuingAuthority.trim(),
        scope: scope.trim() || "Kepatuhan Mutu & Bahan Baku",
        valid_from: validFrom || new Date().toISOString().split("T")[0],
        expiry_date: expiryDate,
        document_url: docUrl.trim() || undefined,
      };
    }

    const success = await onSubmit(payload);
    setIsSubmitting(false);
    if (success) {
      resetForm();
      onClose();
    }
  };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={t("supplyChain.modal.createSupplierTitle")}
      description={t("supplyChain.modal.createSupplierDesc")}
      maxWidth="lg"
    >
      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div className="space-y-1">
            <label className="text-xs font-semibold text-[var(--color-text-secondary)]">
              {t("supplyChain.modal.codeLabel")} *
            </label>
            <input
              type="text"
              required
              value={code}
              onChange={(e) => setCode(e.target.value)}
              placeholder="SUPP-001"
              className="w-full h-10 px-3 text-sm font-mono rounded-lg border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] text-[var(--color-text-primary)] focus:outline-hidden focus:ring-2 focus:ring-[var(--color-action-primary)]"
            />
          </div>

          <div className="space-y-1">
            <label className="text-xs font-semibold text-[var(--color-text-secondary)]">
              {t("supplyChain.modal.nameLabel")} *
            </label>
            <input
              type="text"
              required
              value={companyName}
              onChange={(e) => setCompanyName(e.target.value)}
              placeholder="PT. Berkah Pangan Halal"
              className="w-full h-10 px-3 text-sm rounded-lg border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] text-[var(--color-text-primary)] focus:outline-hidden focus:ring-2 focus:ring-[var(--color-action-primary)]"
            />
          </div>

          <div className="space-y-1">
            <label className="text-xs font-semibold text-[var(--color-text-secondary)]">
              {t("supplyChain.modal.contactPersonLabel")}
            </label>
            <input
              type="text"
              value={contactPerson}
              onChange={(e) => setContactPerson(e.target.value)}
              placeholder="H. Abdullah"
              className="w-full h-10 px-3 text-sm rounded-lg border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] text-[var(--color-text-primary)] focus:outline-hidden focus:ring-2 focus:ring-[var(--color-action-primary)]"
            />
          </div>

          <div className="space-y-1">
            <label className="text-xs font-semibold text-[var(--color-text-secondary)]">
              {t("supplyChain.modal.phoneLabel")}
            </label>
            <input
              type="tel"
              value={contactPhone}
              onChange={(e) => setContactPhone(e.target.value)}
              placeholder="08123456789"
              className="w-full h-10 px-3 text-sm rounded-lg border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] text-[var(--color-text-primary)] focus:outline-hidden focus:ring-2 focus:ring-[var(--color-action-primary)]"
            />
          </div>

          <div className="sm:col-span-2 space-y-1">
            <label className="text-xs font-semibold text-[var(--color-text-secondary)]">
              {t("supplyChain.modal.emailLabel")}
            </label>
            <input
              type="email"
              value={contactEmail}
              onChange={(e) => setContactEmail(e.target.value)}
              placeholder="vendor@berkahpangan.com"
              className="w-full h-10 px-3 text-sm rounded-lg border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] text-[var(--color-text-primary)] focus:outline-hidden focus:ring-2 focus:ring-[var(--color-action-primary)]"
            />
          </div>
        </div>

        {/* Toggle Initial Certificate */}
        <div className="pt-2 border-t border-[var(--color-border-hairline)]">
          <label className="flex items-center gap-2 cursor-pointer select-none">
            <input
              type="checkbox"
              checked={includeCert}
              onChange={(e) => setIncludeCert(e.target.checked)}
              className="h-4 w-4 rounded border-gray-300 text-[var(--color-action-primary)] focus:ring-[var(--color-action-primary)]"
            />
            <span className="text-xs font-bold text-[var(--color-text-primary)]">
              {t("supplyChain.modal.hasInitialCert")}
            </span>
          </label>
        </div>

        {/* Collapsible Certificate Fields */}
        {includeCert && (
          <div className="p-3.5 rounded-xl border border-[var(--color-border-hairline)] bg-[var(--color-surface-muted)]/50 space-y-3">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              {/* Standard Type Selector */}
              <div className="sm:col-span-2 space-y-1">
                <label className="text-[11px] font-semibold text-[var(--color-text-secondary)]">
                  {t("supplyChain.modal.certTypeLabel")} *
                </label>
                <select
                  value={selectedPreset}
                  onChange={(e) => handlePresetChange(e.target.value)}
                  className="w-full h-9 px-2.5 text-xs rounded-md border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] text-[var(--color-text-primary)] focus:outline-hidden focus:ring-2 focus:ring-[var(--color-action-primary)]"
                >
                  {CERT_TYPE_PRESETS.map((preset) => (
                    <option key={preset.id} value={preset.id}>
                      {t(preset.labelKey)}
                    </option>
                  ))}
                </select>
              </div>

              {selectedPreset === "OTHER" && (
                <div className="sm:col-span-2 space-y-1">
                  <label className="text-[11px] font-semibold text-[var(--color-text-secondary)]">
                    {t("supplyChain.modal.customCertTypePlaceholder")} *
                  </label>
                  <input
                    type="text"
                    required={includeCert}
                    value={customCertType}
                    onChange={(e) => setCustomCertType(e.target.value)}
                    placeholder="misal: FSSC 22000 / Fair Trade / Rainforest"
                    className="w-full h-9 px-2.5 text-xs rounded-md border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] text-[var(--color-text-primary)] focus:outline-hidden focus:ring-2 focus:ring-[var(--color-action-primary)]"
                  />
                </div>
              )}

              <div className="space-y-1">
                <label className="text-[11px] font-semibold text-[var(--color-text-secondary)]">
                  {t("supplyChain.modal.certNumberLabel")} *
                </label>
                <input
                  type="text"
                  required={includeCert}
                  value={certNumber}
                  onChange={(e) => setCertNumber(e.target.value)}
                  placeholder="ID3211000012345 / MD 224310001"
                  className="w-full h-9 px-2.5 text-xs font-mono rounded-md border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] text-[var(--color-text-primary)] focus:outline-hidden focus:ring-2 focus:ring-[var(--color-action-primary)]"
                />
              </div>

              <div className="space-y-1">
                <label className="text-[11px] font-semibold text-[var(--color-text-secondary)]">
                  {t("supplyChain.modal.authorityLabel")} *
                </label>
                <input
                  type="text"
                  required={includeCert}
                  value={issuingAuthority}
                  onChange={(e) => setIssuingAuthority(e.target.value)}
                  placeholder="BPJPH Kemenag / BPOM RI / Kemenkes"
                  className="w-full h-9 px-2.5 text-xs rounded-md border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] text-[var(--color-text-primary)] focus:outline-hidden focus:ring-2 focus:ring-[var(--color-action-primary)]"
                />
              </div>

              <div className="space-y-1">
                <label className="text-[11px] font-semibold text-[var(--color-text-secondary)]">
                  {t("supplyChain.modal.validFromLabel")}
                </label>
                <input
                  type="date"
                  value={validFrom}
                  onChange={(e) => setValidFrom(e.target.value)}
                  className="w-full h-9 px-2.5 text-xs rounded-md border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] text-[var(--color-text-primary)] focus:outline-hidden focus:ring-2 focus:ring-[var(--color-action-primary)]"
                />
              </div>

              <div className="space-y-1">
                <label className="text-[11px] font-semibold text-[var(--color-text-secondary)]">
                  {t("supplyChain.modal.expiryDateLabel")} *
                </label>
                <input
                  type="date"
                  required={includeCert}
                  value={expiryDate}
                  onChange={(e) => setExpiryDate(e.target.value)}
                  className="w-full h-9 px-2.5 text-xs rounded-md border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] text-[var(--color-text-primary)] focus:outline-hidden focus:ring-2 focus:ring-[var(--color-action-primary)]"
                />
              </div>

              <div className="sm:col-span-2 space-y-1">
                <label className="text-[11px] font-semibold text-[var(--color-text-secondary)]">
                  {t("supplyChain.modal.scopeLabel")}
                </label>
                <input
                  type="text"
                  value={scope}
                  onChange={(e) => setScope(e.target.value)}
                  placeholder="Daging Ayam Segar, Sapi Potong, Tepung Gandum"
                  className="w-full h-9 px-2.5 text-xs rounded-md border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] text-[var(--color-text-primary)] focus:outline-hidden focus:ring-2 focus:ring-[var(--color-action-primary)]"
                />
              </div>
            </div>
          </div>
        )}

        <div className="flex items-center justify-end gap-2.5 pt-3 border-t border-[var(--color-border-hairline)]">
          <Button type="button" variant="secondary" onClick={onClose} disabled={isSubmitting}>
            {t("supplyChain.modal.cancel")}
          </Button>
          <Button type="submit" disabled={isSubmitting}>
            {isSubmitting ? t("supplyChain.modal.saving") : t("supplyChain.modal.submitSave")}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
