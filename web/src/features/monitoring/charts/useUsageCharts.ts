import { useCallback, useEffect, useRef, useState } from 'react';
import {
  isUsageServiceId,
  normalizeUsageServiceBase,
  usageServiceApi,
  type UsageChartsQueryParams,
  type UsageChartsResponse,
  type UsageServiceInfo,
} from '@/services/api/usageService';
import { useAuthStore, useUsageServiceStore } from '@/stores';
import { detectApiBaseFromLocation } from '@/utils/connection';

export interface ResolveUsageChartsServiceBaseInput {
  usageServiceEnabled: boolean;
  usageServiceBase: string;
  apiBase: string;
  locationBase: string;
  getInfo: (base: string) => Promise<UsageServiceInfo>;
}

export interface UseUsageChartsReturn {
  charts: UsageChartsResponse | null;
  loading: boolean;
  error: string;
  lastRefreshedAt: Date | null;
  usageServiceAvailable: boolean;
  loadCharts: () => Promise<void>;
}

export async function resolveUsageChartsServiceBase({
  usageServiceEnabled,
  usageServiceBase,
  apiBase,
  locationBase,
  getInfo,
}: ResolveUsageChartsServiceBaseInput): Promise<string> {
  if (usageServiceEnabled && usageServiceBase) {
    return normalizeUsageServiceBase(usageServiceBase);
  }

  const candidates = Array.from(
    new Set(
      [apiBase, locationBase]
        .map((value) => normalizeUsageServiceBase(value || ''))
        .filter(Boolean)
    )
  );

  for (const candidate of candidates) {
    try {
      const info = await getInfo(candidate);
      if (isUsageServiceId(info.service)) {
        return candidate;
      }
    } catch {
      // Non-CPA-PC management APIs do not expose usage-service metadata.
    }
  }

  return '';
}

export function useUsageCharts(params: UsageChartsQueryParams = {}): UseUsageChartsReturn {
  const apiBase = useAuthStore((state) => state.apiBase);
  const managementKey = useAuthStore((state) => state.managementKey);
  const usageServiceEnabled = useUsageServiceStore((state) => state.enabled);
  const usageServiceBase = useUsageServiceStore((state) => state.serviceBase);
  const [charts, setCharts] = useState<UsageChartsResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [lastRefreshedAt, setLastRefreshedAt] = useState<Date | null>(null);
  const [usageServiceAvailable, setUsageServiceAvailable] = useState(false);
  const requestIdRef = useRef(0);

  const loadCharts = useCallback(async () => {
    const requestId = requestIdRef.current + 1;
    requestIdRef.current = requestId;
    setLoading(true);
    setError('');

    try {
      const serviceBase = await resolveUsageChartsServiceBase({
        usageServiceEnabled,
        usageServiceBase,
        apiBase,
        locationBase: detectApiBaseFromLocation(),
        getInfo: usageServiceApi.getInfo,
      });
      if (!serviceBase) {
        if (requestIdRef.current !== requestId) return;
        setUsageServiceAvailable(false);
        setCharts(null);
        setLastRefreshedAt(null);
        return;
      }

      const response = await usageServiceApi.getUsageCharts(serviceBase, params, managementKey);
      if (requestIdRef.current !== requestId) return;
      setUsageServiceAvailable(true);
      setCharts(response);
      setLastRefreshedAt(new Date());
    } catch (err) {
      if (requestIdRef.current !== requestId) return;
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      if (requestIdRef.current === requestId) {
        setLoading(false);
      }
    }
  }, [apiBase, managementKey, params, usageServiceBase, usageServiceEnabled]);

  useEffect(() => {
    void loadCharts();
  }, [loadCharts]);

  return {
    charts,
    loading,
    error,
    lastRefreshedAt,
    usageServiceAvailable,
    loadCharts,
  };
}
