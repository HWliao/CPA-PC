import { isUsageServiceId, type UsageServiceInfo } from '@/services/api/usageService';

export const resolveUsageServiceLoginMode = (info?: UsageServiceInfo | null) => {
  const hostedByUsageService = isUsageServiceId(info?.service);
  const embeddedUsageService = hostedByUsageService && info?.mode === 'embedded';
  return {
    hostedByUsageService,
    embeddedUsageService,
    usageServiceNeedsSetup: hostedByUsageService && info?.configured !== true,
  };
};
