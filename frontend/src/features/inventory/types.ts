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
  expected_quantity?: number;
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

export interface StockCardItem {
  movement_id: string;
  product_id: string;
  warehouse_location: string;
  quantity_delta: number;
  running_balance: number;
  movement_type: string;
  source_document_type: string;
  source_document_id?: string | null;
  source_document_line_id?: string | null;
  actor_id?: string | null;
  reason?: string | null;
  occurred_at: string;
  created_at: string;
}

export interface StockCardResponse {
  product_id: string;
  product_name: string;
  product_sku: string;
  warehouse_location: string;
  opening_balance: number;
  closing_balance: number;
  current_on_hand: number;
  movements: StockCardItem[];
  total_movements: number;
  limit: number;
  offset: number;
  start_date?: string | null;
  end_date?: string | null;
  as_of: string;
}

export interface StockOverviewCard {
  total_skus: number;
  total_units_on_hand: number;
  low_stock_skus: number;
  out_of_stock_skus: number;
  warehouse_location: string;
  as_of: string;
}
