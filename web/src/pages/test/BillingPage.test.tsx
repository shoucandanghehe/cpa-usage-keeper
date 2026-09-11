// @vitest-environment happy-dom

import { act, createRef, StrictMode } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import i18n from '@/i18n';
import type { BillingAccount, BillingBill } from '@/lib/billing';
import type * as API from '@/lib/api';

const api = vi.hoisted(() => ({ fetchBillingAccounts: vi.fn(), fetchBillingBills: vi.fn(), saveBillingBill: vi.fn(), deleteBillingBill: vi.fn() }));
vi.mock('@/lib/api', async (importOriginal) => ({ ...await importOriginal<typeof API>(), ...api }));
import { BillingPage } from '../BillingPage';

const accounts: BillingAccount[] = ['A', 'B'].map((label) => ({
  auth_index: `account-${label}`, name: `Account ${label}`, alias: '', plan: 'Pro', active_start: '2026-07-01T00:00:01Z', active_until: '2026-08-01T00:00:01Z',
}));
const saved: BillingBill = {
  id: 1, auth_index: accounts[0].auth_index, account: accounts[0].name, alias: '', plan: 'Pro', cycle: 1,
  window_start: '2026-07-01T00:00:01Z', window_end: '2026-08-01T00:00:01Z', calculated_at: '2026-08-02T00:00:02Z',
  fee_cents: 101, fee_note: 'Saved fee note', events: 2, total_cost_usd: 3,
  members: [
    { api_key_id: 1, name: 'Participant A', requests: 1, success: 1, failure: 0, total_tokens: 10, input_tokens: 6, output_tokens: 4, reasoning_tokens: 1, cache_read_tokens: 2, cost_usd: 2, share: 2 / 3, amount_cents: 67 },
    { api_key_id: 2, name: 'Participant B', requests: 1, success: 0, failure: 1, total_tokens: 5, input_tokens: 3, output_tokens: 2, reasoning_tokens: 0, cache_read_tokens: 1, cost_usd: 1, share: 1 / 3, amount_cents: 34 },
  ],
};

function button(label: string, scope: ParentNode = document): HTMLButtonElement {
  const result = Array.from(scope.querySelectorAll<HTMLButtonElement>('button')).find((node) => node.textContent?.trim() === label);
  if (!result) throw new Error(`Missing button ${label}`);
  return result;
}

async function chooseAccount(name: string) {
  await act(async () => document.querySelector<HTMLButtonElement>('button[aria-haspopup="listbox"]')!.click());
  const option = Array.from(document.querySelectorAll<HTMLButtonElement>('[role="option"]')).find((node) => node.textContent?.includes(name));
  if (!option) throw new Error(`Missing account ${name}`);
  await act(async () => option.click());
}

