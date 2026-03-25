/**
 * 配置相关类型定义
 * 与基线 /config 返回结构保持一致（内部使用驼峰形式）
 */

import type { GeminiKeyConfig, ProviderKeyConfig, OpenAIProviderConfig } from './provider';
import type { AmpcodeConfig } from './ampcode';

export interface QuotaExceededConfig {
  switchProject?: boolean;
  switchPreviewModel?: boolean;
}

export interface HealthCheckConfig {
  enabled?: boolean;
  intervalSeconds?: number;
  parallelism?: number;
  unauthorizedAction?: 'delete' | 'disable' | 'ignore';
  zeroQuotaAction?: 'disable' | 'ignore';
  providerPolicies?: Record<
    string,
    {
      unauthorizedAction?: 'delete' | 'disable' | 'ignore';
      zeroQuotaAction?: 'disable' | 'ignore';
    }
  >;
  notifications?: {
    bark?: {
      enabled?: boolean;
      url?: string;
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
}

export interface Config {
  debug?: boolean;
  proxyUrl?: string;
  requestRetry?: number;
  quotaExceeded?: QuotaExceededConfig;
  usageStatisticsEnabled?: boolean;
  requestLog?: boolean;
  loggingToFile?: boolean;
  logsMaxTotalSizeMb?: number;
  wsAuth?: boolean;
  forceModelPrefix?: boolean;
  routingStrategy?: string;
  healthCheck?: HealthCheckConfig;
  apiKeys?: string[];
  ampcode?: AmpcodeConfig;
  geminiApiKeys?: GeminiKeyConfig[];
  codexApiKeys?: ProviderKeyConfig[];
  claudeApiKeys?: ProviderKeyConfig[];
  vertexApiKeys?: ProviderKeyConfig[];
  openaiCompatibility?: OpenAIProviderConfig[];
  oauthExcludedModels?: Record<string, string[]>;
  raw?: Record<string, unknown>;
}

export type RawConfigSection =
  | 'debug'
  | 'proxy-url'
  | 'request-retry'
  | 'quota-exceeded'
  | 'usage-statistics-enabled'
  | 'request-log'
  | 'logging-to-file'
  | 'logs-max-total-size-mb'
  | 'ws-auth'
  | 'force-model-prefix'
  | 'health-check'
  | 'routing/strategy'
  | 'api-keys'
  | 'ampcode'
  | 'gemini-api-key'
  | 'codex-api-key'
  | 'claude-api-key'
  | 'vertex-api-key'
  | 'openai-compatibility'
  | 'oauth-excluded-models';

export interface ConfigCache {
  data: Config;
  timestamp: number;
}
