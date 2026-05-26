import type { UsageChartsGranularity, UsageChartsQueryParams, UsageChartsRange } from '@/services/api/usageService';

export type UsageChartsDimension = 'global' | 'account' | 'apiKey' | 'model';

export type UsageChartsFilterState = {
  range: UsageChartsRange;
  dimension: UsageChartsDimension;
  account: string;
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
  { value: '10m', labelKey: 'monitoring.charts_granularity_10m', defaultLabel: '10 minutes' },
  { value: 'hour', labelKey: 'monitoring.charts_granularity_hour', defaultLabel: 'Hourly' },
  { value: 'day', labelKey: 'monitoring.charts_granularity_day', defaultLabel: 'Daily' },
];

export const USAGE_CHART_DIMENSION_OPTIONS: Array<{
  value: UsageChartsDimension;
  labelKey: string;
  defaultLabel: string;
}> = [
  { value: 'global', labelKey: 'monitoring.charts_dimension_global', defaultLabel: 'Global total' },
  { value: 'account', labelKey: 'monitoring.charts_dimension_account', defaultLabel: 'Account' },
  { value: 'apiKey', labelKey: 'monitoring.charts_dimension_api_key', defaultLabel: 'Caller key' },
  { value: 'model', labelKey: 'monitoring.charts_dimension_model', defaultLabel: 'Model' },
];

export const createDefaultUsageChartsFilterState = (): UsageChartsFilterState => ({
  range: '1h',
  dimension: 'global',
  account: '',
  apiKeyHash: '',
  model: '',
});

export const resolveDefaultUsageChartsGranularity = (
  range: UsageChartsRange
): UsageChartsGranularity => {
  if (range === '1h') return '10m';
  if (range === '7d') return 'day';
  return 'hour';
};

export const shouldDisableUsageChartsFilter = (
  filter: Exclude<UsageChartsDimension, 'global'>,
  dimension: UsageChartsDimension
): boolean => filter === dimension;

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
    granularity: resolveDefaultUsageChartsGranularity(state.range),
  };

  if (!shouldDisableUsageChartsFilter('account', state.dimension)) {
    appendNonEmptyParam(params, 'account', state.account);
  }
  if (!shouldDisableUsageChartsFilter('apiKey', state.dimension)) {
    appendNonEmptyParam(params, 'apiKeyHash', state.apiKeyHash);
  }
  if (!shouldDisableUsageChartsFilter('model', state.dimension)) {
    appendNonEmptyParam(params, 'model', state.model);
  }
  return params;
}
