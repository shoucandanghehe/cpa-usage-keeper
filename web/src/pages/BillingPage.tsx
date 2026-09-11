import { useCallback, useEffect, useId, useMemo, useRef, useState, type FormEvent, type RefObject } from 'react';
import { useTranslation } from 'react-i18next';
import { ApiError, deleteBillingBill, fetchBillingAccounts, fetchBillingBills, saveBillingBill } from '@/lib/api';
import type { BillingAccount, BillingBill, SaveBillingBillRequest } from '@/lib/billing';
import { billingDraftFromBill, billingDraftRequest, billingSnapshotCSV, billingYuan, newBillingDraft, type BillingDraft } from '@/utils/billing';
import { Button } from '@/components/ui/Button';
import { Card } from '@/components/ui/Card';
import { Input } from '@/components/ui/Input';
import { Modal } from '@/components/ui/Modal';
import { Select } from '@/components/ui/Select';
import styles from './BillingPage.module.scss';

type AccountBills = {
  bills: BillingBill[];
  draft: BillingDraft;
  editingId: number | null;
  snapshot: BillingBill | null;
};
type Confirmation = { kind: 'save'; request: SaveBillingBillRequest } | { kind: 'delete'; bill: BillingBill };

interface BillingPageProps {
  active: boolean;
  onAuthRequired?: () => void;
  refreshRef: RefObject<(() => Promise<void>) | null>;
  onDownload: (blob: Blob, filename: string) => void;
}