describe('BillingPage snapshot lifecycle', () => {
  let root: Root;
  let container: HTMLDivElement;
  const refreshRef = createRef<(() => Promise<void>) | null>();
  const download = vi.fn();
  const render = async (active = true) => {
    await act(async () => root.render(<StrictMode><BillingPage active={active} refreshRef={refreshRef} onDownload={download} /></StrictMode>));
  };

  beforeEach(async () => {
    globalThis.IS_REACT_ACT_ENVIRONMENT = true;
    await i18n.changeLanguage('en');
    Object.values(api).forEach((mock) => mock.mockReset());
    download.mockReset();
    api.fetchBillingAccounts.mockResolvedValue({ accounts });
    api.fetchBillingBills.mockResolvedValue({ bills: [saved] });
    container = document.createElement('div');
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(async () => {
    await act(async () => root.unmount());
    container.remove();
    vi.restoreAllMocks();
  });

  it('retains the snapshot on failed recalculation and replaces it only after successful save', async () => {
    await render();
    expect(container.querySelector('table')?.textContent).toContain('CNY 0.67');
    const failedSave = Promise.withResolvers<BillingBill>();
    api.saveBillingBill.mockReturnValueOnce(failedSave.promise);
    await act(async () => container.querySelector('form')!.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true })));
    await act(async () => button('Recalculate and save', document.querySelector('[role="dialog"]')!).click());
    expect(container.querySelector<HTMLButtonElement>('button[aria-haspopup="listbox"]')!.disabled).toBe(true);
    expect(container.querySelector('table')?.textContent).toContain('CNY 0.67');
    await act(async () => failedSave.reject(new Error('Missing model pricing')));
    expect(container.querySelector('[role="alert"]')?.textContent).toBe('Missing model pricing');
    expect(container.querySelector('table')?.textContent).toContain('CNY 0.67');

    const successfulSave = Promise.withResolvers<BillingBill>();
    api.saveBillingBill.mockReturnValueOnce(successfulSave.promise);
    await act(async () => container.querySelector('form')!.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true })));
    await act(async () => button('Recalculate and save', document.querySelector('[role="dialog"]')!).click());
    expect(container.querySelector('table')?.textContent).toContain('CNY 0.67');
    await act(async () => successfulSave.resolve({ ...saved, members: saved.members.map((member, index) => ({ ...member, amount_cents: index === 0 ? 50 : 51 })) }));
    expect(container.querySelector('table')?.textContent).toContain('CNY 0.50');
    expect(container.querySelector('table')?.textContent).not.toContain('CNY 0.67');
  });

  it('keeps an account snapshot when a late response belongs to another account and when changing tabs', async () => {
    await render();
    const otherAccount = Promise.withResolvers<{ bills: BillingBill[] }>();
    api.fetchBillingBills.mockReturnValueOnce(otherAccount.promise);
    await chooseAccount('Account B');
    expect(container.querySelector('table')).toBeNull();
    await chooseAccount('Account A');
    expect(container.querySelector('table')?.textContent).toContain('CNY 0.67');
    await act(async () => otherAccount.resolve({ bills: [{ ...saved, id: 2, auth_index: accounts[1].auth_index, account: accounts[1].name, cycle: 7, members: [{ ...saved.members[0], name: 'Other account participant', amount_cents: 101 }] }] }));
    expect(container.querySelector('table')?.textContent).not.toContain('Other account participant');
    await render(false);
    await render(true);
    expect(container.querySelector('table')?.textContent).toContain('CNY 0.67');
    await chooseAccount('Account B');
    expect(container.querySelector('table')?.textContent).toContain('Other account participant');
    expect(container.querySelector<HTMLInputElement>('input[type="number"]')?.value).toBe('7');
  });

  it('refreshes the list without silently replacing the selected snapshot and retains it on refresh failure', async () => {
    await render();
    api.fetchBillingBills.mockResolvedValue({ bills: [{ ...saved, fee_note: 'Changed in another session', members: [] }] });
    await act(async () => refreshRef.current?.());
    expect(container.querySelector('table')?.textContent).toContain('Participant A');
    expect(container.textContent).toContain('Saved fee note');
    expect(container.textContent).not.toContain('Changed in another session');
    api.fetchBillingBills.mockRejectedValue(new Error('Connection failed'));
    await act(async () => refreshRef.current?.());
    expect(container.querySelector('[role="alert"]')?.textContent).toBe('Connection failed');
    expect(container.querySelector('table')?.textContent).toContain('Participant A');
    await act(async () => button('Export snapshot CSV').click());
    const [blob] = download.mock.calls[0] as [Blob, string];
    expect(await blob.text()).toContain('Saved fee note');
    expect(await blob.text()).not.toContain('Changed in another session');
  });

  it('requires confirmation before deletion and preserves the snapshot when deletion fails', async () => {
    await render();
    api.deleteBillingBill.mockRejectedValue(new Error('Delete rejected'));
    await act(async () => button('Delete').click());
    await act(async () => button('Cancel', document.querySelector('[role="dialog"]')!).click());
    expect(container.querySelector('table')?.textContent).toContain('Participant A');
    await act(async () => button('Delete', container).click());
    await act(async () => button('Delete', document.querySelector('[role="dialog"]')!).click());
    expect(container.querySelector('[role="alert"]')?.textContent).toBe('Delete rejected');
    expect(container.querySelector('table')?.textContent).toContain('Participant A');
  });
});
