/**
 * Tenet Commerce — Product Inventory Domain Types
 * Contracts aligned with Go Backend: backend/internal/pos/models.go
 */

export interface InventoryProduct {
  id: string;
  category_id?: string | null;
  category_name?: string;
  sku: string;
  barcode?: string | null;
  name: string;
  description?: string | null;
  unit_price: number;
  cost_price: number;
  stock_quantity: number;
  reorder_threshold?: number;
  warehouse_location?: string;
  compliance_tags?: string[];
  is_halal_certified: boolean;
  is_active: boolean;
  created_at?: string;
  updated_at?: string;
}

export interface Category {
  id: string;
  name: string;
  code?: string;
  product_count?: number;
}

export type AdjustmentType = "ADD" | "SUBTRACT" | "SET";

export type AdjustmentReason =
  | "DAMAGE"
  | "EXPIRED"
  | "AUDIT_CORRECTION"
  | "RESTOCK"
  | "OTHER";

export interface StockAdjustmentPayload {
  product_id: string;
  adjustment_type: AdjustmentType;
  quantity: number;
  reason: AdjustmentReason;
  notes?: string;
}

export interface StockAdjustmentResponse {
  adjustment_id: string;
  product_id: string;
  product_name: string;
  previous_quantity: number;
  new_quantity: number;
  quantity_delta: number;
  reason: string;
  ledger_entry_number?: string;
  adjusted_at: string;
}

export interface CreateProductPayload {
  name: string;
  sku: string;
  barcode?: string;
  description?: string;
  category_id?: string;
  unit_price: number;
  cost_price: number;
  initial_stock: number;
  reorder_threshold: number;
  warehouse_location?: string;
  compliance_tags?: string[];
  is_active?: boolean;
}

export interface UpdateProductPayload {
  name: string;
  barcode?: string;
  description?: string;
  category_id?: string;
  unit_price: number;
  cost_price: number;
  reorder_threshold: number;
  warehouse_location?: string;
  compliance_tags?: string[];
  is_active?: boolean;
}

export type StockStatusFilter = "all" | "low_stock" | "out_of_stock";

export interface InventoryFilter {
  search: string;
  category_id: string;
  stock_status: StockStatusFilter;
}
