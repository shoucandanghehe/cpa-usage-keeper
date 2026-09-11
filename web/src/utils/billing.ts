import type { BillingAccount, BillingBill, SaveBillingBillRequest } from '@/lib/billing';

export interface BillingDraft {
  cycle: string;
  start: string;
  end: string;
  fee: string;
  feeNote: string;
  originalStart?: string;
  originalEnd?: string;
}

const pad = (value: number, width = 2) => String(value).padStart(width, '0');

export function billingLocalDateTime(timestamp: string): string {
  const date = new Date(timestamp);
  if (!Number.isFinite(date.getTime())) return '';
  const seconds = `${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`;
  const milliseconds = date.getMilliseconds() ? `.${pad(date.getMilliseconds(), 3)}` : '';
  return `${pad(date.getFullYear(), 4)}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${seconds}${milliseconds}`;
}

export function billingTimestamp(value: string, original?: string): string | null {
  // Preserve the saved instant, including the offset during a repeated DST hour.
  if (original && value === billingLocalDateTime(original)) return original;
  const match = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})(?::(\d{2})(?:\.(\d{1,3}))?)?$/.exec(value);
  if (!match) return null;
  const date = new Date(value);
  const expected = [Number(match[1]), Number(match[2]) - 1, Number(match[3]), Number(match[4]), Number(match[5]), Number(match[6] ?? 0), Number((match[7] ?? '').padEnd(3, '0'))];
  const actual = [date.getFullYear(), date.getMonth(), date.getDate(), date.getHours(), date.getMinutes(), date.getSeconds(), date.getMilliseconds()];
  // Browsers otherwise normalize invalid dates and nonexistent local DST times.
  if (actual.some((part, index) => part !== expected[index])) return null;
  return date.toISOString();
}

export function addBillingCalendarMonth(timestamp: string): string {
  const date = new Date(timestamp);
  if (!Number.isFinite(date.getTime())) return '';
  const day = date.getDate();
  date.setDate(1);
  date.setMonth(date.getMonth() + 1);
  const lastDay = new Date(date.getFullYear(), date.getMonth() + 1, 0).getDate();
  date.setDate(Math.min(day, lastDay));
  return date.toISOString();
}

export const billingYuan = (cents: number): string => (cents / 100).toFixed(2);

export function billingDraftFromBill(bill: BillingBill): BillingDraft {
  return {
    cycle: String(bill.cycle),
    start: billingLocalDateTime(bill.window_start),
    end: billingLocalDateTime(bill.window_end),
    fee: billingYuan(bill.fee_cents),
    feeNote: bill.fee_note,
    originalStart: bill.window_start,
    originalEnd: bill.window_end,
  };
}

export function newBillingDraft(account: BillingAccount, bills: readonly BillingBill[]): BillingDraft {
  const previous = bills.reduce<BillingBill | undefined>((latest, bill) => !latest || bill.cycle > latest.cycle ? bill : latest, undefined);
  const start = previous?.window_end ?? account.active_start ?? '';
  const end = previous ? addBillingCalendarMonth(start) : account.active_until ?? (start ? addBillingCalendarMonth(start) : '');
  return {
    cycle: String((previous?.cycle ?? 0) + 1),
    start: billingLocalDateTime(start),
    end: billingLocalDateTime(end),
    fee: previous ? billingYuan(previous.fee_cents) : '',
    feeNote: '',
    originalStart: start || undefined,
    originalEnd: end || undefined,
  };
}

export function billingDraftRequest(authIndex: string, draft: BillingDraft): SaveBillingBillRequest | null {
  const cycle = Number(draft.cycle);
  const start = billingTimestamp(draft.start, draft.originalStart);
  const end = billingTimestamp(draft.end, draft.originalEnd);
  const fee = draft.fee.trim();
  if (!/^\d+$/.test(draft.cycle) || !Number.isSafeInteger(cycle) || cycle <= 0
    || !start || !end || new Date(start).getTime() >= new Date(end).getTime()
    || !/^\d{1,9}(?:\.\d{1,2})?$/.test(fee) || Number(fee) <= 0 || Number(fee) > 999999999.99) return null;
  return { auth_index: authIndex, cycle, window_start: start, window_end: end, fee, fee_note: draft.feeNote };
}

const csvCell = (value: string | number): string => {
  let text = String(value);
  if (typeof value === 'string') {
    let first = 0;
    while (first < text.length && (text.charCodeAt(first) < 32 || /\s/u.test(text[first]))) first++;
    if ((first < text.length && '=+-@'.includes(text[first])) || ['\t', '\r', '\n'].includes(text[0])) text = `'${text}`;
  }
  return `"${text.replace(/"/g, '""')}"`;
};

export function billingSnapshotCSV(bill: BillingBill): string {
  const rows: (string | number)[][] = [[
    'bill_id', 'auth_index', 'account', 'alias', 'plan', 'cycle', 'window_start_inclusive', 'window_end_exclusive',
    'fee_cny', 'fee_note', 'calculated_at', 'events', 'total_cost_usd', 'api_key_id', 'name', 'requests',
    'success', 'failure', 'total_tokens', 'input_tokens', 'output_tokens', 'reasoning_tokens', 'cache_read_tokens',
    'cache_read_ratio', 'cost_usd', 'share', 'amount_cny',
  ]];
  for (const member of bill.members) {
    rows.push([
      bill.id, bill.auth_index, bill.account, bill.alias, bill.plan, bill.cycle, bill.window_start, bill.window_end,
      billingYuan(bill.fee_cents), bill.fee_note, bill.calculated_at, bill.events, bill.total_cost_usd.toFixed(4),
      member.api_key_id, member.name, member.requests, member.success, member.failure, member.total_tokens,
      member.input_tokens, member.output_tokens, member.reasoning_tokens, member.cache_read_tokens,
      member.input_tokens > 0 ? member.cache_read_tokens / member.input_tokens : '',
      member.cost_usd.toFixed(4), member.share, billingYuan(member.amount_cents),
    ]);
  }
  return `\uFEFF${rows.map((row) => row.map(csvCell).join(',')).join('\r\n')}\r\n`;
}
