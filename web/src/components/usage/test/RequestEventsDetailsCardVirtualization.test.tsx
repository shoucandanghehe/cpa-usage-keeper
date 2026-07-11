// @vitest-environment happy-dom

import React, { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { UsageEvent } from '@/lib/types';
import { RequestEventsDetailsCard } from '../RequestEventsDetailsCard';

const buildEvent = (index: number): UsageEvent => ({
  id: String(index + 1),
  timestamp: '2026-07-11T10:00:00.000Z',
  api_key: 'Production Key',
  model: `model-${index}`,
  endpoint: 'POST /v1/messages',
  source: 'Provider A',
  source_raw: 'source-a',
  source_type: 'openai',
  auth_index: '1',
  request_id: `request-${index}`,
  failed: false,
  latency_ms: 120,
  ttft_ms: 45,
  speed_tps: 30,
  tokens: {
    input_tokens: 100,
    output_tokens: 60,
    reasoning_tokens: 20,
    cache_read_tokens: 20,
    cache_creation_tokens: 0,
    total_tokens: 200,
  },
  cost_usd: 0.1234,
  cost_available: true,
  pricing_style: 'claude',
});

const baseProps: Omit<React.ComponentProps<typeof RequestEventsDetailsCard>, 'events' | 'totalCount' | 'totalPages'> = {
  loading: false,
  page: 1,
  pageSize: 1000,
  pageSizeOptions: [20, 100, 1000],
  modelOptions: [],
  sourceOptions: [],
  modelFilter: '__all__',
  sourceFilter: '__all__',
  resultFilter: '__all__',
  initialVisibleColumnIds: ['timestamp', 'model', 'total_tokens'],
  onPageChange: () => undefined,
  onPageSizeChange: () => undefined,
  onModelFilterChange: () => undefined,
  onSourceFilterChange: () => undefined,
  onResultFilterChange: () => undefined,
};

const rect = (width: number, height: number): DOMRect => ({
  x: 0,
  y: 0,
  top: 0,
  right: width,
  bottom: height,
  left: 0,
  width,
  height,
  toJSON: () => ({}),
});

class TestResizeObserver implements ResizeObserver {
  private readonly callback: ResizeObserverCallback;

  constructor(callback: ResizeObserverCallback) {
    this.callback = callback;
  }

  observe(target: Element) {
    const contentRect = target.getBoundingClientRect();
    this.callback([{
      target,
      contentRect,
      borderBoxSize: [{ inlineSize: contentRect.width, blockSize: contentRect.height }],
      contentBoxSize: [{ inlineSize: contentRect.width, blockSize: contentRect.height }],
      devicePixelContentBoxSize: [],
    } as unknown as ResizeObserverEntry], this);
  }

  disconnect() {}

  unobserve() {}
}

describe('RequestEventsDetailsCard event table virtualization', () => {
  let container: HTMLDivElement;
  let root: Root;

  beforeEach(() => {
    globalThis.IS_REACT_ACT_ENVIRONMENT = true;
    vi.stubGlobal('ResizeObserver', TestResizeObserver);
    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function getBoundingClientRect() {
      const className = typeof this.className === 'string' ? this.className : '';
      if (className.includes('requestEventsTableWrapper')) {
        return rect(1200, 600);
      }
      if (this instanceof HTMLTableRowElement) {
        const spacerHeight = Number.parseFloat(this.style.height);
        return rect(1200, Number.isFinite(spacerHeight) ? spacerHeight : 44);
      }
      return rect(1200, 600);
    });
    container = document.createElement('div');
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(async () => {
    await act(async () => root.unmount());
    container.remove();
    document.body.innerHTML = '';
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it('keeps a 1000-event page bounded in the DOM and advances the window on scroll', async () => {
    const events = Array.from({ length: 1000 }, (_, index) => buildEvent(index));
    await act(async () => {
      root.render(
        <RequestEventsDetailsCard
          {...baseProps}
          events={events}
          totalCount={events.length}
          totalPages={1}
        />,
      );
      await Promise.resolve();
    });

    const scroller = document.querySelector<HTMLElement>('[class*="requestEventsTableWrapper"]');
    const table = scroller?.querySelector('table');
    expect(scroller?.dataset.virtualized).toBe('true');
    expect(table?.getAttribute('aria-rowcount')).toBe('1001');

    const initialRows = Array.from(scroller?.querySelectorAll<HTMLTableRowElement>('tbody tr[data-index]') ?? []);
    const initialIndexes = initialRows.map((row) => Number(row.dataset.index));
    expect(initialRows.length).toBeGreaterThan(0);
    expect(initialRows.length).toBeLessThan(100);

    if (scroller) {
      scroller.scrollTop = 22_000;
      await act(async () => {
        scroller.dispatchEvent(new Event('scroll'));
        await new Promise((resolve) => window.setTimeout(resolve, 0));
      });
    }

    const scrolledRows = Array.from(scroller?.querySelectorAll<HTMLTableRowElement>('tbody tr[data-index]') ?? []);
    const scrolledIndexes = scrolledRows.map((row) => Number(row.dataset.index));
    expect(scrolledRows.length).toBeGreaterThan(0);
    expect(scrolledRows.length).toBeLessThan(100);
    expect(Math.min(...scrolledIndexes)).toBeGreaterThan(Math.min(...initialIndexes));

    if (scroller) {
      scroller.scrollTop = 43_500;
      await act(async () => {
        scroller.dispatchEvent(new Event('scroll'));
        await new Promise((resolve) => window.setTimeout(resolve, 0));
      });
    }

    const finalRows = Array.from(scroller?.querySelectorAll<HTMLTableRowElement>('tbody tr[data-index]') ?? []);
    const finalIndexes = finalRows.map((row) => Number(row.dataset.index));
    expect(finalRows.length).toBeLessThan(100);
    expect(Math.max(...finalIndexes)).toBe(999);
  });

  it('keeps small pages fully rendered without virtual spacer rows', async () => {
    const events = Array.from({ length: 3 }, (_, index) => buildEvent(index));
    await act(async () => {
      root.render(
        <RequestEventsDetailsCard
          {...baseProps}
          events={events}
          pageSize={20}
          totalCount={events.length}
          totalPages={1}
        />,
      );
      await Promise.resolve();
    });

    const scroller = document.querySelector<HTMLElement>('[class*="requestEventsTableWrapper"]');
    const rows = scroller?.querySelectorAll('tbody tr') ?? [];
    expect(scroller?.dataset.virtualized).toBe('false');
    expect(rows).toHaveLength(3);
    expect(scroller?.querySelector('[class*="requestEventsVirtualSpacerRow"]')).toBeNull();
    expect(scroller?.textContent).toContain('model-2');
  });
});
