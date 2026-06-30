/**
 * 版本相关 API
 */

import axios from 'axios';
import { REQUEST_TIMEOUT_MS } from '@/utils/constants';
import { apiClient } from './client';
import type { ServerRuntimeKind } from '@/types';
import { isRecord } from '@/utils/helpers';

export interface ManagerLatestRelease {
  tag_name?: string;
  name?: string;
  html_url?: string;
  published_at?: string;
  [key: string]: unknown;
}

const CPA_PC_LATEST_RELEASE_URL = 'https://api.github.com/repos/HWliao/CPA-PC/releases/latest';

export const versionApi = {
  checkLatest: () => apiClient.get<Record<string, unknown>>('/latest-version'),

  async detectRuntimeKind(): Promise<ServerRuntimeKind> {
    try {
      const data = await apiClient.get('/nodes');
      return isRecord(data) && Array.isArray(data.nodes) ? 'home' : 'unknown';
    } catch (error: unknown) {
      const status = isRecord(error) ? error.status : undefined;
      if (status === 404 || status === 405) {
        return 'cpa';
      }
      return 'unknown';
    }
  },

  checkManagerLatest: async () => {
    const response = await axios.get<ManagerLatestRelease>(CPA_PC_LATEST_RELEASE_URL, {
      timeout: REQUEST_TIMEOUT_MS,
      headers: {
        Accept: 'application/vnd.github+json',
      },
    });
    return response.data;
  },
};
