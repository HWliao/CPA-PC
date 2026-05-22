import { useEffect, useMemo, useState } from 'react';
import {
  isUsageServiceId,
  normalizeUsageServiceBase,
  type UsageServiceInfo,
  usageServiceApi,
} from '@/services/api/usageService';
import { useAuthStore, useUsageServiceStore } from '@/stores';
import { detectApiBaseFromLocation } from '@/utils/connection';

export type RequestMonitoringUnavailableReason =
  | 'checking'
  | 'service_not_configured'
  | 'service_unavailable'
  | 'monitoring_disabled';

export interface RequestMonitoringAvailability {
  checking: boolean;
  available: boolean;
  serviceBase: string;
  reason: RequestMonitoringUnavailableReason | '';
}

interface RequestMonitoringStateInput {
  info?: UsageServiceInfo | null;
  collectorEnabled: boolean;
  hasCPAConnection: boolean;
  cpaUsageEnabled?: boolean;
}

export function resolveRequestMonitoringState({
  info,
  collectorEnabled,
  hasCPAConnection,
  cpaUsageEnabled,
}: RequestMonitoringStateInput): Pick<RequestMonitoringAvailability, 'available' | 'reason'> {
  const embeddedConfigured = info?.mode === 'embedded' && info.configured === true;
  const cpaUsageAvailable = cpaUsageEnabled !== false || embeddedConfigured;

  if (!collectorEnabled || !cpaUsageAvailable) {
    return { available: false, reason: 'monitoring_disabled' };
  }
  if (hasCPAConnection || embeddedConfigured) {
    return { available: true, reason: '' };
  }
  return { available: false, reason: 'service_not_configured' };
}

export function useRequestMonitoringAvailability(): RequestMonitoringAvailability {
  const apiBase = useAuthStore((state) => state.apiBase);
  const managementKey = useAuthStore((state) => state.managementKey);
  const usageServiceEnabled = useUsageServiceStore((state) => state.enabled);
  const usageServiceBase = useUsageServiceStore((state) => state.serviceBase);
  const usageServiceRevision = useUsageServiceStore((state) => state.revision);
  const [state, setState] = useState<RequestMonitoringAvailability>({
    checking: true,
    available: false,
    serviceBase: '',
    reason: 'checking',
  });

  const candidates = useMemo(() => {
    return Array.from(
      new Set(
        [
          usageServiceEnabled && usageServiceBase ? usageServiceBase : '',
          apiBase,
          detectApiBaseFromLocation(),
        ]
          .map((value) => normalizeUsageServiceBase(value || ''))
          .filter(Boolean)
      )
    );
  }, [apiBase, usageServiceBase, usageServiceEnabled]);

  useEffect(() => {
    let cancelled = false;
    // Read revision so explicit Usage Service config updates retrigger detection.
    void usageServiceRevision;

    const detect = async () => {
      if (!managementKey || candidates.length === 0) {
        setState({
          checking: false,
          available: false,
          serviceBase: '',
          reason: 'service_not_configured',
        });
        return;
      }

      setState((current) => ({ ...current, checking: true, reason: 'checking' }));
      const hasConfiguredUsageService = Boolean(usageServiceEnabled && usageServiceBase);

      for (const candidate of candidates) {
        try {
          const info = await usageServiceApi.getInfo(candidate);
          if (!isUsageServiceId(info.service)) {
            continue;
          }
          const response = await usageServiceApi.getManagerConfig(candidate, managementKey);
          const collectorEnabled = response.config.collector?.enabled !== false;
          const hasCPAConnection = Boolean(
            response.config.cpaConnection?.cpaBaseUrl &&
              response.config.cpaConnection?.managementKey
          );
          const resolved = resolveRequestMonitoringState({
            info,
            collectorEnabled,
            hasCPAConnection,
            cpaUsageEnabled: response.cpaUsage?.usageStatisticsEnabled,
          });
          if (cancelled) return;
          setState({
            checking: false,
            available: resolved.available,
            serviceBase: candidate,
            reason: resolved.reason,
          });
          return;
        } catch {
          // A regular CPA panel or an unreachable external Usage Service is handled below.
        }
      }

      if (cancelled) return;
      setState({
        checking: false,
        available: false,
        serviceBase: '',
        reason: hasConfiguredUsageService ? 'service_unavailable' : 'service_not_configured',
      });
    };

    void detect();

    return () => {
      cancelled = true;
    };
  }, [candidates, managementKey, usageServiceBase, usageServiceEnabled, usageServiceRevision]);

  return state;
}
