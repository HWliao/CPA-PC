import axios from 'axios';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { usageServiceApi, type UsageChartsResponse } from '@/services/api/usageService';

vi.mock('axios', () => ({
  default: {
    get: vi.fn(),
    isAxiosError: vi.fn(() => false),
  },
}));

const chartResponse: UsageChartsResponse = {
  range: '1h',
  granularity: 'hour',
  startMs: 1,
  endMs: 2,
  bucketMs: 3,
  filters: {},
  options: { providers: [], apiKeys: [], models: [] },
  global: { buckets: [] },
  byProvider: { series: [] },
  byApiKey: { series: [] },
  byModel: { series: [] },
  missingPriceModels: [],
  generatedAtMs: 4,
};

describe('usageServiceApi.getUsageCharts', () => {
  beforeEach(() => {
    vi.mocked(axios.get).mockReset();
    vi.mocked(axios.get).mockResolvedValue({ data: chartResponse });
  });

  it('calls the charts endpoint with supported params and management auth', async () => {
    const response = await usageServiceApi.getUsageCharts(
      'http://127.0.0.1:8317/v0/management',
      {
        range: '1h',
        granularity: 'hour',
        provider: 'auth:auth-a',
        apiKeyHash: 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
        model: 'gpt-test',
      },
      'manage-secret'
    );

    expect(response).toBe(chartResponse);
    expect(axios.get).toHaveBeenCalledWith(
      'http://127.0.0.1:8317/v0/management/usage/charts',
      expect.objectContaining({
        headers: { Authorization: 'Bearer manage-secret' },
        params: {
          range: '1h',
          granularity: 'hour',
          provider: 'auth:auth-a',
          apiKeyHash: 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
          model: 'gpt-test',
        },
      })
    );
  });

  it('omits empty optional params', async () => {
    await usageServiceApi.getUsageCharts(
      'http://127.0.0.1:8317',
      { range: '1h', provider: '', model: undefined },
      'manage-secret'
    );

    expect(axios.get).toHaveBeenCalledWith(
      'http://127.0.0.1:8317/v0/management/usage/charts',
      expect.objectContaining({
        params: { range: '1h' },
      })
    );
  });
});
