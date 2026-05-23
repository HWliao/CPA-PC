import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/Button';
import { Card } from '@/components/ui/Card';
import { LoadingSpinner } from '@/components/ui/LoadingSpinner';
import { IconChartLine, IconExternalLink, IconRefreshCw } from '@/components/ui/icons';
import { useUsageCharts } from '@/features/monitoring/charts/useUsageCharts';
import { formatCompactNumber, formatUsd } from '@/utils/usage';
import styles from './MonitoringChartsPage.module.scss';

const DEFAULT_CHART_PARAMS = {
  range: '1h',
  granularity: 'hour',
} as const;

export function MonitoringChartsPage() {
  const { t, i18n } = useTranslation();
  const { charts, loading, error, lastRefreshedAt, usageServiceAvailable, loadCharts } =
    useUsageCharts(DEFAULT_CHART_PARAMS);

  const globalBucketCount = charts?.global.buckets.length ?? 0;
  const globalTotals = charts?.global.buckets.reduce(
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
  const statusTone = error ? 'bad' : loading ? 'info' : usageServiceAvailable ? 'good' : 'warn';
  const statusLabel = error
    ? t('monitoring.charts_status_error', { defaultValue: 'Chart load failed' })
    : loading
      ? t('monitoring.charts_status_loading', { defaultValue: 'Loading chart data' })
      : usageServiceAvailable
        ? t('monitoring.charts_status_ready', { defaultValue: 'Charts ready' })
        : t('monitoring.charts_status_unavailable', { defaultValue: 'Usage service unavailable' });

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
        <section className={styles.summaryGrid}>
          <Card className={styles.summaryCard}>
            <span>{t('monitoring.input_tokens')}</span>
            <strong>{formatCompactNumber(globalTotals?.inputTokens ?? 0)}</strong>
          </Card>
          <Card className={styles.summaryCard}>
            <span>{t('monitoring.output_tokens')}</span>
            <strong>{formatCompactNumber(globalTotals?.outputTokens ?? 0)}</strong>
          </Card>
          <Card className={styles.summaryCard}>
            <span>{t('monitoring.cached_tokens')}</span>
            <strong>{formatCompactNumber(globalTotals?.cachedTokens ?? 0)}</strong>
          </Card>
          <Card className={styles.summaryCard}>
            <span>{t('monitoring.estimated_cost')}</span>
            <strong>{formatUsd(globalTotals?.totalCost ?? 0)}</strong>
          </Card>
        </section>
      )}
    </div>
  );
}
