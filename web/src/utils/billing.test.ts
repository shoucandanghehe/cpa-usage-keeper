import { describe, expect, it } from 'vitest';
import type { BillingAccount, BillingBill } from '@/lib/billing';
import { addBillingCalendarMonth, billingDraftFromBill, billingDraftRequest, billingLocalDateTime, billingSnapshotCSV, billingTimestamp, newBillingDraft } from './billing';

const account: BillingAccount = {
  auth_index: 'account-a', name: 'Account A', alias: '', plan: 'Pro',
  active_start: '2026-08-03T10:11:12+08:00', active_until: '2026-09-03T10:11:12+08:00',
};
const bill: BillingBill = {
  id: 7, auth_index: account.auth_index, account: account.name, alias: '', plan: 'Pro', cycle: 4,
  window_start: '2026-07-31T10:11:12+08:00', window_end: '2026-08-31T10:11:12+08:00',
  fee_cents: 101, fee_note: 'Snapshot fee', calculated_at: '2026-09-01T03:04:05Z', events: 2, total_cost_usd: 3,
  members: [
    { api_key_id: 1, name: 'Participant A', requests: 1, success: 1, failure: 0, total_tokens: 10, input_tokens: 6, output_tokens: 4, reasoning_tokens: 1, cache_read_tokens: 2, cost_usd: 2, share: 2 / 3, amount_cents: 67 },
    { api_key_id: 2, name: 'Participant B', requests: 1, success: 0, failure: 1, total_tokens: 5, input_tokens: 3, output_tokens: 2, reasoning_tokens: 0, cache_read_tokens: 1, cost_usd: 1, share: 1 / 3, amount_cents: 34 },
  ],
};

describe('billing dates and snapshots', () => {
  it('clamps a calendar month at leap-year month-end and preserves local seconds', () => {
    const next = new Date(addBillingCalendarMonth(new Date(2024, 0, 31, 12, 34, 56).toISOString()));
    expect([next.getFullYear(), next.getMonth(), next.getDate(), next.getHours(), next.getMinutes(), next.getSeconds()]).toEqual([2024, 1, 29, 12, 34, 56]);
    const yearEnd = new Date(addBillingCalendarMonth(new Date(2025, 11, 31, 12, 34, 56).toISOString()));
    expect([yearEnd.getFullYear(), yearEnd.getMonth(), yearEnd.getDate()]).toEqual([2026, 0, 31]);
  });

  it('preserves saved offset instants while converting edited local timestamps without losing seconds', () => {
    const saved = '2026-11-01T01:22:33.123-05:00';
    expect(billingTimestamp(billingLocalDateTime(saved), saved)).toBe(saved);
    const local = new Date(2026, 6, 14, 17, 23, 49, 123);
    expect(billingTimestamp(billingLocalDateTime(local.toISOString()))).toBe(local.toISOString());
    expect(billingTimestamp('2026-02-30T12:00:00')).toBeNull();
  });

  it('suggests independent initial windows and derives the next cycle from saved bounds rather than live account changes', () => {
    const first = newBillingDraft(account, []);
    expect(billingDraftRequest(account.auth_index, { ...first, fee: '841.22' })).toMatchObject({ cycle: 1, window_start: account.active_start, window_end: account.active_until });
    const next = newBillingDraft({ ...account, active_start: '2030-01-01T00:00:00Z' }, [bill]);
    const request = billingDraftRequest(account.auth_index, next)!;
    expect(request.cycle).toBe(5);
    expect(request.window_start).toBe(bill.window_end);
    expect(request.window_end).toBe(addBillingCalendarMonth(bill.window_end));
    expect(request.fee).toBe('1.01');
  });

  it('rejects unsafe cycle numbers, empty windows, and sub-cent fees before saving', () => {
    const draft = billingDraftFromBill(bill);
    expect(billingDraftRequest(account.auth_index, { ...draft, cycle: '9007199254740992' })).toBeNull();
    expect(billingDraftRequest(account.auth_index, { ...draft, end: draft.start, originalEnd: undefined })).toBeNull();
    expect(billingDraftRequest(account.auth_index, { ...draft, fee: '0.001' })).toBeNull();
  });

  it('exports persisted allocations verbatim with BOM, CSV escaping, and formula protection', () => {
    const csv = billingSnapshotCSV({
      ...bill, account: '=HYPERLINK("https://invalid")', fee_note: ' \t+SUM(1,2)\nsecond line',
      members: [{ ...bill.members[0], name: '@SUM(1,2)' }, { ...bill.members[1], name: 'A "quoted", name' }],
    });
    expect(csv.startsWith('\uFEFF')).toBe(true);
    expect(csv).toContain('"\'=HYPERLINK(""https://invalid"")"');
    expect(csv).toContain('"\' \t+SUM(1,2)\nsecond line"');
    expect(csv).toContain('"\'@SUM(1,2)"');
    expect(csv).toContain('"A ""quoted"", name"');
    expect(csv).toContain(`"2.0000","${2 / 3}","0.67"\r\n`);
    expect(csv).toContain(`"1.0000","${1 / 3}","0.34"\r\n`);
    expect(csv).toContain(`"${bill.window_start}","${bill.window_end}","1.01"`);
  });
});
