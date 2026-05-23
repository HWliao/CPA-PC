import type { EChartsCoreOption } from 'echarts/core';
import type { UsageChartMetricBucket, UsageChartSeries } from '@/services/api/usageService';

export type UsageChartMetricFamily = 'tokens' | 'cumulativeTokens' | 'cost' | 'cumulativeCost' | 'tpm';

type UsageChartMetricKey = Extract<
  keyof UsageChartMetricBucket,
  | 'inputTokens'
  | 'outputTokens'
  | 'cachedTokens'
  | 'totalCost'
  | 'tpmInput'
  | 'tpmOutput'
  | 'tpmCached'
>;

type MetricDefinition = {
  key: UsageChartMetricKey;
  label: string;
  suffixLabel?: string;
};

type UsageLineSeries = {
  data: number[];
  name: string;
};

export interface BuildGlobalUsageChartOptionInput {
  title: string;
  family: UsageChartMetricFamily;
  buckets: UsageChartMetricBucket[];
}

export interface BuildSeriesUsageChartOptionInput {
  title: string;
  family: UsageChartMetricFamily;
  series: UsageChartSeries[];
}

const METRICS_BY_FAMILY: Record<UsageChartMetricFamily, MetricDefinition[]> = {
  tokens: [
    { key: 'inputTokens', label: 'Input tokens', suffixLabel: 'input tokens' },
    { key: 'outputTokens', label: 'Output tokens', suffixLabel: 'output tokens' },
    { key: 'cachedTokens', label: 'Cached tokens', suffixLabel: 'cached tokens' },
  ],
  cumulativeTokens: [
    { key: 'inputTokens', label: 'Input tokens', suffixLabel: 'input tokens' },
    { key: 'outputTokens', label: 'Output tokens', suffixLabel: 'output tokens' },
    { key: 'cachedTokens', label: 'Cached tokens', suffixLabel: 'cached tokens' },
  ],
  cost: [{ key: 'totalCost', label: 'Cost' }],
  cumulativeCost: [{ key: 'totalCost', label: 'Cost' }],
  tpm: [
    { key: 'tpmInput', label: 'Input TPM', suffixLabel: 'input TPM' },
    { key: 'tpmOutput', label: 'Output TPM', suffixLabel: 'output TPM' },
    { key: 'tpmCached', label: 'Cached TPM', suffixLabel: 'cached TPM' },
  ],
};

const Y_AXIS_NAME_BY_FAMILY: Record<UsageChartMetricFamily, string> = {
  tokens: 'Tokens',
  cumulativeTokens: 'Tokens',
  cost: 'USD',
  cumulativeCost: 'USD',
  tpm: 'TPM',
};

const readMetricValue = (bucket: UsageChartMetricBucket | undefined, key: UsageChartMetricKey): number => {
  if (!bucket) return 0;
  const value = bucket[key];
  return Number.isFinite(value) ? value : 0;
};

const buildMetricData = (
  buckets: Array<UsageChartMetricBucket | undefined>,
  key: UsageChartMetricKey,
  cumulative: boolean
): number[] => {
  let runningTotal = 0;
  return buckets.map((bucket) => {
    const value = readMetricValue(bucket, key);
    if (!cumulative) return value;
    runningTotal += value;
    return runningTotal;
  });
};

export const formatChartMetricValue = (value: number): string => {
  const num = Number(value);
  if (!Number.isFinite(num)) return '0';

  const abs = Math.abs(num);
  if (abs >= 1_000_000_000) return `${(num / 1_000_000_000).toFixed(2)}B`;
  if (abs >= 1_000_000) return `${(num / 1_000_000).toFixed(2)}M`;
  if (abs >= 1_000) return `${(num / 1_000).toFixed(2)}K`;
  if (abs >= 100) return num.toFixed(0);
  if (abs === 0) return '0';
  return num.toFixed(2);
};

const buildBaseLineChartOption = ({
  title,
  labels,
  yAxisName,
  series,
}: {
  title: string;
  labels: string[];
  yAxisName: string;
  series: UsageLineSeries[];
}): EChartsCoreOption => ({
  title: {
    text: title,
    left: 0,
    top: 0,
    textStyle: {
      fontSize: 14,
      fontWeight: 700,
    },
  },
  tooltip: {
    trigger: 'axis',
    valueFormatter: (value: number) => formatChartMetricValue(value),
  },
  legend: {
    type: 'scroll',
    bottom: 0,
  },
  grid: {
    top: 48,
    right: 16,
    bottom: 72,
    left: 12,
    containLabel: true,
  },
  xAxis: {
    type: 'category',
    boundaryGap: false,
    data: labels,
  },
  yAxis: {
    type: 'value',
    name: yAxisName,
    axisLabel: {
      formatter: (value: number) => formatChartMetricValue(value),
    },
  },
  series: series.map((item) => ({
    name: item.name,
    type: 'line',
    showSymbol: false,
    smooth: true,
    data: item.data,
  })),
});

export function buildGlobalUsageChartOption({
  title,
  family,
  buckets,
}: BuildGlobalUsageChartOptionInput): EChartsCoreOption {
  const metrics = METRICS_BY_FAMILY[family];
  const cumulative = isCumulativeFamily(family);
  return buildBaseLineChartOption({
    title,
    labels: buckets.map((bucket) => bucket.label),
    yAxisName: Y_AXIS_NAME_BY_FAMILY[family],
    series: metrics.map((metric) => ({
      name: metric.label,
      data: buildMetricData(buckets, metric.key, cumulative),
    })),
  });
}

export function buildSeriesUsageChartOption({
  title,
  family,
  series,
}: BuildSeriesUsageChartOptionInput): EChartsCoreOption {
  const metrics = METRICS_BY_FAMILY[family];
  const cumulative = isCumulativeFamily(family);
  const labelsSource = series.find((item) => item.buckets.length > 0)?.buckets ?? [];
  const startMsValues = labelsSource.map((bucket) => bucket.startMs);

  return buildBaseLineChartOption({
    title,
    labels: labelsSource.map((bucket) => bucket.label),
    yAxisName: Y_AXIS_NAME_BY_FAMILY[family],
    series: series.flatMap((item) => {
      const bucketsByStartMs = new Map(item.buckets.map((bucket) => [bucket.startMs, bucket]));
      return metrics.map((metric) => ({
        name: metrics.length === 1 ? item.label : `${item.label} ${metric.suffixLabel ?? metric.label}`,
        data: buildMetricData(
          startMsValues.map((startMs) => bucketsByStartMs.get(startMs)),
          metric.key,
          cumulative
        ),
      }));
    }),
  });
}

function isCumulativeFamily(family: UsageChartMetricFamily): boolean {
  return family === 'cumulativeTokens' || family === 'cumulativeCost';
}
