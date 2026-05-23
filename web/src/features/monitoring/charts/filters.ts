import type { UsageChartsGranularity, UsageChartsQueryParams, UsageChartsRange } from '@/services/api/usageService';

export type UsageChartsFilterState = {
  range: UsageChartsRange;
  granularity: UsageChartsGranularity;
  provider: string;
  authIndex: string;
  apiKeyHash: string;
  model: string;
};

export const USAGE_CHART_RANGE_OPTIONS: Array<{
  value: UsageChartsRange;
  labelKey: string;
  defaultLabel: string;
}> = [
  { value: '1h', labelKey: 'monitoring.charts_range_1h', defaultLabel: 'Last 1 hour' },
  { value: '5h', labelKey: 'monitoring.charts_range_5h', defaultLabel: 'Last 5 hours' },
  { value: '24h', labelKey: 'monitoring.charts_range_24h', defaultLabel: 'Last 24 hours' },
  { value: '7d', labelKey: 'monitoring.charts_range_7d', defaultLabel: 'Last 7 days' },
];

export const USAGE_CHART_GRANULARITY_OPTIONS: Array<{
  value: UsageChartsGranularity;
  labelKey: string;
  defaultLabel: string;
}> = [
  { value: 'hour', labelKey: 'monitoring.charts_granularity_hour', defaultLabel: 'Hourly' },
  { value: 'day', labelKey: 'monitoring.charts_granularity_day', defaultLabel: 'Daily' },
];

export const createDefaultUsageChartsFilterState = (): UsageChartsFilterState => ({
  range: '1h',
  granularity: 'hour',
  provider: '',
  authIndex: '',
  apiKeyHash: '',
  model: '',
});

export const resolveDefaultUsageChartsGranularity = (
  range: UsageChartsRange
): UsageChartsGranularity => (range === '7d' ? 'day' : 'hour');

const appendNonEmptyParam = <K extends keyof UsageChartsQueryParams>(
  params: UsageChartsQueryParams,
  key: K,
  value: string
) => {
  const trimmed = value.trim();
  if (trimmed) {
    params[key] = trimmed as UsageChartsQueryParams[K];
  }
};

export function buildUsageChartsQueryParams(state: UsageChartsFilterState): UsageChartsQueryParams {
  const params: UsageChartsQueryParams = {
    range: state.range,
    granularity: state.granularity,
  };

  appendNonEmptyParam(params, 'provider', state.provider);
  appendNonEmptyParam(params, 'authIndex', state.authIndex);
  appendNonEmptyParam(params, 'apiKeyHash', state.apiKeyHash);
  appendNonEmptyParam(params, 'model', state.model);
  return params;
}
