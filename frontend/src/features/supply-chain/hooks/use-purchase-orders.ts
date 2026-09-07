"use client";

import * as React from "react";
import { apiClient } from "@/lib/api";
import type { Supplier } from "../types";
import type { PurchaseOrderDetail, PurchaseOrderDraft, PurchaseOrderSummary } from "../purchase-orders";
import type { InventoryProduct } from "@/features/inventory/types";

export function usePurchaseOrders() {
  const [orders, setOrders] = React.useState<PurchaseOrderSummary[]>([]);
  const [suppliers, setSuppliers] = React.useState<Supplier[]>([]);
  const [products, setProducts] = React.useState<InventoryProduct[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);

  const refresh = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    const [ordersRes, suppliersRes, productsRes] = await Promise.all([
      apiClient.get<PurchaseOrderSummary[]>("/supply-chain/purchase-orders"),
      apiClient.get<Supplier[]>("/supply-chain/suppliers"),
      apiClient.get<InventoryProduct[]>("/pos/products"),
    ]);
    if (ordersRes.success && ordersRes.data) setOrders(ordersRes.data);
    if (suppliersRes.success && suppliersRes.data) setSuppliers(suppliersRes.data);
    if (productsRes.success && productsRes.data) setProducts(productsRes.data);
    const failure = [ordersRes, suppliersRes, productsRes].find((result) => !result.success);
    if (failure && !failure.success) setError(failure.error?.message || "");
    setLoading(false);
  }, []);

  React.useEffect(() => { void refresh(); }, [refresh]);

  const getDetail = React.useCallback(async (id: string) => {
    const result = await apiClient.get<PurchaseOrderDetail>(`/supply-chain/purchase-orders/${id}`);
    if (!result.success || !result.data) throw new Error(result.error?.message || "");
    return result.data;
  }, []);

  const create = React.useCallback(async (draft: PurchaseOrderDraft) => {
    const result = await apiClient.post<PurchaseOrderSummary>("/supply-chain/purchase-orders", draft, {
      headers: { "Idempotency-Key": `po_ui_${crypto.randomUUID()}` },
    });
    if (!result.success || !result.data) throw new Error(result.error?.message || "");
    await refresh();
    return result.data;
  }, [refresh]);

  const cancel = React.useCallback(async (id: string, reason: string) => {
    const result = await apiClient.put<{ cancelled: boolean; purchase_order: PurchaseOrderSummary }>(`/supply-chain/purchase-orders/${id}/cancel`, { reason }, {
      headers: { "Idempotency-Key": `po_cancel_${id}_${crypto.randomUUID()}` },
    });
    if (!result.success || !result.data?.purchase_order) throw new Error(result.error?.message || "");
    await refresh();
    return result.data.purchase_order;
  }, [refresh]);

  return { orders, suppliers, products, loading, error, refresh, getDetail, create, cancel };
}
