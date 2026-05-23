import { renderToStaticMarkup } from 'react-dom/server';
import type { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { mainRoutes } from '@/router/MainRoutes';
import { useUsageCharts, type UseUsageChartsReturn } from '@/features/monitoring/charts/useUsageCharts';
import type { UsageChartsResponse } from '@/services/api/usageService';
import { MonitoringChartsPage } from './MonitoringChartsPage';

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual<typeof import('react-router-dom')>('react-router-dom');
  return {
    ...actual,
    Link: ({ to, children, className }: { to: string; children: ReactNode; className?: string }) => (
      <a href={to} className={className}>{children}</a>
    ),
  };
});

vi.mock('react-i18next', () => ({
  initReactI18next: {
    type: '3rdParty',
    init: vi.fn(),
  },
  useTranslation: () => ({
    t: (key: string, options?: Record<string, unknown>) => String(options?.defaultValue ?? key),
    i18n: { language: 'en' },
  }),
}));

vi.mock('@/features/monitoring/charts/useUsageCharts', () => ({
  useUsageCharts: vi.fn(),
}));

const createHookState = (overrides: Partial<UseUsageChartsReturn> = {}): UseUsageChartsReturn => ({
  charts: null,
  loading: false,
  error: '',
  lastRefreshedAt: null,
  usageServiceAvailable: true,
  loadCharts: vi.fn(async () => {}),
  ...overrides,
});

const createChartsResponse = (overrides: Partial<UsageChartsResponse> = {}): UsageChartsResponse => ({
  range: '1h',
  granularity: 'hour',
  startMs: 0,
  endMs: 3600000,
  bucketMs: 3600000,
  filters: {},
  options: {
    providers: [],
    authFiles: [],
    apiKeys: [],
    models: [],
  },
  global: {
    buckets: [
      {
        startMs: 0,
        endMs: 3600000,
        label: '10:00',
        inputTokens: 100,
        outputTokens: 25,
        cachedTokens: 10,
        totalCost: 0.04,
        tpmInput: 8,
        tpmOutput: 2,
        tpmCached: 1,
      },
    ],
  },
  byProviderAuthFile: { series: [] },
  byApiKey: { series: [] },
  byModel: { series: [] },
  missingPriceModels: [],
  generatedAtMs: 0,
  ...overrides,
});

describe('MonitoringChartsPage', () => {
  beforeEach(() => {
    vi.mocked(useUsageCharts).mockReset();
  });

  it('is registered as a monitoring route', () => {
    expect(mainRoutes.some((route) => route.path === '/monitoring/charts')).toBe(true);
  });

  it('renders the loading state', () => {
    vi.mocked(useUsageCharts).mockReturnValue(createHookState({ loading: true }));

    const html = renderToStaticMarkup(<MonitoringChartsPage />);

    expect(html).toContain('Monitoring Charts');
    expect(html).toContain('Loading chart data');
  });

  it('renders error and empty states', () => {
    vi.mocked(useUsageCharts).mockReturnValue(createHookState({ error: 'boom' }));
    expect(renderToStaticMarkup(<MonitoringChartsPage />)).toContain('boom');

    vi.mocked(useUsageCharts).mockReturnValue(createHookState({ charts: null }));
    expect(renderToStaticMarkup(<MonitoringChartsPage />)).toContain('No chart data yet');
  });

  it('renders global chart panels and missing price warnings', () => {
    vi.mocked(useUsageCharts).mockReturnValue(
      createHookState({
        charts: createChartsResponse({ missingPriceModels: ['unknown-model'] }),
      })
    );

    const html = renderToStaticMarkup(<MonitoringChartsPage />);

    expect(html).toContain('Global tokens');
    expect(html).toContain('Global cost');
    expect(html).toContain('Global TPM');
    expect(html).toContain('Missing model prices');
    expect(html).toContain('unknown-model');
  });
});
