import { renderToStaticMarkup } from 'react-dom/server';
import { act, type ReactNode } from 'react';
import { create, type ReactTestRenderer } from 'react-test-renderer';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { Select } from '@/components/ui/Select';
import { mainRoutes } from '@/router/MainRoutes';
import { useUsageCharts, type UseUsageChartsReturn } from '@/features/monitoring/charts/useUsageCharts';
import type { UsageChartSeries, UsageChartsResponse } from '@/services/api/usageService';
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
    providers: ['openai'],
    authFiles: [{ authIndex: '2', label: 'Team Codex', provider: 'openai' }],
    apiKeys: [{ apiKeyHash: 'hash-1', label: 'Build key' }],
    models: ['gpt-5'],
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

const createSeries = (key: string, label: string): UsageChartSeries => ({
  key,
  label,
  buckets: [
    {
      startMs: 0,
      endMs: 3600000,
      label: '10:00',
      inputTokens: 12,
      outputTokens: 4,
      cachedTokens: 2,
      totalCost: 0.01,
      tpmInput: 1,
      tpmOutput: 0.5,
      tpmCached: 0.25,
    },
  ],
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

  it('passes range, granularity, and filters to the chart loader', () => {
    vi.mocked(useUsageCharts).mockImplementation(() =>
      createHookState({ charts: createChartsResponse() })
    );

    let renderer: ReactTestRenderer;
    act(() => {
      renderer = create(<MonitoringChartsPage />);
    });

    const latestParams = () => {
      const calls = vi.mocked(useUsageCharts).mock.calls;
      return calls[calls.length - 1]?.[0];
    };
    const selectByLabel = (ariaLabel: string) => {
      const match = renderer!.root
        .findAllByType(Select)
        .find((node) => node.props.ariaLabel === ariaLabel);
      if (!match) throw new Error(`Select not found: ${ariaLabel}`);
      return match;
    };

    expect(latestParams()).toEqual({ range: '1h', granularity: 'hour' });

    act(() => {
      selectByLabel('Time range').props.onChange('7d');
    });
    expect(latestParams()).toEqual({ range: '7d', granularity: 'day' });

    act(() => {
      selectByLabel('Granularity').props.onChange('hour');
      selectByLabel('Provider').props.onChange('openai');
      selectByLabel('Auth file').props.onChange('2');
      selectByLabel('API key').props.onChange('hash-1');
      selectByLabel('Model').props.onChange('gpt-5');
    });
    expect(latestParams()).toEqual({
      range: '7d',
      granularity: 'hour',
      provider: 'openai',
      authIndex: '2',
      apiKeyHash: 'hash-1',
      model: 'gpt-5',
    });

    renderer!.unmount();
  });

  it('renders dimension chart sections for non-empty series', () => {
    vi.mocked(useUsageCharts).mockReturnValue(
      createHookState({
        charts: createChartsResponse({
          byProviderAuthFile: { series: [createSeries('provider', 'OpenAI / Team Codex')] },
          byApiKey: { series: [createSeries('api-key', 'Build key')] },
          byModel: { series: [createSeries('model', 'gpt-5')] },
        }),
      })
    );

    const html = renderToStaticMarkup(<MonitoringChartsPage />);

    expect(html).toContain('Provider/auth files');
    expect(html).toContain('Provider/auth tokens');
    expect(html).toContain('Provider/auth cost');
    expect(html).toContain('Provider/auth TPM');
    expect(html).toContain('API-key tokens');
    expect(html).toContain('API-key cost');
    expect(html).toContain('API-key TPM');
    expect(html).toContain('Model tokens');
    expect(html).toContain('Model cost');
    expect(html).toContain('Model TPM');
  });

  it('renders empty states for dimensions without series', () => {
    vi.mocked(useUsageCharts).mockReturnValue(createHookState({ charts: createChartsResponse() }));

    const html = renderToStaticMarkup(<MonitoringChartsPage />);

    expect(html).toContain('No provider/auth-file series');
    expect(html).toContain('No API-key series');
    expect(html).toContain('No model series');
  });
});
