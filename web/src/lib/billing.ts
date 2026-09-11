export interface BillingAccount {
  auth_index: string;
  name: string;
  alias: string;
  plan: string;
  active_start: string | null;
  active_until: string | null;
}

export interface BillingMember {
  api_key_id: number;
  name: string;
  requests: number;
  success: number;
  failure: number;
  total_tokens: number;
  input_tokens: number;
  output_tokens: number;
  reasoning_tokens: number;
  cache_read_tokens: number;
  cost_usd: number;
  share: number;
  amount_cents: number;
}

export interface BillingBill {
  id: number;
  auth_index: string;
  cycle: number;
  account: string;
  alias: string;
  plan: string;
  window_start: string;
  window_end: string;
  fee_cents: number;
  fee_note: string;
  calculated_at: string;
  events: number;
  total_cost_usd: number;
  members: BillingMember[];
}

export interface SaveBillingBillRequest {
  auth_index: string;
  cycle: number;
  window_start: string;
  window_end: string;
  fee: string;
  fee_note: string;
}
