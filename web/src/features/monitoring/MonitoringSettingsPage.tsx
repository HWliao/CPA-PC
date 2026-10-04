import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Link } from 'react-router-dom';
import { Button } from '@/components/ui/Button';
import { Input } from '@/components/ui/Input';
import { ToggleSwitch } from '@/components/ui/ToggleSwitch';
import { useAuthStore, useNotificationStore, useUsageServiceStore } from '@/stores';
import { type ManagerConfig, usageServiceApi } from '@/services/api/usageService';
import { detectApiBaseFromLocation } from '@/utils/connection';
import { getUsageServiceBootstrapToSync } from './usageServiceBootstrap';
import styles from './MonitoringSettingsPage.module.scss';

export function MonitoringSettingsPage() {
  const { t } = useTranslation();
  const apiBase = useAuthStore((state) => state.apiBase);
  const managementKey = useAuthStore((state) => state.managementKey);
  const connected = useAuthStore((state) => state.connectionStatus === 'connected');
  const usageServiceEnabled = useUsageServiceStore((state) => state.enabled);
  const usageServiceBase = useUsageServiceStore((state) => state.serviceBase);
  const setUsageServiceConfig = useUsageServiceStore((state) => state.setUsageServiceConfig);
  const showNotification = useNotificationStore((state) => state.showNotification);
  const base = usageServiceEnabled && usageServiceBase ? usageServiceBase : apiBase || detectApiBaseFromLocation();

  const [config, setConfig] = useState<ManagerConfig | null>(null);
  const [enabled, setEnabled] = useState(true);
  const [queryLimit, setQueryLimit] = useState('50000');
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    if (!managementKey || !base) return;
    setLoading(true);
    setError('');
    try {
      const response = await usageServiceApi.getManagerConfig(base, managementKey);
      setConfig(response.config);
      setEnabled(response.config.collector?.enabled !== false);
      setQueryLimit(String(response.config.collector?.queryLimit || 50000));
      const bootstrap = getUsageServiceBootstrapToSync({ serviceBase: base, usageServiceEnabled, usageServiceBase });
      if (bootstrap) setUsageServiceConfig({ enabled: true, serviceBase: bootstrap });
    } catch (err) {
      setError(err instanceof Error ? err.message : t('config_management.manager.load_failed'));
    } finally {
      setLoading(false);
    }
  }, [base, managementKey, setUsageServiceConfig, t, usageServiceBase, usageServiceEnabled]);

  useEffect(() => {
    void load();
  }, [load]);

  const save = async () => {
    if (!config || !base) return;
    const limit = Number(queryLimit);
    if (!Number.isSafeInteger(limit) || limit <= 0) {
      setError(t('config_management.manager.number_invalid', { label: t('config_management.manager.query_limit') }));
      return;
    }
    setSaving(true);
    setError('');
    try {
      const response = await usageServiceApi.saveManagerConfig(base, {
        ...config,
        collector: { ...config.collector, enabled, queryLimit: limit },
      }, managementKey);
      setConfig(response.config);
      setUsageServiceConfig({ enabled: true, serviceBase: base });
      showNotification(t('config_management.manager.save_success'), 'success');
    } catch (err) {
      setError(err instanceof Error ? err.message : t('config_management.manager.save_failed'));
    } finally {
      setSaving(false);
    }
  };

  return (
    <main className={styles.page}>
      <Link to="/monitoring">{t('monitoring.charts_back')}</Link>
      <h1>{t('config_management.manager.title')}</h1>
      <p>{t('config_management.manager.description')}</p>
      {error && <div className="error-box" role="alert">{error}</div>}
      <div className={styles.controls}>
        <ToggleSwitch
          label={t('config_management.manager.request_monitoring_enabled')}
          checked={enabled}
          onChange={setEnabled}
          disabled={!connected || loading || saving || !config}
        />
        <Input
          label={t('config_management.manager.query_limit')}
          type="number"
          min="1"
          value={queryLimit}
          onChange={(event) => setQueryLimit(event.target.value)}
          disabled={!connected || loading || saving || !config}
        />
        <div className={styles.actions}>
          <Button variant="secondary" onClick={() => void load()} disabled={loading || saving}>
            {t('common.refresh')}
          </Button>
          <Button onClick={() => void save()} disabled={!connected || loading || saving || !config}>
            {t('common.save')}
          </Button>
        </div>
      </div>
    </main>
  );
}
