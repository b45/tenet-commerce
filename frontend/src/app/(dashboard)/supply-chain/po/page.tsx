"use client";

import * as React from "react";
import { useSearchParams } from "next/navigation";
import { useTranslation } from "@/lib/i18n";
import { formatIDR } from "@/lib/money";
import { usePurchaseOrders } from "@/features/supply-chain/hooks/use-purchase-orders";

export default function PurchaseOrdersPage() {
  const { t } = useTranslation();
  const params = useSearchParams();
  const { orders, suppliers, products, loading, error, getDetail, create, cancel } = usePurchaseOrders();
  const [selectedId, setSelectedId] = React.useState(params.get("id"));
  const [supplierId, setSupplierId] = React.useState(params.get("supplier_id") || "");
  const [productId, setProductId] = React.useState("");
  const [quantity, setQuantity] = React.useState(1);
  const [unitCost, setUnitCost] = React.useState(0);
  const [detail, setDetail] = React.useState<Awaited<ReturnType<typeof getDetail>> | null>(null);
  const [feedback, setFeedback] = React.useState<string | null>(null);
  const [busy, setBusy] = React.useState(false);

  React.useEffect(() => {
    if (!selectedId) { setDetail(null); return; }
    void getDetail(selectedId).then(setDetail).catch((err: Error) => setFeedback(err.message));
  }, [getDetail, selectedId]);

  const chosenProduct = products.find((product) => product.id === productId);
  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!supplierId || !productId || quantity < 1 || unitCost < 0) return;
    setBusy(true); setFeedback(null);
    try {
      const created = await create({ supplier_id: supplierId, items: [{ product_id: productId, quantity, unit_cost: unitCost }] });
      setSelectedId(created.id);
      setFeedback(t("supplyChain.purchaseOrder.created", { number: created.po_number }));
    } catch (err: unknown) { setFeedback(err instanceof Error && err.message ? err.message : t("supplyChain.purchaseOrder.error")); }
    finally { setBusy(false); }
  };

  return (
    <div className="mx-auto max-w-7xl space-y-6">
      <header><h1 className="text-2xl font-bold">{t("supplyChain.purchaseOrder.title")}</h1><p className="text-sm text-[var(--color-text-secondary)]">{t("inventory.procurementDraft.description")}</p></header>
      {error && <div role="alert" className="rounded-xl border border-rose-200 bg-rose-50 p-3 text-sm text-rose-700">{error}</div>}
      {feedback && <div role="status" className="rounded-xl border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-700">{feedback}</div>}
      <section className="rounded-2xl border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] p-5 shadow-sm">
        <h2 className="mb-4 text-lg font-semibold">{t("supplyChain.purchaseOrder.create")}</h2>
        <form onSubmit={submit} className="grid gap-4 md:grid-cols-4">
          <label className="space-y-1 text-sm"><span>{t("supplyChain.purchaseOrder.supplier")}</span><select className="w-full rounded-lg border p-2" value={supplierId} onChange={(event) => setSupplierId(event.target.value)} required><option value="">{t("supplyChain.purchaseOrder.select")}</option>{suppliers.map((supplier) => <option key={supplier.id} value={supplier.id}>{supplier.company_name}</option>)}</select></label>
          <label className="space-y-1 text-sm"><span>{t("supplyChain.purchaseOrder.product")}</span><select className="w-full rounded-lg border p-2" value={productId} onChange={(event) => { setProductId(event.target.value); const product = products.find((item) => item.id === event.target.value); setUnitCost(product?.cost_price || 0); }} required><option value="">{t("supplyChain.purchaseOrder.select")}</option>{products.filter((product) => product.is_active).map((product) => <option key={product.id} value={product.id}>{product.sku} — {product.name}</option>)}</select></label>
          <label className="space-y-1 text-sm"><span>{t("supplyChain.purchaseOrder.quantity")}</span><input className="w-full rounded-lg border p-2" type="number" min="1" step="1" value={quantity} onChange={(event) => setQuantity(Number(event.target.value))} required /></label>
          <label className="space-y-1 text-sm"><span>{t("supplyChain.purchaseOrder.unitCost")}</span><input className="w-full rounded-lg border p-2" type="number" min="0" step="1" value={unitCost} onChange={(event) => setUnitCost(Number(event.target.value))} required /></label>
          <div className="md:col-span-4 flex items-center justify-between"><span className="text-sm">{chosenProduct?.name || ""} · {formatIDR(quantity * unitCost)}</span><button disabled={busy || loading} className="rounded-lg bg-[var(--color-action-primary)] px-4 py-2 text-sm font-semibold text-white disabled:opacity-50" type="submit">{busy ? t("common.actions.loading") : t("supplyChain.purchaseOrder.create")}</button></div>
        </form>
      </section>
      <section className="overflow-hidden rounded-2xl border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] shadow-sm"><div className="overflow-x-auto"><table className="w-full text-left text-sm"><thead className="bg-[var(--color-surface-muted)]"><tr><th className="p-3">{t("supplyChain.purchaseOrder.orderNumber")}</th><th className="p-3">{t("supplyChain.purchaseOrder.supplier")}</th><th className="p-3">{t("supplyChain.purchaseOrder.status")}</th><th className="p-3">{t("supplyChain.purchaseOrder.total")}</th></tr></thead><tbody>{orders.map((order) => <tr key={order.id} className="cursor-pointer border-t hover:bg-[var(--color-surface-muted)]" onClick={() => setSelectedId(order.id)}><td className="p-3 font-medium">{order.po_number}</td><td className="p-3">{order.supplier_name}</td><td className="p-3">{order.status}</td><td className="p-3">{formatIDR(order.total_amount)}</td></tr>)}</tbody></table></div>{!loading && orders.length === 0 && <p className="p-6 text-sm text-[var(--color-text-secondary)]">{t("supplyChain.purchaseOrder.empty")}</p>}</section>
      {detail && <section className="rounded-2xl border border-[var(--color-border-hairline)] bg-[var(--color-surface-base)] p-5 shadow-sm"><div className="flex items-start justify-between"><div><h2 className="text-lg font-semibold">{detail.po_number}</h2><p className="text-sm text-[var(--color-text-secondary)]">{detail.supplier_name} · {detail.status}</p></div>{["DRAFT", "ISSUED"].includes(detail.status) && <button className="rounded-lg border border-rose-300 px-3 py-2 text-sm text-rose-700" onClick={() => void cancel(detail.id, t("supplyChain.purchaseOrder.cancelReason")).then(() => setDetail(null)).catch((err: Error) => setFeedback(err.message))}>{t("supplyChain.purchaseOrder.cancel")}</button>}</div><div className="mt-4 overflow-x-auto"><table className="w-full text-left text-sm"><thead><tr><th className="p-2">{t("supplyChain.purchaseOrder.product")}</th><th className="p-2">{t("supplyChain.purchaseOrder.quantity")}</th><th className="p-2">{t("supplyChain.purchaseOrder.remaining")}</th><th className="p-2">{t("supplyChain.purchaseOrder.unitCost")}</th></tr></thead><tbody>{detail.items.map((item) => <tr key={item.id} className="border-t"><td className="p-2">{item.product_name} ({item.product_sku})</td><td className="p-2">{item.quantity}</td><td className="p-2">{item.remaining_quantity}</td><td className="p-2">{formatIDR(item.unit_cost)}</td></tr>)}</tbody></table></div></section>}
    </div>
  );
}
