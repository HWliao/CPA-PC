import { useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import type { EChartsCoreOption } from 'echarts/core';
import { Button } from '@/components/ui/Button';
import { Card } from '@/components/ui/Card';
import { LoadingSpinner } from '@/components/ui/LoadingSpinner';
import { Select } from '@/components/ui/Select';
import { IconChartLine, IconExternalLink, IconRefreshCw } from '@/components/ui/icons';
import { EChartPanel } from '@/features/monitoring/charts/EChartPanel';
import {
  buildGlobalUsageChartOption,
  buildSeriesUsageChartOption,
  formatChartMetricValue,
  type UsageChartMetricFamily,
} from '@/features/monitoring/charts/chartOptions';
import {
  USAGE_CHART_DIMENSION_OPTIONS,
  USAGE_CHART_RANGE_OPTIONS,
  buildUsageChartsQueryParams,
  createDefaultUsageChartsFilterState,
  shouldDisableUsageChartsFilter,
  type UsageChartsDimension,
  type UsageChartsFilterState,
} from '@/features/monitoring/charts/filters';
import { useUsageCharts } from '@/features/monitoring/charts/useUsageCharts';
import type { UsageChartMetricBucket, UsageChartSeries, UsageChartsRange } from '@/services/api/usageService';
import { formatUsd } from '@/utils/usage';
import styles from './MonitoringChartsPage.module.scss';

type TokenChartFamily = Extract<UsageChartMetricFamily, 'tokens' | 'cumulativeTokens'>;
type CostChartFamily = Extract<UsageChartMetricFamily, 'cost' | 'cumulativeCost'>;

type ChartDefinition = {
  description: string;
  family: UsageChartMetricFamily;
  title: string;
};

export function MonitoringChartsPage() {
  const { t, i18n } = useTranslation();
  const [filterState, setFilterState] = useState(createDefaultUsageChartsFilterState);
  const [activeTokenChart, setActiveTokenChart] = useState<TokenChartFamily>('tokens');
  const [activeCostChart, setActiveCostChart] = useState<CostChartFamily>('cost');
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
  const hasGlobalUsageValues = globalBuckets.some(
    (bucket) =>
      bucket.inputTokens !== 0 ||
      bucket.outputTokens !== 0 ||
      bucket.cachedTokens !== 0 ||
      bucket.totalCost !== 0 ||
      bucket.tpmInput !== 0 ||
      bucket.tpmOutput !== 0 ||
      bucket.tpmCached !== 0
  );
  const accountSeries = charts?.byAccount.series ?? [];
  const apiKeySeries = charts?.byApiKey.series ?? [];
  const modelSeries = charts?.byModel.series ?? [];
  const hasDimensionSeries = Boolean(
    charts && (accountSeries.length > 0 || apiKeySeries.length > 0 || modelSeries.length > 0)
  );
  const hasData = Boolean(charts && (hasGlobalUsageValues || hasDimensionSeries));
  const missingPriceModels = charts?.missingPriceModels ?? [];
  const accountOptions = [
    { value: '', label: t('monitoring.charts_filter_all_accounts', { defaultValue: 'All accounts' }) },
    ...(charts?.options.accounts ?? []).map((item) => ({ value: item.value, label: item.label })),
  ];
  const apiKeyOptions = [
    { value: '', label: t('monitoring.charts_filter_all_api_keys', { defaultValue: 'All caller keys' }) },
    ...(charts?.options.apiKeys ?? []).map((item) => ({ value: item.value, label: item.label })),
  ];
  const modelOptions = [
    { value: '', label: t('monitoring.charts_filter_all_models', { defaultValue: 'All models' }) },
    ...(charts?.options.models ?? []).map((item) => ({ value: item.value, label: item.label })),
  ];
  const dimensionOptions = USAGE_CHART_DIMENSION_OPTIONS.map((option) => ({
    value: option.value,
    label: t(option.labelKey, { defaultValue: option.defaultLabel }),
  }));
  const activeDimensionLabel =
    dimensionOptions.find((option) => option.value === filterState.dimension)?.label ??
    t('monitoring.charts_dimension_global', { defaultValue: 'Global total' });
  const activeSeries = resolveActiveDimensionSeries(filterState.dimension, {
    account: accountSeries,
    apiKey: apiKeySeries,
    model: modelSeries,
  });
  const hasActiveChartData = filterState.dimension === 'global' ? hasGlobalUsageValues : activeSeries.length > 0;
  const showAccountFilter = !shouldDisableUsageChartsFilter('account', filterState.dimension);
  const showApiKeyFilter = !shouldDisableUsageChartsFilter('apiKey', filterState.dimension);
  const showModelFilter = !shouldDisableUsageChartsFilter('model', filterState.dimension);
  const tokenUsageChart: ChartDefinition = {
    family: 'tokens',
    title: t('monitoring.charts_tokens_title', { defaultValue: 'Token usage' }),
    description: t('monitoring.charts_tokens_desc', {
      defaultValue: 'Input, output, and cached tokens by bucket.',
    }),
  };
  const cumulativeTokenUsageChart: ChartDefinition = {
    family: 'cumulativeTokens',
    title: t('monitoring.charts_cumulative_tokens_title', {
      defaultValue: 'Cumulative token usage',
    }),
    description: t('monitoring.charts_cumulative_tokens_desc', {
      defaultValue: 'Running input, output, and cached token totals over time.',
    }),
  };
  const costUsageChart: ChartDefinition = {
    family: 'cost',
    title: t('monitoring.charts_cost_title', { defaultValue: 'Cost' }),
    description: t('monitoring.charts_cost_desc', {
      defaultValue: 'Estimated spend by bucket based on configured model prices.',
    }),
  };
  const cumulativeCostUsageChart: ChartDefinition = {
    family: 'cumulativeCost',
    title: t('monitoring.charts_cumulative_cost_title', { defaultValue: 'Cumulative cost' }),
    description: t('monitoring.charts_cumulative_cost_desc', {
      defaultValue: 'Running estimated spend totals over time.',
    }),
  };
  const tpmUsageChart: ChartDefinition = {
    family: 'tpm',
    title: t('monitoring.charts_tpm_title', { defaultValue: 'TPM' }),
    description: t('monitoring.charts_tpm_desc', {
      defaultValue: 'Input, output, and cached tokens per minute.',
    }),
  };
  const activeTokenChartDefinition =
    activeTokenChart === 'tokens' ? tokenUsageChart : cumulativeTokenUsageChart;
  const activeCostChartDefinition = activeCostChart === 'cost' ? costUsageChart : cumulativeCostUsageChart;
  const statusTone = error ? 'bad' : loading ? 'info' : usageServiceAvailable ? 'good' : 'warn';
  const statusLabel = error
    ? t('monitoring.charts_status_error', { defaultValue: 'Chart load failed' })
    : loading
      ? t('monitoring.charts_status_loading', { defaultValue: 'Loading chart data' })
      : usageServiceAvailable
        ? t('monitoring.charts_status_ready', { defaultValue: 'Charts ready' })
        : t('monitoring.charts_status_unavailable', { defaultValue: 'Usage service unavailable' });

  const handleRangeChange = (value: string) => {
    setFilterState((current) => ({ ...current, range: value as UsageChartsRange }));
  };
  const handleDimensionChange = (value: string) => {
    const dimension = value as UsageChartsDimension;
    setFilterState((current) => clearFilterForDimension({ ...current, dimension }, dimension));
  };
  const handleFilterChange = (key: keyof Pick<UsageChartsFilterState, 'account' | 'apiKeyHash' | 'model'>) =>
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
                {`${t('monitoring.charts_granularity_label', { defaultValue: 'Granularity' })}: ${charts?.granularity ?? chartParams.granularity ?? '10m'}`}
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
            <span>{t('monitoring.charts_dimension_label', { defaultValue: 'Chart dimension' })}</span>
            <Select
              ariaLabel={t('monitoring.charts_dimension_label', { defaultValue: 'Chart dimension' })}
              value={filterState.dimension}
              options={dimensionOptions}
              onChange={handleDimensionChange}
            />
          </div>
          {showAccountFilter ? (
            <div className={styles.filterField}>
              <span>{t('monitoring.charts_account_label', { defaultValue: 'Account' })}</span>
              <Select
                ariaLabel={t('monitoring.charts_account_label', { defaultValue: 'Account' })}
                value={filterState.account}
                options={accountOptions}
                onChange={handleFilterChange('account')}
              />
            </div>
          ) : null}
          {showApiKeyFilter ? (
            <div className={styles.filterField}>
              <span>{t('monitoring.charts_api_key_label', { defaultValue: 'Caller key' })}</span>
              <Select
                ariaLabel={t('monitoring.charts_api_key_label', { defaultValue: 'Caller key' })}
                value={filterState.apiKeyHash}
                options={apiKeyOptions}
                onChange={handleFilterChange('apiKeyHash')}
              />
            </div>
          ) : null}
          {showModelFilter ? (
            <div className={styles.filterField}>
              <span>{t('monitoring.charts_model_label', { defaultValue: 'Model' })}</span>
              <Select
                ariaLabel={t('monitoring.charts_model_label', { defaultValue: 'Model' })}
                value={filterState.model}
                options={modelOptions}
                onChange={handleFilterChange('model')}
              />
            </div>
          ) : null}
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
              <strong>{formatChartMetricValue(globalTotals.inputTokens)}</strong>
            </Card>
            <Card className={styles.summaryCard}>
              <span>{t('monitoring.output_tokens')}</span>
              <strong>{formatChartMetricValue(globalTotals.outputTokens)}</strong>
            </Card>
            <Card className={styles.summaryCard}>
              <span>{t('monitoring.cached_tokens')}</span>
              <strong>{formatChartMetricValue(globalTotals.cachedTokens)}</strong>
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

          {!hasActiveChartData ? (
            <Card className={styles.dimensionEmpty}>
              <IconChartLine size={20} />
              <strong>
                {filterState.dimension === 'global'
                  ? t('monitoring.charts_empty_title', { defaultValue: 'No chart data yet' })
                  : t('monitoring.charts_dimension_empty_title', {
                      defaultValue: `No ${activeDimensionLabel} series`,
                    })}
              </strong>
              <span>
                {filterState.dimension === 'global'
                  ? t('monitoring.charts_empty_desc', {
                      defaultValue: 'Recent usage events will appear here after requests are recorded.',
                    })
                  : t('monitoring.charts_dimension_empty_desc', {
                      defaultValue: `Switch dimensions or adjust filters when ${activeDimensionLabel} series are unavailable.`,
                    })}
              </span>
            </Card>
          ) : (
            <section className={styles.chartGrid}>
              <UsageMetricChartCard
                activeDimensionLabel={activeDimensionLabel}
                chart={activeTokenChartDefinition}
                groupKey="tokens"
                onSelectTab={(family) => setActiveTokenChart(family as TokenChartFamily)}
                option={buildUsageChartOption(
                  activeTokenChartDefinition,
                  filterState.dimension,
                  globalBuckets,
                  activeSeries
                )}
                tabs={[tokenUsageChart, cumulativeTokenUsageChart]}
              />
              <UsageMetricChartCard
                activeDimensionLabel={activeDimensionLabel}
                chart={activeCostChartDefinition}
                groupKey="cost"
                onSelectTab={(family) => setActiveCostChart(family as CostChartFamily)}
                option={buildUsageChartOption(
                  activeCostChartDefinition,
                  filterState.dimension,
                  globalBuckets,
                  activeSeries
                )}
                tabs={[costUsageChart, cumulativeCostUsageChart]}
              />
              <UsageMetricChartCard
                activeDimensionLabel={activeDimensionLabel}
                chart={tpmUsageChart}
                groupKey="tpm"
                option={buildUsageChartOption(tpmUsageChart, filterState.dimension, globalBuckets, activeSeries)}
              />
            </section>
          )}
        </>
      )}
    </div>
  );
}

