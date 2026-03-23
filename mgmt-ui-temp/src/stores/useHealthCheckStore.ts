import { create } from 'zustand';
import { healthCheckApi } from '@/services/api/healthcheck';
import { useConfigStore } from './useConfigStore';
import type { HealthCheckSnapshot } from '@/types';

interface HealthCheckState {
  snapshot: HealthCheckSnapshot | null;
  loading: boolean;
  saving: boolean;
  error: string | null;
  pollTimer: number | null;
  fetchSnapshot: () => Promise<HealthCheckSnapshot>;
  runNow: () => Promise<{ id: string; started_at: string; triggered_by: string; total: number }>;
  stopRun: () => Promise<{ id: string; started_at: string; triggered_by: string; total: number; stopping?: boolean }>;
  startPolling: () => void;
  stopPolling: () => void;
  setEnabled: (enabled: boolean) => Promise<void>;
  setIntervalSeconds: (seconds: number) => Promise<void>;
  setParallelism: (value: number) => Promise<void>;
  setUnauthorizedAction: (value: 'delete' | 'disable' | 'ignore') => Promise<void>;
  setZeroQuotaAction: (value: 'disable' | 'ignore') => Promise<void>;
  setProviderPolicies: (
    value: Record<string, { unauthorizedAction?: 'delete' | 'disable' | 'ignore'; zeroQuotaAction?: 'disable' | 'ignore' }>
  ) => Promise<void>;
  setNotifications: (value: {
    bark?: { enabled?: boolean; serverUrl?: string; deviceKey?: string; group?: string };
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
  }) => Promise<void>;
}

export const useHealthCheckStore = create<HealthCheckState>((set, get) => ({
  snapshot: null,
  loading: false,
  saving: false,
  error: null,
  pollTimer: null,
  fetchSnapshot: async () => {
    set({ loading: true, error: null });
    try {
      const snapshot = await healthCheckApi.getSnapshot();
      set({ snapshot, loading: false });
      if (snapshot.summary.running) {
        get().startPolling();
      } else {
        get().stopPolling();
      }
      useConfigStore.getState().updateConfigValue('health-check', {
        enabled: snapshot.summary.enabled,
        intervalSeconds: snapshot.summary.interval_seconds,
        parallelism: snapshot.summary.parallelism,
        unauthorizedAction: snapshot.summary.unauthorized_action,
        zeroQuotaAction: snapshot.summary.zero_quota_action,
        providerPolicies: snapshot.summary.provider_policies,
        notifications: snapshot.summary.notifications
      });
      return snapshot;
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to fetch health check';
      set({ loading: false, error: message });
      throw error;
    }
  },
  runNow: async () => {
    set({ saving: true, error: null });
    try {
      const run = await healthCheckApi.runNow();
      set({ saving: false });
      get().startPolling();
      await get().fetchSnapshot();
      return run.run;
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to run health check';
      set({ saving: false, error: message });
      throw error;
    }
  },
  stopRun: async () => {
    set({ saving: true, error: null });
    try {
      const run = await healthCheckApi.stopRun();
      set({ saving: false });
      get().startPolling();
      await get().fetchSnapshot();
      return run.run;
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to stop health check';
      set({ saving: false, error: message });
      throw error;
    }
  },
  startPolling: () => {
    if (get().pollTimer !== null || typeof window === 'undefined') return;
    const timer = window.setInterval(() => {
      void get().fetchSnapshot().catch(() => undefined);
    }, 1000);
    set({ pollTimer: timer });
  },
  stopPolling: () => {
    const timer = get().pollTimer;
    if (timer !== null && typeof window !== 'undefined') {
      window.clearInterval(timer);
    }
    set({ pollTimer: null });
  },
  setEnabled: async (enabled) => {
    set({ saving: true, error: null });
    try {
      await healthCheckApi.updateEnabled(enabled);
      set({ saving: false });
      await get().fetchSnapshot();
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to update health check';
      set({ saving: false, error: message });
      throw error;
    }
  },
  setIntervalSeconds: async (seconds) => {
    set({ saving: true, error: null });
    try {
      await healthCheckApi.updateInterval(seconds);
      set({ saving: false });
      await get().fetchSnapshot();
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to update interval';
      set({ saving: false, error: message });
      throw error;
    }
  },
  setParallelism: async (value) => {
    set({ saving: true, error: null });
    try {
      await healthCheckApi.updateParallelism(value);
      set({ saving: false });
      await get().fetchSnapshot();
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to update parallelism';
      set({ saving: false, error: message });
      throw error;
    }
  },
  setUnauthorizedAction: async (value) => {
    set({ saving: true, error: null });
    try {
      await healthCheckApi.updateUnauthorizedAction(value);
      set({ saving: false });
      await get().fetchSnapshot();
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to update action';
      set({ saving: false, error: message });
      throw error;
    }
  },
  setZeroQuotaAction: async (value) => {
    set({ saving: true, error: null });
    try {
      await healthCheckApi.updateZeroQuotaAction(value);
      set({ saving: false });
      await get().fetchSnapshot();
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to update action';
      set({ saving: false, error: message });
      throw error;
    }
  },
  setProviderPolicies: async (value) => {
    set({ saving: true, error: null });
    try {
      await healthCheckApi.updateProviderPolicies(value);
      set({ saving: false });
      await get().fetchSnapshot();
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to update provider policies';
      set({ saving: false, error: message });
      throw error;
    }
  },
  setNotifications: async (value) => {
    set({ saving: true, error: null });
    try {
      await healthCheckApi.updateNotifications(value);
      set({ saving: false });
      await get().fetchSnapshot();
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to update notifications';
      set({ saving: false, error: message });
      throw error;
    }
  }
}));
