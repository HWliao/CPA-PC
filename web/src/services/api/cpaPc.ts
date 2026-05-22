import axios from 'axios';
import { REQUEST_TIMEOUT_MS } from '@/utils/constants';
import { normalizeApiBase } from '@/utils/connection';

export interface CpaPcInfo {
  service?: string;
  mode?: string;
  version?: string;
  buildDate?: string;
  startedAt?: number;
  configured?: boolean;
  cliProxyApi?: {
    version?: string;
  };
}

const buildUrl = (base: string, path: string): string => {
  const normalized = normalizeApiBase(base).replace(/\/+$/, '');
  return `${normalized}${path}`;
};

export const cpaPcApi = {
  async getInfo(base: string): Promise<CpaPcInfo> {
    const response = await axios.get<CpaPcInfo>(buildUrl(base, '/cpa-pc/info'), {
      timeout: REQUEST_TIMEOUT_MS,
    });
    return response.data;
  },
};
