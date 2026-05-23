import { describe, expect, it, vi } from 'vitest';
import { resolveUsageChartsServiceBase } from './useUsageCharts';

describe('resolveUsageChartsServiceBase', () => {
  it('uses explicitly configured usage service base first', async () => {
    const getInfo = vi.fn();

    const base = await resolveUsageChartsServiceBase({
      usageServiceEnabled: true,
      usageServiceBase: 'http://usage.local:9000/',
      apiBase: 'http://cpa.local:8317',
      locationBase: 'http://localhost:8317',
      getInfo,
    });

    expect(base).toBe('http://usage.local:9000');
    expect(getInfo).not.toHaveBeenCalled();
  });

  it('detects embedded cpa-pc usage service from candidates', async () => {
    const getInfo = vi.fn(async (base: string) => ({
      service: base.includes('127.0.0.1') ? 'cpa-pc' : 'cli-proxy-api',
    }));

    const base = await resolveUsageChartsServiceBase({
      usageServiceEnabled: false,
      usageServiceBase: '',
      apiBase: 'http://127.0.0.1:8317',
      locationBase: 'http://localhost:8317',
      getInfo,
    });

    expect(base).toBe('http://127.0.0.1:8317');
    expect(getInfo).toHaveBeenCalledWith('http://127.0.0.1:8317');
  });

  it('returns an empty base when no candidate exposes usage service metadata', async () => {
    const getInfo = vi.fn(async () => ({ service: 'cli-proxy-api' }));

    const base = await resolveUsageChartsServiceBase({
      usageServiceEnabled: false,
      usageServiceBase: '',
      apiBase: 'http://127.0.0.1:8317',
      locationBase: 'http://localhost:8317',
      getInfo,
    });

    expect(base).toBe('');
  });
});
