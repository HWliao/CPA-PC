import { describe, expect, it } from 'vitest';
import { resolveRequestMonitoringState } from './useRequestMonitoringAvailability';

describe('resolveRequestMonitoringState', () => {
  it('treats configured embedded CPA-PC as available without a stored CPA management key', () => {
    expect(
      resolveRequestMonitoringState({
        info: { service: 'cpa-pc', mode: 'embedded', configured: true },
        collectorEnabled: true,
        hasCPAConnection: false,
      })
    ).toEqual({ available: true, reason: '' });
  });

  it('keeps unconfigured embedded Usage Service unavailable', () => {
    expect(
      resolveRequestMonitoringState({
        info: { service: 'cpa-pc', mode: 'embedded', configured: false },
        collectorEnabled: true,
        hasCPAConnection: false,
      })
    ).toEqual({ available: false, reason: 'service_not_configured' });
  });

  it('requires CPA usage statistics only for external collectors when the service reports it disabled', () => {
    expect(
      resolveRequestMonitoringState({
        info: { service: 'cpa-manager', configured: true },
        collectorEnabled: true,
        hasCPAConnection: true,
        cpaUsageEnabled: false,
      })
    ).toEqual({ available: false, reason: 'monitoring_disabled' });
  });
});