export function BillingPage({ active, onAuthRequired, refreshRef, onDownload }: BillingPageProps) {
  const { t, i18n } = useTranslation();
  const accountLabelId = useId();
  const noteId = useId();
  const [accounts, setAccounts] = useState<BillingAccount[]>([]);
  const [accountsLoaded, setAccountsLoaded] = useState(false);
  const [accountsLoading, setAccountsLoading] = useState(false);
  const [accountError, setAccountError] = useState('');
  const [selected, setSelected] = useState('');
  const [entries, setEntries] = useState<Record<string, AccountBills>>({});
  const [billsLoading, setBillsLoading] = useState('');
  const [billError, setBillError] = useState<{ authIndex: string; message: string } | null>(null);
  const [notice, setNotice] = useState<{ authIndex: string; message: string } | null>(null);
  const [busy, setBusy] = useState<'save' | 'delete' | null>(null);
  const [confirmation, setConfirmation] = useState<Confirmation | null>(null);
  const busyRef = useRef(false);
  const mountedRef = useRef(true);
  const loadedAccountsRef = useRef(new Set<string>());
  const accountsControllerRef = useRef<AbortController | null>(null);
  const billsControllerRef = useRef<AbortController | null>(null);
  const account = accounts.find((item) => item.auth_index === selected);
  const entry = Object.hasOwn(entries, selected) ? entries[selected] : undefined;
  const snapshot = entry?.snapshot;
  const locked = busy !== null || confirmation !== null || accountsLoading || Boolean(selected && billsLoading === selected);
  const timeZone = Intl.DateTimeFormat().resolvedOptions().timeZone;
  const numberFormat = useMemo(() => new Intl.NumberFormat(i18n.language), [i18n.language]);
  const costFormat = useMemo(() => new Intl.NumberFormat(i18n.language, { minimumFractionDigits: 4, maximumFractionDigits: 4 }), [i18n.language]);
  const percentFormat = useMemo(() => new Intl.NumberFormat(i18n.language, { style: 'percent', minimumFractionDigits: 2, maximumFractionDigits: 2 }), [i18n.language]);
  const dateFormat = useMemo(() => new Intl.DateTimeFormat(i18n.language, {
    year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit', second: '2-digit', hourCycle: 'h23', timeZoneName: 'longOffset',
  }), [i18n.language]);

  const errorMessage = useCallback((error: unknown) => {
    if (error instanceof ApiError && error.status === 401) {
      onAuthRequired?.();
      return t('auth.session_expired');
    }
    return error instanceof Error ? error.message : t('billing.request_failed');
  }, [onAuthRequired, t]);

  const loadAccounts = useCallback(async () => {
    if (busyRef.current) return;
    accountsControllerRef.current?.abort();
    const controller = new AbortController();
    accountsControllerRef.current = controller;
    setAccountsLoading(true);
    setAccountError('');
    try {
      const response = await fetchBillingAccounts(controller.signal);
      if (controller.signal.aborted || !mountedRef.current) return;
      // Keep historical options available if a live account disappears after loading.
      setAccounts((current) => [...response.accounts, ...current.filter((item) => !response.accounts.some((next) => next.auth_index === item.auth_index))]);
      setAccountsLoaded(true);
      setSelected((current) => current || response.accounts[0]?.auth_index || '');
    } catch (error) {
      if (!controller.signal.aborted && mountedRef.current) setAccountError(errorMessage(error));
    } finally {
      if (accountsControllerRef.current === controller) {
        accountsControllerRef.current = null;
        if (mountedRef.current) setAccountsLoading(false);
      }
    }
  }, [errorMessage]);

  const loadBills = useCallback(async (target: BillingAccount) => {
    if (busyRef.current) return;
    billsControllerRef.current?.abort();
    const controller = new AbortController();
    billsControllerRef.current = controller;
    const authIndex = target.auth_index;
    setBillsLoading(authIndex);
    setBillError(null);
    try {
      const response = await fetchBillingBills(authIndex, controller.signal);
      if (controller.signal.aborted || !mountedRef.current) return;
      const bills = [...response.bills].sort((a, b) => b.cycle - a.cycle);
      loadedAccountsRef.current.add(authIndex);
      setEntries((current) => {
        const existing = Object.hasOwn(current, authIndex) ? current[authIndex] : undefined;
        const first = bills[0];
        return { ...current, [authIndex]: existing
          ? { ...existing, bills }
          : { bills, draft: first ? billingDraftFromBill(first) : newBillingDraft(target, bills), editingId: first?.id ?? null, snapshot: first ?? null } };
      });
    } catch (error) {
      if (!controller.signal.aborted && mountedRef.current) setBillError({ authIndex, message: errorMessage(error) });
    } finally {
      if (billsControllerRef.current === controller) {
        billsControllerRef.current = null;
        if (mountedRef.current) setBillsLoading('');
      }
    }
  }, [errorMessage]);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
      accountsControllerRef.current?.abort();
      billsControllerRef.current?.abort();
    };
  }, []);

  useEffect(() => {
    if (active) void loadAccounts();
  }, [active, loadAccounts]);

  useEffect(() => {
    // Responses are cached under their own account; changing selection never exposes another account's snapshot.
    if (active && account && !loadedAccountsRef.current.has(account.auth_index)) void loadBills(account);
  }, [active, account, loadBills]);

  useEffect(() => {
    const refresh = async () => {
      if (busyRef.current || confirmation) return;
      await Promise.all([loadAccounts(), account ? loadBills(account) : Promise.resolve()]);
    };
    refreshRef.current = refresh;
    return () => { if (refreshRef.current === refresh) refreshRef.current = null; };
  }, [account, confirmation, loadAccounts, loadBills, refreshRef]);

  const changeDraft = (field: keyof BillingDraft, value: string) => {
    if (locked || !entry) return;
    setEntries((current) => ({ ...current, [selected]: { ...current[selected], draft: { ...current[selected].draft, [field]: value } } }));
    setNotice(null);
  };

  const selectBill = (bill: BillingBill) => {
    if (locked) return;
    setEntries((current) => ({ ...current, [selected]: { ...current[selected], draft: billingDraftFromBill(bill), editingId: bill.id, snapshot: bill } }));
    setBillError(null);
    setNotice(null);
  };

  const beginNewCycle = () => {
    if (locked || !entry || !account) return;
    setEntries((current) => ({ ...current, [selected]: { ...current[selected], draft: newBillingDraft(account, current[selected].bills), editingId: null } }));
    setBillError(null);
    setNotice(null);
  };

  const commitBill = async (request: SaveBillingBillRequest) => {
    if (busyRef.current) return;
    busyRef.current = true;
    accountsControllerRef.current?.abort();
    billsControllerRef.current?.abort();
    setBusy('save');
    setBillError(null);
    setNotice(null);
    try {
      const bill = await saveBillingBill(request);
      if (!mountedRef.current) return;
      loadedAccountsRef.current.add(request.auth_index);
      setEntries((current) => ({ ...current, [request.auth_index]: {
        bills: [bill, ...current[request.auth_index].bills.filter((item) => item.id !== bill.id && item.cycle !== bill.cycle)].sort((a, b) => b.cycle - a.cycle),
        draft: billingDraftFromBill(bill), editingId: bill.id, snapshot: bill,
      } }));
      setNotice({ authIndex: request.auth_index, message: t('billing.saved') });
    } catch (error) {
      if (mountedRef.current) setBillError({ authIndex: request.auth_index, message: errorMessage(error) });
    } finally {
      busyRef.current = false;
      if (mountedRef.current) { setBusy(null); setConfirmation(null); }
    }
  };

  const submitBill = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (locked || busyRef.current || !entry) return;
    const request = billingDraftRequest(selected, entry.draft);
    if (!request) {
      setBillError({ authIndex: selected, message: t('billing.invalid_form') });
      return;
    }
    if (entry.bills.some((bill) => bill.cycle === request.cycle) || entry.editingId !== null) setConfirmation({ kind: 'save', request });
    else void commitBill(request);
  };

  const removeBill = async (bill: BillingBill) => {
    const target = accounts.find((item) => item.auth_index === bill.auth_index);
    if (busyRef.current || !target) return;
    busyRef.current = true;
    accountsControllerRef.current?.abort();
    billsControllerRef.current?.abort();
    setBusy('delete');
    setBillError(null);
    setNotice(null);
    try {
      await deleteBillingBill(bill.id);
      if (!mountedRef.current) return;
      setEntries((current) => {
        const existing = current[bill.auth_index];
        const bills = existing.bills.filter((item) => item.id !== bill.id);
        const next = bills[0];
        return { ...current, [bill.auth_index]: {
          bills,
          draft: next ? billingDraftFromBill(next) : newBillingDraft(target, bills),
          editingId: next?.id ?? null,
          snapshot: next ?? null,
        } };
      });
      setNotice({ authIndex: bill.auth_index, message: t('billing.deleted') });
    } catch (error) {
      if (mountedRef.current) setBillError({ authIndex: bill.auth_index, message: errorMessage(error) });
    } finally {
      busyRef.current = false;
      if (mountedRef.current) { setBusy(null); setConfirmation(null); }
    }
  };

  if (!active) return null;
  const existingCycle = entry && (entry.editingId !== null || entry.bills.some((bill) => bill.cycle === Number(entry.draft.cycle)));
  const currentError = billError?.authIndex === selected ? billError.message : '';

  return (
    <section className={styles.page} aria-label={t('billing.title')}>
      <header className={styles.intro}>
        <h2>{t('billing.title')}</h2>
        <p>{t('billing.description')}</p>
        <p className={styles.hint}>{t('billing.data_limitation')}</p>
      </header>
      {accountError && <div className="error-box" role="alert">{accountError}</div>}
      <Card title={t('billing.accounts')} extra={<Button variant="secondary" disabled={locked} onClick={() => void refreshRef.current?.()}>{t('usage_stats.refresh')}</Button>}>
        <div className={styles.accountSelector}>
          <label id={accountLabelId}>{t('billing.account')}</label>
          <Select
            value={selected}
            options={accounts.map((item) => ({ value: item.auth_index, label: `${item.alias || item.name || item.auth_index}${item.plan ? ` · ${item.plan}` : ''} · ${item.auth_index}` }))}
            onChange={(value) => { if (!busyRef.current && !confirmation && !accountsLoading) { setSelected(value); setNotice(null); } }}
            placeholder={accountsLoading ? t('common.loading') : t('billing.select_account')}
            ariaLabelledBy={accountLabelId}
            disabled={busy !== null || confirmation !== null || accountsLoading || accounts.length === 0}
            fullWidth
          />
          {account && <p className={styles.hint}>{account.name} · {account.auth_index}{account.plan ? ` · ${account.plan}` : ''}</p>}
        </div>
        {accountsLoaded && accounts.length === 0 && <p>{t('billing.no_accounts')}</p>}
      </Card>
      {currentError && <div className="error-box" role="alert">{currentError}</div>}
      {notice?.authIndex === selected && <p className={styles.notice} role="status">{notice.message}</p>}
      {billsLoading === selected && selected && <p role="status">{t('common.loading')}</p>}
      {account && !entry && !billsLoading && <Button variant="secondary" disabled={locked} onClick={() => void loadBills(account)}>{t('common.retry')}</Button>}
      {entry && account && <>
        <div className={styles.workspace}>
          <Card title={t('billing.saved_cycles')} extra={<Button type="button" variant="secondary" disabled={locked} onClick={beginNewCycle}>{t('billing.new_cycle')}</Button>}>
            {entry.bills.length === 0 ? <p className={styles.hint}>{t('billing.no_bills')}</p> : <ul className={styles.billList} aria-label={t('billing.saved_cycles')}>
              {entry.bills.map((bill) => <li key={bill.id}>
                <button type="button" className={styles.billChoice} aria-pressed={entry.editingId === bill.id} disabled={locked} onClick={() => selectBill(bill)}>
                  <span className={styles.billHeading}><strong>{t('billing.cycle_number', { cycle: bill.cycle })}</strong><span>CNY {billingYuan(bill.fee_cents)}</span></span>
                  <span>{dateFormat.format(new Date(bill.window_start))}</span>
                  <span>{t('billing.until', { end: dateFormat.format(new Date(bill.window_end)) })}</span>
                </button>
              </li>)}
            </ul>}
          </Card>
          <Card title={entry.editingId === null ? t('billing.new_cycle') : t('billing.edit_cycle')} subtitle={t('billing.timezone', { timezone: timeZone })}>
            <form className={styles.form} onSubmit={submitBill}>
              <div className={styles.formGrid}>
                <Input label={t('billing.cycle')} type="number" min="1" max={Number.MAX_SAFE_INTEGER} step="1" value={entry.draft.cycle} onChange={(event) => changeDraft('cycle', event.target.value)} readOnly={entry.editingId !== null} disabled={locked} required />
                <Input label={t('billing.fee')} type="text" inputMode="decimal" placeholder="841.22" value={entry.draft.fee} onChange={(event) => changeDraft('fee', event.target.value)} disabled={locked} required />
                <Input label={t('billing.start')} type="datetime-local" step="0.001" value={entry.draft.start} onChange={(event) => changeDraft('start', event.target.value)} disabled={locked} required />
                <Input label={t('billing.end')} type="datetime-local" step="0.001" value={entry.draft.end} onChange={(event) => changeDraft('end', event.target.value)} disabled={locked} required />
              </div>
              <div className="form-group">
                <label htmlFor={noteId}>{t('billing.fee_note')}</label>
                <textarea id={noteId} className={`input ${styles.noteInput}`} rows={3} value={entry.draft.feeNote} onChange={(event) => changeDraft('feeNote', event.target.value)} disabled={locked} />
              </div>
              <p className={styles.hint}>{t('billing.window_hint')}</p>
              <p className={styles.hint}>{t('billing.snapshot_hint')}</p>
              <div className={styles.actions}>
                <Button type="submit" disabled={locked} loading={busy === 'save'}>{existingCycle ? t('billing.recalculate_save') : t('billing.save')}</Button>
                {entry.editingId !== null && <Button type="button" variant="secondary" disabled={locked} onClick={beginNewCycle}>{t('billing.new_cycle')}</Button>}
              </div>
            </form>
          </Card>
        </div>
        {snapshot ? <Card title={t('billing.snapshot_title', { cycle: snapshot.cycle })} subtitle={t('billing.snapshot_subtitle')} extra={
          <div className={styles.actions}>
            <Button variant="secondary" onClick={() => {
              try { onDownload(new Blob([billingSnapshotCSV(snapshot)], { type: 'text/csv;charset=utf-8' }), `billing-${snapshot.id}-cycle-${snapshot.cycle}.csv`); }
              catch (error) { setBillError({ authIndex: selected, message: errorMessage(error) }); }
            }}>{t('billing.export_csv')}</Button>
            <Button variant="danger" disabled={locked} onClick={() => setConfirmation({ kind: 'delete', bill: snapshot })}>{t('common.delete')}</Button>
          </div>
        }>
          <dl className={styles.summary}>
            <div><dt>{t('billing.account')}</dt><dd>{snapshot.alias ? `${snapshot.alias} · ${snapshot.account}` : snapshot.account} · {snapshot.auth_index}</dd></div>
            <div><dt>{t('billing.plan')}</dt><dd>{snapshot.plan || '—'}</dd></div>
            <div><dt>{t('billing.fee')}</dt><dd className={styles.amount}>CNY {billingYuan(snapshot.fee_cents)}</dd></div>
            <div><dt>{t('billing.cost')}</dt><dd>{costFormat.format(snapshot.total_cost_usd)}</dd></div>
            <div><dt>{t('billing.requests')}</dt><dd>{numberFormat.format(snapshot.events)}</dd></div>
            <div><dt>{t('billing.calculated_at')}</dt><dd><time dateTime={snapshot.calculated_at}>{dateFormat.format(new Date(snapshot.calculated_at))}</time></dd></div>
          </dl>
          <dl className={styles.boundaries}>
            <div><dt>{t('billing.start')}</dt><dd><time dateTime={snapshot.window_start}>{dateFormat.format(new Date(snapshot.window_start))}</time><code>{snapshot.window_start}</code></dd></div>
            <div><dt>{t('billing.end')}</dt><dd><time dateTime={snapshot.window_end}>{dateFormat.format(new Date(snapshot.window_end))}</time><code>{snapshot.window_end}</code></dd></div>
            <div><dt>{t('billing.fee_note')}</dt><dd className={styles.note}>{snapshot.fee_note || '—'}</dd></div>
          </dl>
          <div className={styles.tableScroll} role="region" aria-label={t('billing.members')} tabIndex={0}>
            <table className={styles.table}>
              <caption>{t('billing.members')}</caption>
              <thead><tr>
                <th scope="col">{t('billing.member')}</th><th scope="col">{t('billing.requests')}</th><th scope="col">{t('billing.success')}</th><th scope="col">{t('billing.failure')}</th><th scope="col">{t('billing.tokens')}</th><th scope="col">{t('billing.cost')}</th><th scope="col">{t('billing.share')}</th><th scope="col">{t('billing.amount')}</th>
              </tr></thead>
              <tbody>{snapshot.members.map((member) => <tr key={member.api_key_id}>
                <th scope="row"><span className={styles.memberName}>{member.name}</span><small className={styles.hint}>#{member.api_key_id}</small></th>
                <td>{numberFormat.format(member.requests)}</td><td>{numberFormat.format(member.success)}</td><td>{numberFormat.format(member.failure)}</td>
                <td><details><summary>{numberFormat.format(member.total_tokens)}</summary><dl className={styles.tokenDetails}>
                  <div><dt>{t('billing.input_tokens')}</dt><dd>{numberFormat.format(member.input_tokens)}</dd></div>
                  <div><dt>{t('billing.output_tokens')}</dt><dd>{numberFormat.format(member.output_tokens)}</dd></div>
                  <div><dt>{t('billing.reasoning_tokens')}</dt><dd>{numberFormat.format(member.reasoning_tokens)}</dd></div>
                  <div><dt>{t('billing.cache_read_tokens')}</dt><dd>{numberFormat.format(member.cache_read_tokens)}</dd></div>
                  <div><dt>{t('billing.cache_read_ratio')}</dt><dd>{member.input_tokens > 0 ? percentFormat.format(member.cache_read_tokens / member.input_tokens) : '—'}</dd></div>
                </dl></details></td>
                <td>{costFormat.format(member.cost_usd)}</td><td>{percentFormat.format(member.share)}</td><td className={styles.amount}>CNY {billingYuan(member.amount_cents)}</td>
              </tr>)}</tbody>
              <tfoot><tr>
                <th scope="row" colSpan={5}>{t('billing.total')}</th>
                <td>{costFormat.format(snapshot.total_cost_usd)}</td>
                <td>{percentFormat.format(1)}</td>
                <td className={styles.amount}>CNY {billingYuan(snapshot.fee_cents)}</td>
              </tr></tfoot>
            </table>
          </div>
          <p className={styles.hint}>{t('billing.amount_hint')}</p>
        </Card> : <Card><p className={styles.hint}>{t('billing.no_snapshot')}</p></Card>}
      </>}
      <Modal
        open={confirmation !== null}
        title={confirmation?.kind === 'delete' ? t('billing.delete_title') : t('billing.recalculate_title')}
        onClose={() => { if (!busyRef.current) setConfirmation(null); }}
        closeDisabled={busy !== null}
        footer={<>
          <Button variant="secondary" disabled={busy !== null} onClick={() => setConfirmation(null)}>{t('common.cancel')}</Button>
          <Button variant={confirmation?.kind === 'delete' ? 'danger' : 'primary'} loading={busy !== null} onClick={() => {
            if (confirmation?.kind === 'delete') void removeBill(confirmation.bill);
            else if (confirmation?.kind === 'save') void commitBill(confirmation.request);
          }}>{confirmation?.kind === 'delete' ? t('common.delete') : t('billing.recalculate_save')}</Button>
        </>}
      >
        <p>{confirmation?.kind === 'delete' ? t('billing.delete_body', { cycle: confirmation.bill.cycle }) : t('billing.recalculate_body', { cycle: confirmation?.request.cycle })}</p>
      </Modal>
    </section>
  );
}
