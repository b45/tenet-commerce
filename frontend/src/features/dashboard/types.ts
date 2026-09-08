export interface SalesSummary {
  today_gross_sales: number;
  today_net_sales: number;
  today_orders_count: number;
  all_time_orders_count: number;
  average_order_value: number;
}

export interface LowStockItem {
  product_id: string;
  sku: string;
  name: string;
  category_name: string;
  current_stock: number;
  threshold: number;
  unit_price: number;
}

export interface InventoryAlerts {
  low_stock_count: number;
  items: LowStockItem[];
}

export interface CertificateAlertItem {
  certificate_id: string;
  supplier_id: string;
  supplier_name: string;
  certificate_number: string;
  issuing_authority: string;
  expiry_date: string;
  days_remaining: number;
  status: "EXPIRING_SOON" | "EXPIRED" | "REVOKED";
}

export interface ComplianceAlerts {
  expiring_certificates_count: number;
  expired_certificates_count: number;
  items: CertificateAlertItem[];
}

export interface FinancialSummary {
  active_accounts_count: number;
  today_journal_entries_count: number;
}

export interface DashboardSummary {
  generated_at: string;
  sales_summary: SalesSummary;
  inventory_alerts: InventoryAlerts;
  compliance_alerts: ComplianceAlerts;
  financial_summary: FinancialSummary;
}
