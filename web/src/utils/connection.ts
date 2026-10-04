import { DEFAULT_API_PORT, MANAGEMENT_API_PREFIX } from './constants';

export const DEFAULT_DOCKER_CPA_BASE_URL = 'http://host.docker.internal:8317';

export const normalizeApiBase = (input: string): string => {
  let base = (input || '').trim();
  if (!base) return '';
  base = base.replace(/\/+$/i, '');
  base = base.replace(/\/v[08]\/management$/i, '');
  if (!/^https?:\/\//i.test(base)) {
    base = `http://${base}`;
  }
  return base;
};

const readEnvDefaultCPAConnectionBase = (): string => {
  try {
    return import.meta.env.VITE_DEFAULT_CPA_BASE_URL || '';
  } catch {
    return '';
  }
};

export const resolveDefaultCPAConnectionBase = (options?: {
  hostedByUsageService?: boolean;
  embeddedUsageService?: boolean;
  currentBase?: string;
  envDefault?: string;
}): string => {
  const envDefault = normalizeApiBase(
    options?.envDefault === undefined ? readEnvDefaultCPAConnectionBase() : options.envDefault
  );
  if (envDefault) return envDefault;
  if (options?.hostedByUsageService && !options.embeddedUsageService) {
    return DEFAULT_DOCKER_CPA_BASE_URL;
  }
  return normalizeApiBase(options?.currentBase || '');
};

export const computeApiUrl = (base: string): string => {
  const normalized = normalizeApiBase(base);
  if (!normalized) return '';
  return `${normalized}${MANAGEMENT_API_PREFIX}`;
};

export const detectApiBaseFromLocation = (): string => {
  try {
    const { protocol, hostname, port } = window.location;
    const normalizedPort = port ? `:${port}` : '';
    return normalizeApiBase(`${protocol}//${hostname}${normalizedPort}`);
  } catch (error) {
    console.warn('Failed to detect api base from location, fallback to default', error);
    return normalizeApiBase(`http://localhost:${DEFAULT_API_PORT}`);
  }
};
