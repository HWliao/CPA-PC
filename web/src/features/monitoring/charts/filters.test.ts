import { describe, expect, it } from 'vitest';
import {
  USAGE_CHART_DIMENSION_OPTIONS,
  USAGE_CHART_RANGE_OPTIONS,
  buildUsageChartsQueryParams,
  createDefaultUsageChartsFilterState,
  resolveDefaultUsageChartsGranularity,
  shouldDisableUsageChartsFilter,
} from './filters';

describe('usage chart filters', () => {
  it('supports only fixed monitoring ranges', () => {
    expect(USAGE_CHART_RANGE_OPTIONS.map((option) => option.value)).toEqual(['1h', '5h', '24h', '7d']);
  });

  it('supports account, caller key, and model dimensions', () => {
    expect(USAGE_CHART_DIMENSION_OPTIONS.map((option) => option.value)).toEqual([
      'global',
      'account',
      'apiKey',
      'model',
    ]);
  });

  it('derives granularity from the selected range', () => {
    expect(resolveDefaultUsageChartsGranularity('1h')).toBe('10m');
    expect(resolveDefaultUsageChartsGranularity('5h')).toBe('hour');
    expect(resolveDefaultUsageChartsGranularity('24h')).toBe('hour');
    expect(resolveDefaultUsageChartsGranularity('7d')).toBe('day');
  });

  it('builds query params without empty filter values', () => {
    expect(buildUsageChartsQueryParams(createDefaultUsageChartsFilterState())).toEqual({
      range: '1h',
      granularity: '10m',
    });

    expect(
      buildUsageChartsQueryParams({
        range: '7d',
        dimension: 'global',
        account: ' Team Codex ',
        provider: '',
        apiKeyHash: 'hash-1',
        model: 'gpt-5',
      })
    ).toEqual({
      range: '7d',
      granularity: 'day',
      account: 'Team Codex',
      apiKeyHash: 'hash-1',
      model: 'gpt-5',
    });
  });

  it('omits filters that are used as the active chart dimension', () => {
    expect(
      buildUsageChartsQueryParams({
        range: '24h',
        dimension: 'account',
        account: 'Team Codex',
        provider: '',
        apiKeyHash: 'hash-1',
        model: 'gpt-5',
      })
    ).toEqual({
      range: '24h',
      granularity: 'hour',
      apiKeyHash: 'hash-1',
      model: 'gpt-5',
    });

    expect(shouldDisableUsageChartsFilter('account', 'account')).toBe(true);
    expect(shouldDisableUsageChartsFilter('account', 'apiKey')).toBe(false);
  });
});
