import { describe, expect, it } from 'vitest';
import {
  USAGE_CHART_RANGE_OPTIONS,
  buildUsageChartsQueryParams,
  createDefaultUsageChartsFilterState,
  resolveDefaultUsageChartsGranularity,
} from './filters';

describe('usage chart filters', () => {
  it('supports only fixed monitoring ranges', () => {
    expect(USAGE_CHART_RANGE_OPTIONS.map((option) => option.value)).toEqual(['1h', '5h', '24h', '7d']);
  });

  it('uses hour granularity except for the seven day default', () => {
    expect(resolveDefaultUsageChartsGranularity('1h')).toBe('hour');
    expect(resolveDefaultUsageChartsGranularity('5h')).toBe('hour');
    expect(resolveDefaultUsageChartsGranularity('24h')).toBe('hour');
    expect(resolveDefaultUsageChartsGranularity('7d')).toBe('day');
  });

  it('builds query params without empty filter values', () => {
    expect(buildUsageChartsQueryParams(createDefaultUsageChartsFilterState())).toEqual({
      range: '1h',
      granularity: 'hour',
    });

    expect(
      buildUsageChartsQueryParams({
        range: '7d',
        granularity: 'day',
        provider: ' openai ',
        authIndex: '2',
        apiKeyHash: 'hash-1',
        model: 'gpt-5',
      })
    ).toEqual({
      range: '7d',
      granularity: 'day',
      provider: 'openai',
      authIndex: '2',
      apiKeyHash: 'hash-1',
      model: 'gpt-5',
    });
  });
});
