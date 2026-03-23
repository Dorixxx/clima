import { apiClient } from './client';
import type { HealthCheckSnapshot } from '@/types';

type ProviderPolicy = {
  unauthorizedAction?: 'delete' | 'disable' | 'ignore';
  zeroQuotaAction?: 'disable' | 'ignore';
};

type HealthCheckNotifications = {
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

const normalizeProviderPolicies = (value: unknown): Record<string, ProviderPolicy> | undefined => {
  if (!value || typeof value !== 'object') {
    return undefined;
  }

  const entries = Object.entries(value as Record<string, unknown>).map(([provider, raw]) => {
    if (!raw || typeof raw !== 'object') {
      return [provider, {} satisfies ProviderPolicy] as const;
    }
    const record = raw as Record<string, unknown>;
    const unauthorizedRaw = record['unauthorized-action'] ?? record.unauthorizedAction;
    const zeroQuotaRaw = record['zero-quota-action'] ?? record.zeroQuotaAction;
    return [
      provider,
      {
        unauthorizedAction: typeof unauthorizedRaw === 'string' ? (unauthorizedRaw as ProviderPolicy['unauthorizedAction']) : undefined,
        zeroQuotaAction: typeof zeroQuotaRaw === 'string' ? (zeroQuotaRaw as ProviderPolicy['zeroQuotaAction']) : undefined
      }
    ] as const;
  });

  return Object.fromEntries(entries);
};

const normalizeSnapshot = (snapshot: HealthCheckSnapshot): HealthCheckSnapshot => ({
  ...snapshot,
  summary: {
    ...snapshot.summary,
    provider_policies: normalizeProviderPolicies(snapshot.summary.provider_policies),
    notifications: snapshot.summary.notifications
      ? {
          bark: snapshot.summary.notifications.bark
            ? {
                enabled: snapshot.summary.notifications.bark.enabled,
                serverUrl:
                  (snapshot.summary.notifications.bark as Record<string, unknown>)['server-url'] as string | undefined ??
                  snapshot.summary.notifications.bark.serverUrl,
                deviceKey:
                  (snapshot.summary.notifications.bark as Record<string, unknown>)['device-key'] as string | undefined ??
                  snapshot.summary.notifications.bark.deviceKey,
                group: snapshot.summary.notifications.bark.group
              }
            : undefined,
          email: snapshot.summary.notifications.email
            ? {
                enabled: snapshot.summary.notifications.email.enabled,
                smtpHost:
                  (snapshot.summary.notifications.email as Record<string, unknown>)['smtp-host'] as string | undefined ??
                  snapshot.summary.notifications.email.smtpHost,
                smtpPort:
                  (snapshot.summary.notifications.email as Record<string, unknown>)['smtp-port'] as number | undefined ??
                  snapshot.summary.notifications.email.smtpPort,
                username: snapshot.summary.notifications.email.username,
                password: snapshot.summary.notifications.email.password,
                from: snapshot.summary.notifications.email.from,
                to: snapshot.summary.notifications.email.to,
                subjectPrefix:
                  (snapshot.summary.notifications.email as Record<string, unknown>)['subject-prefix'] as string | undefined ??
                  snapshot.summary.notifications.email.subjectPrefix
              }
            : undefined
        }
      : undefined
  }
});

export const healthCheckApi = {
  getSnapshot: async () => normalizeSnapshot(await apiClient.get<HealthCheckSnapshot>('/health-check')),
  runNow: () =>
    apiClient.post<{ status: string; run: { id: string; started_at: string; triggered_by: string; total: number } }>(
      '/health-check/run'
    ),
  stopRun: () =>
    apiClient.post<{ status: string; run: { id: string; started_at: string; triggered_by: string; total: number; stopping?: boolean } }>(
      '/health-check/stop'
    ),
  updateEnabled: (enabled: boolean) => apiClient.put('/health-check/enabled', { value: enabled }),
  updateInterval: (seconds: number) => apiClient.put('/health-check/interval', { value: seconds }),
  updateParallelism: (value: number) => apiClient.put('/health-check/parallelism', { value }),
  updateUnauthorizedAction: (value: 'delete' | 'disable' | 'ignore') =>
    apiClient.put('/health-check/unauthorized-action', { value }),
  updateZeroQuotaAction: (value: 'disable' | 'ignore') =>
    apiClient.put('/health-check/zero-quota-action', { value }),
  updateProviderPolicies: (value: Record<string, ProviderPolicy>) =>
    apiClient.put('/health-check/provider-policies', {
      value: Object.fromEntries(
        Object.entries(value).map(([provider, policy]) => [
          provider,
          {
            'unauthorized-action': policy.unauthorizedAction,
            'zero-quota-action': policy.zeroQuotaAction
          }
        ])
      )
    }),
  updateNotifications: (value: HealthCheckNotifications) =>
    apiClient.put('/health-check/notifications', {
      value: {
        bark: value.bark
          ? {
              enabled: value.bark.enabled,
              'server-url': value.bark.serverUrl,
              'device-key': value.bark.deviceKey,
              group: value.bark.group
            }
          : undefined,
        email: value.email
          ? {
              enabled: value.email.enabled,
              'smtp-host': value.email.smtpHost,
              'smtp-port': value.email.smtpPort,
              username: value.email.username,
              password: value.email.password,
              from: value.email.from,
              to: value.email.to,
              'subject-prefix': value.email.subjectPrefix
            }
          : undefined
      }
    })
};
