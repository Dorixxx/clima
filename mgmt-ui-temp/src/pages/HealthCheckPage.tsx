import { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { Button } from '@/components/ui/Button';
import { Select } from '@/components/ui/Select';
import { useHeaderRefresh } from '@/hooks/useHeaderRefresh';
import { useHealthCheckStore, useNotificationStore } from '@/stores';
import type { HealthCheckEntry } from '@/types';
import styles from './HealthCheckPage.module.scss';

const statusClassName = (status: string) => {
  switch (status) {
    case 'healthy':
      return styles.statusHealthy;
    case 'unauthorized':
      return styles.statusUnauthorized;
    case 'zero_quota':
      return styles.statusQuota;
    case 'disabled':
      return styles.statusDisabled;
    case 'error':
      return styles.statusError;
    default:
      return styles.statusSkipped;
  }
};

export function HealthCheckPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { showNotification } = useNotificationStore();
  const { snapshot, loading, saving, fetchSnapshot, runNow, stopRun } = useHealthCheckStore();
  const [selectedRunId, setSelectedRunId] = useState('');

  useEffect(() => {
    void fetchSnapshot();
  }, [fetchSnapshot]);

  useHeaderRefresh(() => fetchSnapshot().then(() => undefined));

  const summary = snapshot?.summary;
  const currentRun = summary?.current_run;
  const history = snapshot?.history ?? [];
  const selectedRun = useMemo(() => {
    if (!history.length) return null;
    return history.find((item) => item.id === selectedRunId) ?? history[0];
  }, [history, selectedRunId]);
  const entries: HealthCheckEntry[] = selectedRun?.entries ?? [];
  const progressEntries = currentRun?.latest_entries ?? [];

  const doughnutStyle = useMemo(() => {
    const total = Math.max(
      1,
      (summary?.last_run_healthy ?? 0) +
        (summary?.last_run_unauthorized ?? 0) +
        (summary?.last_run_zero_quota ?? 0) +
        (summary?.last_run_errors ?? 0)
    );
    const healthy = ((summary?.last_run_healthy ?? 0) / total) * 100;
    const unauthorized = ((summary?.last_run_unauthorized ?? 0) / total) * 100;
    const quota = ((summary?.last_run_zero_quota ?? 0) / total) * 100;
    const errors = ((summary?.last_run_errors ?? 0) / total) * 100;
    return {
      background: `conic-gradient(
        var(--success-color) 0 ${healthy}%,
        var(--error-color) ${healthy}% ${healthy + unauthorized}%,
        var(--warning-color) ${healthy + unauthorized}% ${healthy + unauthorized + quota}%,
        var(--danger-color) ${healthy + unauthorized + quota}% ${healthy + unauthorized + quota + errors}%,
        var(--bg-tertiary) ${healthy + unauthorized + quota + errors}% 100%
      )`
    };
  }, [summary]);

  const handleRunNow = async () => {
    try {
      const run = await runNow();
      setSelectedRunId(run.id);
      showNotification(t('health_check.run_started'), 'success');
    } catch (error) {
      showNotification(error instanceof Error ? error.message : t('health_check.run_error'), 'error');
    }
  };

  const handleStop = async () => {
    try {
      await stopRun();
      showNotification(t('health_check.stop_requested'), 'success');
    } catch (error) {
      showNotification(error instanceof Error ? error.message : t('health_check.stop_failed'), 'error');
    }
  };

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <div>
          <h1 className={styles.pageTitle}>{t('health_check.title')}</h1>
          <p className={styles.pageSubtitle}>{t('health_check.subtitle')}</p>
        </div>
        <div className={styles.actionGroup}>
          <Button variant="secondary" onClick={() => navigate('/config')} disabled={saving}>
            {t('health_check.open_config')}
          </Button>
          {summary?.running ? (
            <Button variant="danger" onClick={handleStop} loading={saving} disabled={loading || currentRun?.stopping}>
              {currentRun?.stopping ? t('health_check.stopping') : t('health_check.stop')}
            </Button>
          ) : (
            <Button onClick={handleRunNow} loading={saving} disabled={loading}>
              {t('health_check.run_now')}
            </Button>
          )}
        </div>
      </div>

      {currentRun && (
        <section className={styles.progressCard}>
          <div className={styles.cardHeader}>
            <h2>{currentRun.stopping ? t('health_check.stopping') : t('health_check.running')}</h2>
            <span className={styles.lastRunText}>
              {currentRun.processed}/{currentRun.total} · {currentRun.progress_pct.toFixed(0)}%
            </span>
          </div>
          <div className={styles.progressBar}>
            <div className={styles.progressValue} style={{ width: `${currentRun.progress_pct}%` }} />
          </div>
          <div className={styles.progressMeta}>
            <span>{currentRun.current_name || '-'}</span>
            <span>{t('health_check.remaining', { count: currentRun.estimated_left })}</span>
          </div>
          {!!progressEntries.length && (
            <div className={styles.progressList}>
              {progressEntries.slice().reverse().map((entry) => (
                <div key={`${entry.auth_id}-${entry.checked_at}`} className={styles.progressItem}>
                  <span>{entry.name}</span>
                  <span className={`${styles.statusBadge} ${statusClassName(entry.status)}`}>{entry.status}</span>
                </div>
              ))}
            </div>
          )}
        </section>
      )}

      <div className={styles.heroGrid}>
        <section className={styles.card}>
          <div className={styles.cardHeader}>
            <h2>{t('health_check.overview')}</h2>
            <span className={styles.lastRunText}>
              {summary?.last_run_at ? new Date(summary.last_run_at).toLocaleString() : t('health_check.never_run')}
            </span>
          </div>
          <div className={styles.metrics}>
            <div className={styles.doughnutWrap}>
              <div className={styles.doughnut} style={doughnutStyle}>
                <div className={styles.doughnutInner}>
                  <strong>{summary?.last_run_total ?? 0}</strong>
                  <span>{t('health_check.accounts')}</span>
                </div>
              </div>
            </div>
            <div className={styles.metricList}>
              <div><span>{t('health_check.healthy')}</span><strong>{summary?.last_run_healthy ?? 0}</strong></div>
              <div><span>{t('health_check.unauthorized')}</span><strong>{summary?.last_run_unauthorized ?? 0}</strong></div>
              <div><span>{t('health_check.zero_quota')}</span><strong>{summary?.last_run_zero_quota ?? 0}</strong></div>
              <div><span>{t('health_check.errors')}</span><strong>{summary?.last_run_errors ?? 0}</strong></div>
            </div>
          </div>
        </section>

        <section className={styles.card}>
          <div className={styles.cardHeader}>
            <h2>{t('health_check.current_config')}</h2>
            <span className={styles.lastRunText}>{t('health_check.config_managed_in_panel')}</span>
          </div>
          <div className={styles.configList}>
            <div><span>{t('health_check.status')}</span><strong>{summary?.enabled ? t('health_check.enabled') : t('health_check.disabled')}</strong></div>
            <div><span>{t('health_check.interval')}</span><strong>{summary?.interval_seconds ?? 1800}s</strong></div>
            <div><span>{t('health_check.parallelism')}</span><strong>{summary?.parallelism ?? 4}</strong></div>
            <div><span>{t('health_check.unauthorized_action')}</span><strong>{summary?.unauthorized_action ?? 'disable'}</strong></div>
            <div><span>{t('health_check.zero_quota_action')}</span><strong>{summary?.zero_quota_action ?? 'disable'}</strong></div>
            <div><span>{t('health_check.provider_policies')}</span><strong>{Object.keys(summary?.provider_policies ?? {}).length}</strong></div>
            <div><span>{t('health_check.bark')}</span><strong>{summary?.notifications?.bark?.enabled ? 'On' : 'Off'}</strong></div>
            <div><span>{t('health_check.email_notification')}</span><strong>{summary?.notifications?.email?.enabled ? 'On' : 'Off'}</strong></div>
          </div>
        </section>
      </div>

      <section className={styles.card}>
        <div className={styles.cardHeader}>
          <h2>{t('health_check.latest_results')}</h2>
          <div className={styles.historyControl}>
            <span className={styles.lastRunText}>
              {selectedRun ? `${selectedRun.duration_ms} ms` : t('health_check.no_results')}
            </span>
            <Select
              value={selectedRun?.id ?? ''}
              options={history.map((run) => ({
                value: run.id,
                label: `${new Date(run.finished_at).toLocaleString()} · ${run.triggered_by}${run.stopped ? ` · ${t('health_check.stopped')}` : ''}`
              }))}
              onChange={setSelectedRunId}
              disabled={!history.length || Boolean(summary?.running)}
            />
          </div>
        </div>
        <div className={styles.tableWrap}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>{t('health_check.account')}</th>
                <th>{t('health_check.provider')}</th>
                <th>{t('health_check.status')}</th>
                <th>{t('health_check.action')}</th>
                <th>{t('health_check.message')}</th>
              </tr>
            </thead>
            <tbody>
              {entries.length ? (
                entries.map((entry) => (
                  <tr key={`${entry.auth_id}-${entry.checked_at}`}>
                    <td>{entry.name}</td>
                    <td>{entry.provider}</td>
                    <td>
                      <span className={`${styles.statusBadge} ${statusClassName(entry.status)}`}>{entry.status}</span>
                    </td>
                    <td>{entry.action}</td>
                    <td>{entry.message || '-'}</td>
                  </tr>
                ))
              ) : (
                <tr>
                  <td colSpan={5} className={styles.emptyCell}>
                    {loading ? t('common.loading') : t('health_check.no_results')}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}
