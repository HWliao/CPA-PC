import { normalizeUsageServiceBase } from '@/services/api/usageService';

export function getUsageServiceBootstrapToSync({
  serviceBase,
  usageServiceEnabled,
  usageServiceBase,
}: {
  serviceBase: string;
  usageServiceEnabled: boolean;
  usageServiceBase: string;
}): string {
  const normalized = normalizeUsageServiceBase(serviceBase);
  if (!normalized || (usageServiceEnabled && usageServiceBase === normalized)) return '';
  return normalized;
}
