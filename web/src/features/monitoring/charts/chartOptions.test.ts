import { describe, expect, it } from 'vitest';
import type { UsageChartMetricBucket, UsageChartSeries } from '@/services/api/usageService';
import { buildGlobalUsageChartOption, buildSeriesUsageChartOption } from './chartOptions';

type TestAxisOption = {
  data?: string[];
  name?: string;
  type?: string;
};

type TestSeriesOption = {
  data?: number[];
  name?: string;
  showSymbol?: boolean;
  type?: string;
};

type TestOption = {
  legend?: { type?: string };
  series?: TestSeriesOption[];
  tooltip?: { trigger?: string };
  xAxis?: TestAxisOption;
  yAxis?: TestAxisOption;
};

const asTestOption = (value: unknown): TestOption => value as TestOption;

const makeBucket = (overrides: Partial<UsageChartMetricBucket>): UsageChartMetricBucket => ({
  startMs: 0,
  endMs: 0,
  label: '',
  inputTokens: 0,
  outputTokens: 0,
  cachedTokens: 0,
  totalCost: 0,
  tpmInput: 0,
  tpmOutput: 0,
  tpmCached: 0,
  ...overrides,
});

describe('usage chart option builders', () => {
  it('builds a global token line chart from buckets', () => {
    const option = asTestOption(
      buildGlobalUsageChartOption({
        title: 'Global tokens',
        family: 'tokens',
        buckets: [
          makeBucket({ label: '10:00', inputTokens: 100, outputTokens: 40, cachedTokens: 20 }),
          makeBucket({ label: '10:01', inputTokens: 75, outputTokens: 30, cachedTokens: 10 }),
        ],
      })
    );

    expect(option.tooltip).toMatchObject({ trigger: 'axis' });
    expect(option.legend).toMatchObject({ type: 'scroll' });
    expect(option.xAxis).toMatchObject({ type: 'category', data: ['10:00', '10:01'] });
    expect(option.yAxis).toMatchObject({ type: 'value', name: 'Tokens' });
    expect(option.series).toEqual([
      expect.objectContaining({ name: 'Input tokens', type: 'line', showSymbol: false, data: [100, 75] }),
      expect.objectContaining({ name: 'Output tokens', type: 'line', showSymbol: false, data: [40, 30] }),
      expect.objectContaining({ name: 'Cached tokens', type: 'line', showSymbol: false, data: [20, 10] }),
    ]);
  });

  it('builds one cost line per dimension series', () => {
    const series: UsageChartSeries[] = [
      {
        key: 'provider-a:0',
        label: 'Provider A / auth 0',
        buckets: [
          makeBucket({ startMs: 1000, label: '10:00', totalCost: 0.12 }),
          makeBucket({ startMs: 2000, label: '10:01', totalCost: 0.24 }),
        ],
      },
      {
        key: 'provider-b:1',
        label: 'Provider B / auth 1',
        buckets: [
          makeBucket({ startMs: 1000, label: '10:00', totalCost: 0.03 }),
          makeBucket({ startMs: 2000, label: '10:01', totalCost: 0.09 }),
        ],
      },
    ];

    const option = asTestOption(
      buildSeriesUsageChartOption({
        title: 'Cost by provider',
        family: 'cost',
        series,
      })
    );

    expect(option.xAxis).toMatchObject({ data: ['10:00', '10:01'] });
    expect(option.yAxis).toMatchObject({ name: 'USD' });
    expect(option.series).toEqual([
      expect.objectContaining({ name: 'Provider A / auth 0', data: [0.12, 0.24] }),
      expect.objectContaining({ name: 'Provider B / auth 1', data: [0.03, 0.09] }),
    ]);
  });

  it('aligns sparse series buckets by timestamp', () => {
    const series: UsageChartSeries[] = [
      {
        key: 'full',
        label: 'Full series',
        buckets: [
          makeBucket({ startMs: 1000, label: '10:00', tpmInput: 12 }),
          makeBucket({ startMs: 2000, label: '10:01', tpmInput: 18 }),
        ],
      },
      {
        key: 'sparse',
        label: 'Sparse series',
        buckets: [makeBucket({ startMs: 2000, label: '10:01', tpmInput: 5 })],
      },
    ];

    const option = asTestOption(
      buildSeriesUsageChartOption({
        title: 'TPM by model',
        family: 'tpm',
        series,
      })
    );

    expect(option.xAxis).toMatchObject({ data: ['10:00', '10:01'] });
    expect(option.series?.map((item) => item.name)).toEqual([
      'Full series input TPM',
      'Full series output TPM',
      'Full series cached TPM',
      'Sparse series input TPM',
      'Sparse series output TPM',
      'Sparse series cached TPM',
    ]);
    expect(option.series?.[3]?.data).toEqual([0, 5]);
  });
});
