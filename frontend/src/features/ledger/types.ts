export interface LedgerAccount {
  id: string;
  code: string;
  name: string;
  account_type: "ASSET" | "LIABILITY" | "EQUITY" | "REVENUE" | "EXPENSE" | string;
  is_zakat_eligible: boolean;
  is_active: boolean;
  created_at: string;
}

export interface LedgerEntryLine {
  id: string;
  ledger_entry_id: string;
  account_id: string;
  account_code?: string;
  account_name?: string;
  debit_amount: number;
  credit_amount: number;
}

export interface LedgerEntry {
  id: string;
  entry_number: string;
  entry_date: string;
  source_document_type: string;
  source_document_id?: string;
  memo: string;
  status: "POSTED" | "REVERSED" | string;
  reversed_by_entry_id?: string;
  created_at: string;
  total_debit: number;
  total_credit: number;
  lines?: LedgerEntryLine[];
}

export interface TrialBalanceRow {
  account_code: string;
  account_name: string;
  account_type: string;
  total_debit: number;
  total_credit: number;
  balance: number;
}

export interface TrialBalanceSummary {
  as_of_date: string;
  rows: TrialBalanceRow[];
  total_debits: number;
  total_credits: number;
  is_balanced: boolean;
}
