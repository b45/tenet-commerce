"use client";

import * as React from "react";
import { Banknote, AlertCircle, Loader2 } from "lucide-react";
import { formatIDR } from "@/lib/money";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { Alert } from "@/components/ui/alert";
import { cn } from "@/lib/utils";

export interface TenderModalProps {
  isOpen: boolean;
  onClose: () => void;
  totalAmount: number;
  cashTendered: number;
  onCashTenderedChange: (amount: number) => void;
  onSubmit: () => void;
  isSubmitting: boolean;
  errorMessage: string | null;
  step: string;
}

export function TenderModal({
  isOpen,
  onClose,
  totalAmount,
  cashTendered,
  onCashTenderedChange,
  onSubmit,
  isSubmitting,
  errorMessage,
  step,
}: TenderModalProps) {
  const [inputValue, setInputValue] = React.useState<string>(
    cashTendered ? String(cashTendered) : ""
  );

  React.useEffect(() => {
    if (isOpen) {
      setInputValue(cashTendered ? String(cashTendered) : "");
    }
  }, [isOpen, cashTendered]);

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const raw = e.target.value.replace(/\D/g, "");
    setInputValue(raw);
    const num = raw ? parseInt(raw, 10) : 0;
    onCashTenderedChange(num);
  };

  const setPreset = (amount: number) => {
    setInputValue(String(amount));
    onCashTenderedChange(amount);
  };

  const changeAmount = Math.max(0, cashTendered - totalAmount);
  const shortageAmount = Math.max(0, totalAmount - cashTendered);
  const isSufficient = cashTendered >= totalAmount;

  // Preset cash options based on total
  const quickPresets = React.useMemo(() => {
    const presets: number[] = [];
    if (totalAmount <= 50000) presets.push(50000);
    if (totalAmount <= 100000) presets.push(100000);
    if (totalAmount <= 200000 && !presets.includes(200000)) presets.push(200000);
    if (totalAmount <= 500000 && !presets.includes(500000)) presets.push(500000);
    return presets.slice(0, 3);
  }, [totalAmount]);

  return (
    <Modal
      isOpen={isOpen}
      onClose={isSubmitting ? () => {} : onClose}
      title="Pembayaran Tunai (Cash)"
      description="Verifikasi nominal uang yang diterima dari pelanggan"
      maxWidth="md"
    >
      <div className="space-y-5">
        {/* Error / Warning Alert */}
        {errorMessage && (
          <Alert
            variant={step === "unknown_error" ? "warning" : "destructive"}
            title={
              step === "unknown_error"
                ? "Status Belum Diketahui"
                : "Transaksi Tidak Dapat Dilanjutkan"
            }
          >
            <div className="flex items-start gap-1.5">
              <AlertCircle className="w-4 h-4 shrink-0 mt-0.5" />
              <span>{errorMessage}</span>
            </div>
            {step === "unknown_error" && (
              <p className="mt-2 text-xs font-semibold text-amber-800">
                PENTING: Jangan menagih uang kembali ke pelanggan sebelum memeriksa mutasi bank atau laporan audit sistem.
              </p>
            )}
          </Alert>
        )}

        {/* Bill Total Display */}
        <div className="p-4 rounded-[16px] bg-[var(--color-surface-muted)] border border-[var(--color-border-hairline)] text-center">
          <span className="text-xs font-semibold text-[var(--color-text-secondary)] uppercase tracking-wider">
            Total Tagihan
          </span>
          <div className="text-3xl font-bold font-mono text-[var(--color-text-primary)] mt-1 tracking-tight">
            {formatIDR(totalAmount)}
          </div>
        </div>

        {/* Cash Tendered Input */}
        <div>
          <label
            htmlFor="cash_tendered"
            className="block text-xs font-medium text-[var(--color-text-secondary)] mb-1.5"
          >
            Uang Tunai Diterima (Rp)
          </label>

          <div className="relative">
            <div className="absolute left-3.5 top-1/2 -translate-y-1/2 text-sm font-semibold font-mono text-[var(--color-text-muted)]">
              Rp
            </div>
            <input
              id="cash_tendered"
              type="text"
              inputMode="numeric"
              autoFocus
              disabled={isSubmitting}
              value={inputValue ? parseInt(inputValue, 10).toLocaleString("id-ID") : ""}
              onChange={handleInputChange}
              placeholder="0"
              className={cn(
                "w-full h-13 pl-12 pr-4 text-xl font-bold font-mono text-[var(--color-text-primary)]",
                "bg-[var(--color-surface-base)] rounded-[16px] border border-[var(--color-border-subtle)]",
                "focus:outline-none focus:ring-2 focus:ring-[var(--color-action-focus-ring)]",
                !isSufficient && cashTendered > 0 && "border-[var(--color-status-danger-border)] focus:ring-red-200"
              )}
            />
          </div>
        </div>

        {/* Quick Cash Presets */}
        <div className="space-y-1.5">
          <span className="text-[11px] font-medium text-[var(--color-text-muted)]">
            Tombol Cepat:
          </span>
          <div className="flex flex-wrap gap-2">
            <button
              type="button"
              disabled={isSubmitting}
              onClick={() => setPreset(totalAmount)}
              className={cn(
                "px-3 py-1.5 rounded-xl text-xs font-semibold transition-all select-none",
                cashTendered === totalAmount
                  ? "bg-[var(--color-action-primary)] text-white shadow-xs"
                  : "bg-[var(--color-surface-muted)] text-[var(--color-text-primary)] border border-[var(--color-border-hairline)] hover:bg-gray-200"
              )}
            >
              Uang Pas ({formatIDR(totalAmount)})
            </button>

            {quickPresets.map((amount) => (
              <button
                key={amount}
                type="button"
                disabled={isSubmitting}
                onClick={() => setPreset(amount)}
                className={cn(
                  "px-3 py-1.5 rounded-xl text-xs font-semibold font-mono transition-all select-none",
                  cashTendered === amount
                    ? "bg-[var(--color-action-primary)] text-white shadow-xs"
                    : "bg-[var(--color-surface-muted)] text-[var(--color-text-primary)] border border-[var(--color-border-hairline)] hover:bg-gray-200"
                )}
              >
                {formatIDR(amount)}
              </button>
            ))}
          </div>
        </div>

        {/* Real-time Change / Shortage Preview */}
        <div className="p-4 rounded-[16px] border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] flex items-center justify-between">
          <span className="text-xs font-medium text-[var(--color-text-secondary)]">
            {isSufficient ? "Kembalian Pelanggan:" : "Kurang Bayar:"}
          </span>
          <span
            className={cn(
              "text-lg font-bold font-mono tracking-tight",
              isSufficient
                ? "text-[var(--color-status-success-text)]"
                : "text-[var(--color-status-danger-text)]"
            )}
          >
            {isSufficient ? formatIDR(changeAmount) : formatIDR(shortageAmount)}
          </span>
        </div>

        {/* Actions */}
        <div className="pt-2 flex gap-3">
          <Button
            type="button"
            variant="secondary"
            disabled={isSubmitting}
            onClick={onClose}
            className="flex-1 rounded-[14px] h-12"
          >
            Batal
          </Button>

          <Button
            type="button"
            disabled={!isSufficient || isSubmitting}
            onClick={onSubmit}
            className="flex-[2] rounded-[14px] h-12 font-semibold shadow-sm flex items-center justify-center gap-2"
          >
            {isSubmitting ? (
              <>
                <Loader2 className="w-4 h-4 animate-spin" />
                <span>Memproses...</span>
              </>
            ) : (
              <>
                <Banknote className="w-4 h-4" />
                <span>Proses Pembayaran</span>
              </>
            )}
          </Button>
        </div>
      </div>
    </Modal>
  );
}
