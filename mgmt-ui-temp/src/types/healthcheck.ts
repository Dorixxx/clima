export type HealthCheckStatus =
  | 'healthy'
  | 'unauthorized'
  | 'zero_quota'
  | 'disabled'
  | 'error'
  | 'skipped';

export type HealthCheckAction = 'none' | 'ignored' | 'disabled' | 'deleted';

export interface HealthCheckEntry {
  auth_id: string;
  name: string;
  provider: string;
  status: HealthCheckStatus;
  action: HealthCheckAction;
  message?: string;
  checked_at: string;
  disabled: boolean;
}

export interface HealthCheckRun {
  id: string;
  started_at: string;
  finished_at: string;
  duration_ms: number;
  triggered_by: string;
  stopped?: boolean;
  total: number;
  healthy: number;
  unauthorized: number;
  zero_quota: number;
  disabled: number;
  errors: number;
  entries: HealthCheckEntry[];
}

export interface HealthCheckSummary {
  enabled: boolean;
  interval_seconds: number;
  parallelism: number;
  unauthorized_action: 'delete' | 'disable' | 'ignore';
  zero_quota_action: 'disable' | 'ignore';
  provider_policies?: Record<
    string,
    {
      unauthorizedAction?: 'delete' | 'disable' | 'ignore';
      zeroQuotaAction?: 'disable' | 'ignore';
    }
  >;
  notifications?: {
    bark?: {
      enabled?: boolean;
      serverUrl?: string;
      deviceKey?: string;
      group?: string;
    };
    email?: {
      enabled?: boolean;
      smtpHost?: string;
      smtpPort?: number;
      username?: string;
      password?: string;
      from?: string;
      to?: string[];
      subjectPrefix?: string;
    };
  };
  running: boolean;
  last_run_at?: string;
  last_run_status?: string;
  last_run_triggered_by?: string;
  last_run_duration_ms?: number;
  last_run_total: number;
  last_run_healthy: number;
  last_run_unauthorized: number;
  last_run_zero_quota: number;
  last_run_disabled: number;
  last_run_errors: number;
  current_run?: {
    id: string;
    started_at: string;
    triggered_by: string;
    stopping?: boolean;
    total: number;
    processed: number;
    healthy: number;
    unauthorized: number;
    zero_quota: number;
    disabled: number;
    errors: number;
    progress_pct: number;
    current_name?: string;
    estimated_left: number;
    latest_entries?: HealthCheckEntry[];
  };
}

export interface HealthCheckSnapshot {
  summary: HealthCheckSummary;
  history: HealthCheckRun[];
}
