/**
 * Tenet Commerce — POS Domain Types & Interfaces
 * Matches backend contracts specified in docs/design/POS_CONTRACT_MAP.md
 */

export interface Product {
  id: string;
  category_id?: string;
  category_name?: string;
  sku: string;
  barcode: string;
  name: string;
  description?: string;
  unit_price: number;
  cost_price?: number;
  stock_quantity: number;
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

export interface CartItem {
  product: Product;
  quantity: number;
  subtotal: number;
}

export interface CheckoutItemInput {
  sku: string;
  quantity: number;
}

export interface CheckoutRequest {
  items: CheckoutItemInput[];
  payment_method: "CASH";
  cash_tendered: number;
  discount_amount: number;
}

export interface CheckoutResponseItem {
  id?: string;
  transaction_id?: string;
  product_id: string;
  sku: string;
  name?: string;
  product_name?: string;
  quantity: number;
  unit_price: number;
  subtotal?: number;
  subtotal_amount?: number;
}

export interface CheckoutResponse {
  transaction_id: string;
  transaction_number: string;
  status: "COMPLETED" | "PENDING" | "FAILED" | "CANCELLED";
  cashier_id: string;
  subtotal_amount: number;
  tax_amount: number;
  discount_amount: number;
  total_amount: number;
  cash_tendered: number;
  change_amount: number;
  created_at: string;
  items: CheckoutResponseItem[];
}

export interface OrderItem {
  id: string;
  product_id: string;
  sku: string;
  name?: string;
  product_name?: string;
  quantity: number;
  unit_price: number;
  subtotal?: number;
  subtotal_amount?: number;
}

export interface Order {
  id: string;
  transaction_number: string;
  cashier_id: string;
  status: "COMPLETED" | "VOIDED" | "PENDING" | "CANCELLED";
  payment_method: string;
  subtotal_amount: number;
  tax_amount: number;
  discount_amount: number;
  total_amount: number;
  cash_tendered: number;
  change_amount: number;
  void_reason?: string;
  voided_at?: string;
  created_at: string;
  items?: OrderItem[];
}

export interface OrderDetailResponse {
  transaction: Order;
  items: OrderItem[];
}

export type POSViewMode = "register" | "history";

export type CheckoutStep =
  | "idle"
  | "review"
  | "submitting"
  | "completed"
  | "rejected"
  | "unknown_error";