interface UsageMetricChartCardProps {
  activeDimensionLabel: string;
  chart: ChartDefinition;
  groupKey: string;
  onSelectTab?: (family: UsageChartMetricFamily) => void;
  option: EChartsCoreOption;
  tabs?: ChartDefinition[];
}

function UsageMetricChartCard({
  activeDimensionLabel,
  chart,
  groupKey,
  onSelectTab,
  option,
  tabs = [chart],
}: UsageMetricChartCardProps) {
  const hasTabs = tabs.length > 1;
  const panelId = `usage-chart-panel-${groupKey}`;
  const activeTabId = `usage-chart-tab-${groupKey}-${chart.family}`;

  return (
    <Card className={styles.chartCard}>
      <div className={styles.chartHeader}>
        <div className={styles.chartTitleGroup}>
          <h2>{chart.title}</h2>
          <span>{`${activeDimensionLabel}: ${chart.description}`}</span>
        </div>
        {hasTabs ? (
          <div
            aria-label={`${activeDimensionLabel} ${chart.title}`}
            className={styles.chartTabs}
            role="tablist"
          >
            {tabs.map((tab) => {
              const active = tab.family === chart.family;
              return (
                <button
                  key={tab.family}
                  aria-controls={panelId}
                  aria-selected={active}
                  className={`${styles.chartTab} ${active ? styles.chartTabActive : ''}`}
                  id={`usage-chart-tab-${groupKey}-${tab.family}`}
                  onClick={() => onSelectTab?.(tab.family)}
                  role="tab"
                  tabIndex={active ? 0 : -1}
                  type="button"
                >
                  {tab.title}
                </button>
              );
            })}
          </div>
        ) : null}
      </div>
      {hasTabs ? (
        <div aria-labelledby={activeTabId} className={styles.chartPanel} id={panelId} role="tabpanel">
          <EChartPanel
            key={chart.family}
            ariaLabel={`${activeDimensionLabel} ${chart.title}`}
            className={styles.chartCanvas}
            option={option}
          />
        </div>
      ) : (
        <div className={styles.chartPanel}>
          <EChartPanel
            key={chart.family}
            ariaLabel={`${activeDimensionLabel} ${chart.title}`}
            className={styles.chartCanvas}
            option={option}
          />
        </div>
      )}
    </Card>
  );
}

function buildUsageChartOption(
  chart: ChartDefinition,
  dimension: UsageChartsDimension,
  globalBuckets: UsageChartMetricBucket[],
  activeSeries: UsageChartSeries[]
): EChartsCoreOption {
  if (dimension === 'global') {
    return buildGlobalUsageChartOption({
      title: chart.title,
      family: chart.family,
      buckets: globalBuckets,
    });
  }

  return buildSeriesUsageChartOption({
    title: chart.title,
    family: chart.family,
    series: activeSeries,
  });
}

function resolveActiveDimensionSeries(
  dimension: UsageChartsDimension,
  series: Record<Exclude<UsageChartsDimension, 'global'>, UsageChartSeries[]>
): UsageChartSeries[] {
  if (dimension === 'global') return [];
  return series[dimension];
}

function clearFilterForDimension(
  state: UsageChartsFilterState,
  dimension: UsageChartsDimension
): UsageChartsFilterState {
  return {
    ...state,
    account: dimension === 'account' ? '' : state.account,
    apiKeyHash: dimension === 'apiKey' ? '' : state.apiKeyHash,
    model: dimension === 'model' ? '' : state.model,
  };
}
