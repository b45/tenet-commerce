"use client";

import * as React from "react";
import { History, Eye, Ban, RotateCw, AlertTriangle } from "lucide-react";
import { apiClient } from "@/lib/api";
import { formatIDR } from "@/lib/money";
import { formatDateTime } from "@/lib/date";
import { Badge } from "@/components/ui/badge";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { Alert } from "@/components/ui/alert";
import { cn } from "@/lib/utils";
import { useTranslation } from "@/lib/i18n";
import type { Order, OrderDetailResponse } from "../types";

export function OrderHistory() {
  const { t } = useTranslation();
  const [orders, setOrders] = React.useState<Order[]>([]);
  const [isLoading, setIsLoading] = React.useState<boolean>(true);
  const [error, setError] = React.useState<string | null>(null);

  // Selected Order for Detail
  const [selectedOrder, setSelectedOrder] = React.useState<Order | null>(null);

  // Void confirmation state
  const [orderToVoid, setOrderToVoid] = React.useState<Order | null>(null);
  const [voidReason, setVoidReason] = React.useState<string>("");
  const [isVoiding, setIsVoiding] = React.useState<boolean>(false);
  const [voidError, setVoidError] = React.useState<string | null>(null);

  const fetchOrders = React.useCallback(async () => {
    try {
      setIsLoading(true);
      setError(null);
      const res = await apiClient.get<Order[]>("/pos/orders?limit=20");
      if (res.success && res.data) {
        setOrders(Array.isArray(res.data) ? res.data : []);
      } else {
        setError(res.error?.message || "Gagal memuat riwayat transaksi");
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Gagal menghubungi server");
    } finally {
      setIsLoading(false);
    }
  }, []);

  React.useEffect(() => {
    fetchOrders();
  }, [fetchOrders]);

  const handleOpenDetail = async (order: Order) => {
    try {
      const res = await apiClient.get<OrderDetailResponse | Order>(`/pos/orders/${order.id}`);
      if (res.success && res.data) {
        const payload = res.data;
        if ("transaction" in payload && payload.transaction) {
          setSelectedOrder({
            ...order,
            ...payload.transaction,
            items: payload.items || [],
          });
        } else {
          setSelectedOrder({
            ...order,
            ...(payload as Order),
            items: (payload as Order).items || order.items || [],
          });
        }
      } else {
        setSelectedOrder(order);
      }
    } catch {
      setSelectedOrder(order);
    }
  };

  const handleConfirmVoid = async () => {
    if (!orderToVoid) return;
    if (!voidReason.trim()) {
      setVoidError("Alasan pembatalan (void) wajib diisi.");
      return;
    }

    try {
      setIsVoiding(true);
      setVoidError(null);

      const idemKey = typeof crypto !== "undefined" && crypto.randomUUID
        ? crypto.randomUUID()
        : `void_${Date.now()}`;

      const res = await apiClient.post(
        `/pos/orders/${orderToVoid.id}/void`,
        { reason: voidReason.trim() },
        { headers: { "Idempotency-Key": idemKey } }
      );

      if (res.success) {
        setOrderToVoid(null);
        setVoidReason("");
        fetchOrders();
      } else {
        setVoidError(res.error?.message || "Gagal membatalkan transaksi.");
      }
    } catch (err: unknown) {
      setVoidError(err instanceof Error ? err.message : "Gagal menghubungi server");
    } finally {
      setIsVoiding(false);
    }
  };

  return (
    <div className="space-y-4">
      {/* Header controls */}
      {/* Header controls */}
      <div className="flex items-center justify-between gap-4 pb-1">
        <div>
          <h3 className="text-base font-semibold tracking-tight text-[var(--color-text-primary)]">
            {t("history.title")}
          </h3>
          <p className="text-xs text-[var(--color-text-secondary)]">
            {t("history.subtitle")}
          </p>
        </div>

        <button
          type="button"
          onClick={fetchOrders}
          title={t("common.actions.refresh")}
          className="h-9 px-4 rounded-full bg-[var(--color-surface-base)] border border-[var(--color-border-subtle)] text-xs font-semibold text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-surface-muted)] shadow-xs active:scale-95 transition-all inline-flex items-center gap-2 shrink-0"
        >
          <RotateCw className={cn("w-3.5 h-3.5", isLoading && "animate-spin text-[var(--color-action-primary)]")} />
          <span>{t("common.actions.refresh")}</span>
        </button>
      </div>

      {/* Orders Table */}
      <div className="bg-[var(--color-surface-base)] rounded-[20px] border border-[var(--color-border-hairline)] shadow-sm overflow-hidden">
        {isLoading ? (
          <div className="p-10 text-center text-xs text-[var(--color-text-muted)] animate-pulse">
            {t("common.actions.loading")}
          </div>
        ) : error ? (
          <div className="p-8 text-center space-y-3">
            <p className="text-xs text-[var(--color-status-danger-text)]">{error}</p>
            <Button size="sm" variant="secondary" onClick={fetchOrders}>
              {t("common.actions.retry")}
            </Button>
          </div>
        ) : orders.length === 0 ? (
          <div className="p-12 text-center select-none">
            <History className="w-10 h-10 text-[var(--color-text-muted)] mx-auto mb-2" />
            <p className="text-sm font-medium text-[var(--color-text-secondary)]">
              {t("history.emptyHistory")}
            </p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead className="bg-[var(--color-surface-muted)] text-[var(--color-text-secondary)] border-b border-[var(--color-border-hairline)] font-medium select-none">
                <tr>
                  <th className="py-3.5 px-5 text-left font-semibold">{t("history.columns.trxNumber")}</th>
                  <th className="py-3.5 px-5 text-left font-semibold">{t("history.columns.time")}</th>
                  <th className="py-3.5 px-5 text-right font-semibold">{t("history.columns.totalBill")}</th>
                  <th className="py-3.5 px-5 text-center font-semibold">{t("history.columns.method")}</th>
                  <th className="py-3.5 px-5 text-center font-semibold">{t("history.columns.status")}</th>
                  <th className="py-3.5 px-5 text-center font-semibold">{t("history.columns.action")}</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-[var(--color-border-hairline)]">
                {orders.map((order) => (
                  <tr
                    key={order.id}
                    className="hover:bg-[var(--color-surface-muted)]/60 transition-colors"
                  >
                    <td className="py-3.5 px-5 text-left font-mono font-semibold text-[var(--color-text-primary)]">
                      {order.transaction_number}
                    </td>
                    <td className="py-3.5 px-5 text-left text-[var(--color-text-secondary)]">
                      {formatDateTime(order.created_at)}
                    </td>
                    <td className="py-3.5 px-5 text-right font-mono font-bold text-[var(--color-text-primary)]">
                      {formatIDR(order.total_amount)}
                    </td>
                    <td className="py-3.5 px-5 text-center font-medium uppercase text-[var(--color-text-secondary)]">
                      {order.payment_method}
                    </td>
                    <td className="py-3.5 px-5 text-center">
                      <div className="inline-flex justify-center">
                        {order.status === "COMPLETED" ? (
                          <Badge variant="success" className="text-[10px]">
                            COMPLETED
                          </Badge>
                        ) : order.status === "VOIDED" ? (
                          <Badge variant="danger" className="text-[10px]">
                            VOIDED
                          </Badge>
                        ) : (
                          <Badge variant="outline" className="text-[10px]">
                            {order.status}
                          </Badge>
                        )}
                      </div>
                    </td>
                    <td className="py-3.5 px-5 text-center">
                      <div className="inline-flex items-center justify-center gap-1.5">
                        <button
                          type="button"
                          onClick={() => handleOpenDetail(order)}
                          title="Lihat Detail Transaksi"
                          className="p-1.5 rounded-lg text-[var(--color-text-secondary)] hover:text-[var(--color-action-primary)] hover:bg-[var(--color-surface-muted)] transition-colors"
                        >
                          <Eye className="w-4 h-4" />
                        </button>

                        {order.status === "COMPLETED" && (
                          <button
                            type="button"
                            onClick={() => {
                              setOrderToVoid(order);
                              setVoidReason("");
                              setVoidError(null);
                            }}
                            title="Batalkan Transaksi (Void)"
                            className="p-1.5 rounded-lg text-[var(--color-text-secondary)] hover:text-[var(--color-status-danger-text)] hover:bg-[var(--color-surface-muted)] transition-colors"
                          >
                            <Ban className="w-4 h-4" />
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Order Detail Modal */}
      {selectedOrder && (
        <Modal
          isOpen={Boolean(selectedOrder)}
          onClose={() => setSelectedOrder(null)}
          title={`${t("history.detailModal.title")} ${selectedOrder.transaction_number}`}
          description={`${t("history.detailModal.time")}: ${formatDateTime(selectedOrder.created_at)}`}
          maxWidth="md"
        >
          <div className="space-y-4">
            <div className="p-3.5 rounded-xl bg-[var(--color-surface-muted)] border border-[var(--color-border-hairline)] text-xs space-y-1.5">
              <div className="flex justify-between">
                <span>{t("history.detailModal.status")}:</span>
                <span className="font-semibold">{selectedOrder.status}</span>
              </div>
              <div className="flex justify-between">
                <span>{t("history.detailModal.cashierId")}:</span>
                <span className="font-mono">{selectedOrder.cashier_id}</span>
              </div>
              {selectedOrder.void_reason && (
                <div className="flex justify-between text-[var(--color-status-danger-text)]">
                  <span>{t("history.detailModal.voidReason")}:</span>
                  <span>{selectedOrder.void_reason}</span>
                </div>
              )}
            </div>

            {/* Items */}
            <div className="space-y-2">
              <h5 className="text-xs font-semibold text-[var(--color-text-secondary)] uppercase">
                {t("history.detailModal.saleItems")}
              </h5>
              <div className="divide-y divide-[var(--color-border-hairline)] border-y border-[var(--color-border-hairline)] max-h-48 overflow-y-auto">
                {(selectedOrder.items || []).map((item, i) => (
                  <div key={i} className="py-2 flex justify-between text-xs">
                    <div>
                      <p className="font-medium text-[var(--color-text-primary)]">
                        {item.name || item.product_name || "Produk"}
                      </p>
                      <p className="text-[10px] font-mono text-[var(--color-text-muted)]">
                        {item.quantity} x {formatIDR(item.unit_price)}
                      </p>
                    </div>
                    <span className="font-mono font-semibold">
                      {formatIDR(item.subtotal ?? item.subtotal_amount ?? 0)}
                    </span>
                  </div>
                ))}
              </div>
            </div>

            <div className="flex justify-between text-sm font-bold pt-2">
              <span>{t("history.detailModal.totalPayment")}:</span>
              <span className="font-mono">{formatIDR(selectedOrder.total_amount)}</span>
            </div>

            <div className="pt-2">
              <Button
                variant="secondary"
                onClick={() => setSelectedOrder(null)}
                className="w-full rounded-xl"
              >
                {t("history.detailModal.close")}
              </Button>
            </div>
          </div>
        </Modal>
      )}

      {/* Void Order Confirmation Modal */}
      {orderToVoid && (
        <Modal
          isOpen={Boolean(orderToVoid)}
          onClose={isVoiding ? () => {} : () => setOrderToVoid(null)}
          title={t("history.voidModal.title")}
          description={t("history.voidModal.description")}
          maxWidth="sm"
        >
          <div className="space-y-4">
            <Alert variant="destructive">
              <div className="flex items-start gap-2">
                <AlertTriangle className="w-4 h-4 shrink-0 mt-0.5" />
                <span>
                  {t("history.voidModal.warningText", {
                    number: orderToVoid.transaction_number,
                    amount: formatIDR(orderToVoid.total_amount),
                  })}
                </span>
              </div>
            </Alert>

            {voidError && (
              <p className="text-xs text-[var(--color-status-danger-text)]">{voidError}</p>
            )}

            <div>
              <label htmlFor="void_reason_input" className="block text-xs font-medium text-[var(--color-text-secondary)] mb-1">
                {t("history.voidModal.reasonLabel")}
              </label>
              <textarea
                id="void_reason_input"
                rows={2}
                value={voidReason}
                onChange={(e) => setVoidReason(e.target.value)}
                placeholder={t("history.voidModal.reasonPlaceholder")}
                className="w-full p-2.5 text-xs rounded-xl border border-[var(--color-border-subtle)] focus:ring-2 focus:ring-[var(--color-action-focus-ring)]"
              />
            </div>

            <div className="flex gap-2.5 pt-2">
              <Button
                variant="secondary"
                disabled={isVoiding}
                onClick={() => setOrderToVoid(null)}
                className="flex-1 rounded-xl"
              >
                {t("history.voidModal.cancel")}
              </Button>
              <Button
                variant="destructive"
                disabled={isVoiding || !voidReason.trim()}
                onClick={handleConfirmVoid}
                className="flex-1 rounded-xl font-semibold"
              >
                {isVoiding ? t("history.voidModal.processing") : t("history.voidModal.confirmVoid")}
              </Button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  );
}
