import axios from 'axios';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  usageServiceApi,
  type ModelPriceSyncRequest,
  type ModelPriceSyncResponse,
} from './usageService';

vi.mock('axios', () => ({
  default: {
    post: vi.fn(),
    isAxiosError: vi.fn(() => false),
  },
}));

describe('usageServiceApi.syncModelPrices', () => {
  const syncResponse: ModelPriceSyncResponse = {
    source: 'model.dev',
    imported: 1,
    skipped: 0,
    prices: {},
  };

  beforeEach(() => {
    vi.mocked(axios.post).mockReset();
    vi.mocked(axios.post).mockResolvedValue({ data: syncResponse });
  });

  it('posts selected source and provider/model targets', async () => {
    const request: ModelPriceSyncRequest = {
      source: 'model.dev',
      models: [{ provider: 'openai', model: 'gpt-test' }],
    };

    const response = await usageServiceApi.syncModelPrices(
      'http://127.0.0.1:8317/v0/management',
      'manage-secret',
      request
    );

    expect(response).toBe(syncResponse);
    expect(axios.post).toHaveBeenCalledWith(
      'http://127.0.0.1:8317/v0/management/model-prices/sync',
      request,
      expect.objectContaining({
        headers: { Authorization: 'Bearer manage-secret' },
      })
    );
  });
});
