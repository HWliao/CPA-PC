import { renderToStaticMarkup } from 'react-dom/server';
import { act, type ReactNode } from 'react';
import { create, type ReactTestRenderer } from 'react-test-renderer';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { Select } from '@/components/ui/Select';
import { EChartPanel } from '@/features/monitoring/charts/EChartPanel';
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
  granularity: '10m',
  startMs: 0,
  endMs: 600000,
  bucketMs: 600000,
  filters: {},
  options: {
    providers: [{ value: 'auth:2', label: 'Team Codex', provider: 'openai', authIndex: '2' }],
    apiKeys: [{ value: 'hash-1', apiKeyHash: 'hash-1', label: 'Build key' }],
    models: [{ value: 'gpt-5', model: 'gpt-5', label: 'GPT-5' }],
  },
  global: {
    buckets: [
      {
        startMs: 0,
        endMs: 600000,
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
  byProvider: { series: [] },
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

  it('renders the empty state when buckets have no usage values', () => {
    vi.mocked(useUsageCharts).mockReturnValue(
      createHookState({
        charts: createChartsResponse({
          options: { providers: [], apiKeys: [], models: [] },
          global: {
            buckets: [
              {
                startMs: 0,
                endMs: 3600000,
                label: '10:00',
                inputTokens: 0,
                outputTokens: 0,
                cachedTokens: 0,
                totalCost: 0,
                tpmInput: 0,
                tpmOutput: 0,
                tpmCached: 0,
              },
            ],
          },
        }),
      })
    );

    const html = renderToStaticMarkup(<MonitoringChartsPage />);

    expect(html).toContain('No chart data yet');
    expect(html).not.toContain('Token usage');
  });

  it('renders global chart panels and missing price warnings', () => {
    vi.mocked(useUsageCharts).mockReturnValue(
      createHookState({
        charts: createChartsResponse({ missingPriceModels: ['unknown-model'] }),
      })
    );

    const html = renderToStaticMarkup(<MonitoringChartsPage />);

    expect(html).toContain('Token usage');
    expect(html).toContain('Cumulative token usage');
    expect(html).toContain('Cost');
    expect(html).toContain('Cumulative cost');
    expect(html).toContain('TPM');
    expect(html).toContain('Missing model prices');
    expect(html).toContain('unknown-model');
  });

  it('passes derived granularity and linked filters to the chart loader', () => {
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
    const hasSelectByLabel = (ariaLabel: string) =>
      renderer!.root.findAllByType(Select).some((node) => node.props.ariaLabel === ariaLabel);

    expect(latestParams()).toEqual({ range: '1h', granularity: '10m' });

    act(() => {
      selectByLabel('Time range').props.onChange('7d');
    });
    expect(latestParams()).toEqual({ range: '7d', granularity: 'day' });

    act(() => {
      selectByLabel('Provider').props.onChange('auth:2');
      selectByLabel('API key').props.onChange('hash-1');
      selectByLabel('Model').props.onChange('gpt-5');
    });
    expect(latestParams()).toEqual({
      range: '7d',
      granularity: 'day',
      provider: 'auth:2',
      apiKeyHash: 'hash-1',
      model: 'gpt-5',
    });

    act(() => {
      selectByLabel('Chart dimension').props.onChange('provider');
    });
    expect(latestParams()).toEqual({
      range: '7d',
      granularity: 'day',
      apiKeyHash: 'hash-1',
      model: 'gpt-5',
    });
    expect(hasSelectByLabel('Provider')).toBe(false);

    act(() => {
      selectByLabel('Chart dimension').props.onChange('apiKey');
    });
    expect(hasSelectByLabel('API key')).toBe(false);
    expect(hasSelectByLabel('Provider')).toBe(true);

    act(() => {
      selectByLabel('Chart dimension').props.onChange('model');
    });
    expect(hasSelectByLabel('Model')).toBe(false);
    expect(hasSelectByLabel('API key')).toBe(true);

    renderer!.unmount();
  });

  it('renders token and cost chart tabs with three visible chart panels', () => {
    vi.mocked(useUsageCharts).mockReturnValue(createHookState({ charts: createChartsResponse() }));

    let renderer: ReactTestRenderer;
    act(() => {
      renderer = create(<MonitoringChartsPage />);
    });

    expect(renderer!.root.findAllByType(EChartPanel)).toHaveLength(3);
    expect(renderer!.root.findAll((node) => node.props.role === 'tab')).toHaveLength(4);

    const chartOptions = () => renderer!.root.findAllByType(EChartPanel).map((node) => node.props.option);
    expect(chartOptions()[0].series[0].data).toEqual([100]);
    expect(chartOptions()[1].series[0].data).toEqual([0.04]);

    const cumulativeTokenTab = renderer!.root
      .findAll((node) => node.props.role === 'tab')
      .find((node) => node.props.children === 'Cumulative token usage');
    const cumulativeCostTab = renderer!.root
      .findAll((node) => node.props.role === 'tab')
      .find((node) => node.props.children === 'Cumulative cost');
    if (!cumulativeTokenTab || !cumulativeCostTab) throw new Error('Cumulative tabs not found');

    act(() => {
      cumulativeTokenTab.props.onClick();
      cumulativeCostTab.props.onClick();
    });

    expect(chartOptions()[0].title.text).toBe('Cumulative token usage');
    expect(chartOptions()[1].title.text).toBe('Cumulative cost');
    renderer!.unmount();
  });

  it('switches the three visible charts to the selected dimension series', () => {
    vi.mocked(useUsageCharts).mockImplementation(() =>
      createHookState({
        charts: createChartsResponse({
          byProvider: { series: [createSeries('auth:2', 'Team Codex')] },
          byApiKey: { series: [createSeries('api-key', 'Build key')] },
          byModel: { series: [createSeries('model', 'GPT-5')] },
        }),
      })
    );

    let renderer: ReactTestRenderer;
    act(() => {
      renderer = create(<MonitoringChartsPage />);
    });
    const dimensionSelect = renderer!.root
      .findAllByType(Select)
      .find((node) => node.props.ariaLabel === 'Chart dimension');
    if (!dimensionSelect) throw new Error('Dimension select not found');

    act(() => {
      dimensionSelect.props.onChange('provider');
    });

    const chartOptions = renderer!.root.findAllByType(EChartPanel).map((node) => node.props.option);
    expect(chartOptions[0].series[0].name).toBe('Team Codex input tokens');
    expect(chartOptions[1].series[0].name).toBe('Team Codex');
    expect(chartOptions[2].series[0].name).toBe('Team Codex input TPM');
    renderer!.unmount();
  });
});
