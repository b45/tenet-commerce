import type { InventoryProduct } from "@/features/inventory/types";

export interface PurchaseOrderSummary {
  id: string;
  po_number: string;
  supplier_id: string;
  supplier_name: string;
  total_amount: number;
  status: string;
  issued_date: string;
  item_count: number;
}

export interface PurchaseOrderLine {
  id: string;
  product_id: string;
  product_name: string;
  product_sku: string;
  quantity: number;
  received_quantity: number;
  remaining_quantity: number;
  unit_cost: number;
  subtotal: number;
}

export interface PurchaseOrderDetail extends PurchaseOrderSummary {
  compliance_cert_id?: string | null;
  items: PurchaseOrderLine[];
  goods_receipts: Array<{ id: string; gr_number: string; received_date: string }>;
}

export interface PurchaseOrderDraft {
  supplier_id: string;
  compliance_cert_id?: string;
  items: Array<{ product_id: string; quantity: number; unit_cost: number }>;
}

export type { InventoryProduct };
