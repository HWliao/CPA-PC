import { useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/Button';
import { Card } from '@/components/ui/Card';
import { LoadingSpinner } from '@/components/ui/LoadingSpinner';
import { Select } from '@/components/ui/Select';
import { IconChartLine, IconExternalLink, IconRefreshCw } from '@/components/ui/icons';
import { EChartPanel } from '@/features/monitoring/charts/EChartPanel';
import { buildGlobalUsageChartOption } from '@/features/monitoring/charts/chartOptions';
import {
  USAGE_CHART_GRANULARITY_OPTIONS,
  USAGE_CHART_RANGE_OPTIONS,
  buildUsageChartsQueryParams,
  createDefaultUsageChartsFilterState,
  resolveDefaultUsageChartsGranularity,
  type UsageChartsFilterState,
} from '@/features/monitoring/charts/filters';
import { useUsageCharts } from '@/features/monitoring/charts/useUsageCharts';
import type { UsageChartsGranularity, UsageChartsRange } from '@/services/api/usageService';
import { formatCompactNumber, formatUsd } from '@/utils/usage';
import styles from './MonitoringChartsPage.module.scss';

export function MonitoringChartsPage() {
  const { t, i18n } = useTranslation();
  const [filterState, setFilterState] = useState(createDefaultUsageChartsFilterState);
  const chartParams = useMemo(() => buildUsageChartsQueryParams(filterState), [filterState]);
  const { charts, loading, error, lastRefreshedAt, usageServiceAvailable, loadCharts } =
    useUsageCharts(chartParams);

  const globalBuckets = charts?.global.buckets ?? [];
  const globalBucketCount = globalBuckets.length;
  const globalTotals = globalBuckets.reduce(
    (totals, bucket) => ({
      inputTokens: totals.inputTokens + bucket.inputTokens,
      outputTokens: totals.outputTokens + bucket.outputTokens,
      cachedTokens: totals.cachedTokens + bucket.cachedTokens,
      totalCost: totals.totalCost + bucket.totalCost,
    }),
    { inputTokens: 0, outputTokens: 0, cachedTokens: 0, totalCost: 0 }
  );
  const hasData = Boolean(
    charts &&
      (globalBucketCount > 0 ||
        charts.byProviderAuthFile.series.length > 0 ||
        charts.byApiKey.series.length > 0 ||
        charts.byModel.series.length > 0)
  );
  const missingPriceModels = charts?.missingPriceModels ?? [];
  const providerOptions = [
    { value: '', label: t('monitoring.charts_filter_all_providers', { defaultValue: 'All providers' }) },
    ...(charts?.options.providers ?? []).map((provider) => ({ value: provider, label: provider })),
  ];
  const authFileOptions = [
    { value: '', label: t('monitoring.charts_filter_all_auth_files', { defaultValue: 'All auth files' }) },
    ...(charts?.options.authFiles ?? []).map((item) => ({ value: item.authIndex, label: item.label })),
  ];
  const apiKeyOptions = [
    { value: '', label: t('monitoring.charts_filter_all_api_keys', { defaultValue: 'All API keys' }) },
    ...(charts?.options.apiKeys ?? []).map((item) => ({ value: item.apiKeyHash, label: item.label })),
  ];
  const modelOptions = [
    { value: '', label: t('monitoring.charts_filter_all_models', { defaultValue: 'All models' }) },
    ...(charts?.options.models ?? []).map((model) => ({ value: model, label: model })),
  ];
  const globalTokenTitle = t('monitoring.charts_global_tokens', { defaultValue: 'Global tokens' });
  const globalCostTitle = t('monitoring.charts_global_cost', { defaultValue: 'Global cost' });
  const globalTpmTitle = t('monitoring.charts_global_tpm', { defaultValue: 'Global TPM' });
  const statusTone = error ? 'bad' : loading ? 'info' : usageServiceAvailable ? 'good' : 'warn';
  const statusLabel = error
    ? t('monitoring.charts_status_error', { defaultValue: 'Chart load failed' })
    : loading
      ? t('monitoring.charts_status_loading', { defaultValue: 'Loading chart data' })
      : usageServiceAvailable
        ? t('monitoring.charts_status_ready', { defaultValue: 'Charts ready' })
        : t('monitoring.charts_status_unavailable', { defaultValue: 'Usage service unavailable' });

  const handleRangeChange = (value: string) => {
    const range = value as UsageChartsRange;
    setFilterState((current) => ({
      ...current,
      range,
      granularity: resolveDefaultUsageChartsGranularity(range),
    }));
  };
  const handleGranularityChange = (value: string) => {
    setFilterState((current) => ({ ...current, granularity: value as UsageChartsGranularity }));
  };
  const handleFilterChange = (key: keyof Pick<UsageChartsFilterState, 'provider' | 'authIndex' | 'apiKeyHash' | 'model'>) =>
    (value: string) => {
      setFilterState((current) => ({ ...current, [key]: value }));
    };

  return (
    <div className={styles.page}>
      <div className={styles.pageHeader}>
        <h1 className={styles.pageTitle}>
          {t('monitoring.charts_title', { defaultValue: 'Monitoring Charts' })}
        </h1>
        <p className={styles.description}>
          {t('monitoring.charts_desc', {
            defaultValue: 'Visualize token, cost, and TPM trends from local request monitoring data.',
          })}
        </p>
      </div>

      <Card className={`${styles.panel} ${styles.statusPanel}`}>
        <div className={styles.statusBar}>
          <div className={styles.statusInfo}>
            <span className={`${styles.statusBadge} ${styles[`tone-${statusTone}`]}`}>
              <span className={styles.statusDot} aria-hidden="true" />
              {statusLabel}
            </span>
            <div className={styles.statusMeta}>
              <span>
                {`${t('monitoring.charts_range_default', { defaultValue: 'Range' })}: ${charts?.range ?? '1h'}`}
              </span>
              <span>
                {`${t('monitoring.charts_bucket_count', { defaultValue: 'Buckets' })}: ${globalBucketCount}`}
              </span>
              <span>
                {`${t('monitoring.last_sync')}: ${lastRefreshedAt ? lastRefreshedAt.toLocaleTimeString(i18n.language) : '--'}`}
              </span>
            </div>
          </div>

          <div className={styles.statusActions}>
            <Link to="/monitoring" className={styles.quickLink}>
              <IconExternalLink size={14} />
              <span>{t('monitoring.charts_back', { defaultValue: 'Back to Request Monitoring' })}</span>
            </Link>
            <Button variant="secondary" onClick={() => void loadCharts()} disabled={loading}>
              <IconRefreshCw size={14} />
              <span>{t('usage_stats.refresh')}</span>
            </Button>
          </div>
        </div>
      </Card>

      <Card className={`${styles.panel} ${styles.filterPanel}`}>
        <div className={styles.filterHeader}>
          <strong>{t('monitoring.charts_controls_title', { defaultValue: 'Chart controls' })}</strong>
          <span>
            {t('monitoring.charts_controls_desc', {
              defaultValue: 'Choose a fixed time range and combine dimensions to reload the chart data.',
            })}
          </span>
        </div>
        <div className={styles.filterGrid}>
          <div className={styles.filterField}>
            <span>{t('monitoring.charts_range_label', { defaultValue: 'Time range' })}</span>
            <Select
              ariaLabel={t('monitoring.charts_range_label', { defaultValue: 'Time range' })}
              value={filterState.range}
              options={USAGE_CHART_RANGE_OPTIONS.map((option) => ({
                value: option.value,
                label: t(option.labelKey, { defaultValue: option.defaultLabel }),
              }))}
              onChange={handleRangeChange}
            />
          </div>
          <div className={styles.filterField}>
            <span>{t('monitoring.charts_granularity_label', { defaultValue: 'Granularity' })}</span>
            <Select
              ariaLabel={t('monitoring.charts_granularity_label', { defaultValue: 'Granularity' })}
              value={filterState.granularity}
              options={USAGE_CHART_GRANULARITY_OPTIONS.map((option) => ({
                value: option.value,
                label: t(option.labelKey, { defaultValue: option.defaultLabel }),
              }))}
              onChange={handleGranularityChange}
            />
          </div>
          <div className={styles.filterField}>
            <span>{t('monitoring.charts_provider_label', { defaultValue: 'Provider' })}</span>
            <Select
              ariaLabel={t('monitoring.charts_provider_label', { defaultValue: 'Provider' })}
              value={filterState.provider}
              options={providerOptions}
              onChange={handleFilterChange('provider')}
            />
          </div>
          <div className={styles.filterField}>
            <span>{t('monitoring.charts_auth_file_label', { defaultValue: 'Auth file' })}</span>
            <Select
              ariaLabel={t('monitoring.charts_auth_file_label', { defaultValue: 'Auth file' })}
              value={filterState.authIndex}
              options={authFileOptions}
              onChange={handleFilterChange('authIndex')}
            />
          </div>
          <div className={styles.filterField}>
            <span>{t('monitoring.charts_api_key_label', { defaultValue: 'API key' })}</span>
            <Select
              ariaLabel={t('monitoring.charts_api_key_label', { defaultValue: 'API key' })}
              value={filterState.apiKeyHash}
              options={apiKeyOptions}
              onChange={handleFilterChange('apiKeyHash')}
            />
          </div>
          <div className={styles.filterField}>
            <span>{t('monitoring.charts_model_label', { defaultValue: 'Model' })}</span>
            <Select
              ariaLabel={t('monitoring.charts_model_label', { defaultValue: 'Model' })}
              value={filterState.model}
              options={modelOptions}
              onChange={handleFilterChange('model')}
            />
          </div>
        </div>
      </Card>

      {loading ? (
        <Card className={styles.statePanel}>
          <LoadingSpinner size={32} />
          <strong>{t('monitoring.charts_loading', { defaultValue: 'Loading chart data' })}</strong>
        </Card>
      ) : error ? (
        <Card className={`${styles.statePanel} ${styles.errorPanel}`}>
          <strong>{t('monitoring.charts_error_title', { defaultValue: 'Unable to load charts' })}</strong>
          <span>{error}</span>
        </Card>
      ) : !hasData ? (
        <Card className={styles.statePanel}>
          <IconChartLine size={22} />
          <strong>{t('monitoring.charts_empty_title', { defaultValue: 'No chart data yet' })}</strong>
          <span>
            {t('monitoring.charts_empty_desc', {
              defaultValue: 'Recent usage events will appear here after requests are recorded.',
            })}
          </span>
        </Card>
      ) : (
        <>
          <section className={styles.summaryGrid}>
            <Card className={styles.summaryCard}>
              <span>{t('monitoring.input_tokens')}</span>
              <strong>{formatCompactNumber(globalTotals.inputTokens)}</strong>
            </Card>
            <Card className={styles.summaryCard}>
              <span>{t('monitoring.output_tokens')}</span>
              <strong>{formatCompactNumber(globalTotals.outputTokens)}</strong>
            </Card>
            <Card className={styles.summaryCard}>
              <span>{t('monitoring.cached_tokens')}</span>
              <strong>{formatCompactNumber(globalTotals.cachedTokens)}</strong>
            </Card>
            <Card className={styles.summaryCard}>
              <span>{t('monitoring.estimated_cost')}</span>
              <strong>{formatUsd(globalTotals.totalCost)}</strong>
            </Card>
          </section>

          {missingPriceModels.length > 0 ? (
            <Card className={styles.warningPanel}>
              <div>
                <strong>
                  {t('monitoring.charts_missing_prices_title', { defaultValue: 'Missing model prices' })}
                </strong>
                <span>
                  {t('monitoring.charts_missing_prices_desc', {
                    defaultValue: 'Cost charts may be incomplete until prices are configured for these models.',
                  })}
                </span>
              </div>
              <div className={styles.warningList}>
                {missingPriceModels.map((model) => (
                  <code key={model}>{model}</code>
                ))}
              </div>
            </Card>
          ) : null}

          <section className={styles.chartGrid}>
            <Card className={styles.chartCard}>
              <div className={styles.chartHeader}>
                <h2>{globalTokenTitle}</h2>
                <span>
                  {t('monitoring.charts_global_tokens_desc', {
                    defaultValue: 'Input, output, and cached tokens by bucket.',
                  })}
                </span>
              </div>
              <EChartPanel
                ariaLabel={globalTokenTitle}
                className={styles.chartCanvas}
                option={buildGlobalUsageChartOption({
                  title: globalTokenTitle,
                  family: 'tokens',
                  buckets: globalBuckets,
                })}
              />
            </Card>
            <Card className={styles.chartCard}>
              <div className={styles.chartHeader}>
                <h2>{globalCostTitle}</h2>
                <span>
                  {t('monitoring.charts_global_cost_desc', {
                    defaultValue: 'Estimated spend by bucket based on configured model prices.',
                  })}
                </span>
              </div>
              <EChartPanel
                ariaLabel={globalCostTitle}
                className={styles.chartCanvas}
                option={buildGlobalUsageChartOption({
                  title: globalCostTitle,
                  family: 'cost',
                  buckets: globalBuckets,
                })}
              />
            </Card>
            <Card className={styles.chartCard}>
              <div className={styles.chartHeader}>
                <h2>{globalTpmTitle}</h2>
                <span>
                  {t('monitoring.charts_global_tpm_desc', {
                    defaultValue: 'Input, output, and cached tokens per minute.',
                  })}
                </span>
              </div>
              <EChartPanel
                ariaLabel={globalTpmTitle}
                className={styles.chartCanvas}
                option={buildGlobalUsageChartOption({
                  title: globalTpmTitle,
                  family: 'tpm',
                  buckets: globalBuckets,
                })}
              />
            </Card>
          </section>
        </>
      )}
    </div>
  );
}
