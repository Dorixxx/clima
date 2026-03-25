export type PayloadParamValueType = 'string' | 'number' | 'boolean' | 'json';
export type PayloadParamValidationErrorCode =
  | 'payload_invalid_number'
  | 'payload_invalid_boolean'
  | 'payload_invalid_json';

export type VisualConfigFieldPath =
  | 'port'
  | 'logsMaxTotalSizeMb'
  | 'requestRetry'
  | 'maxRetryCredentials'
  | 'maxRetryInterval'
  | 'healthCheck.intervalSeconds'
  | 'healthCheck.parallelism'
  | 'healthCheck.email.smtpPort'
  | 'streaming.keepaliveSeconds'
  | 'streaming.bootstrapRetries'
  | 'streaming.nonstreamKeepaliveInterval';

export type VisualConfigValidationErrorCode = 'port_range' | 'non_negative_integer';

export type VisualConfigValidationErrors = Partial<
  Record<VisualConfigFieldPath, VisualConfigValidationErrorCode>
>;

export type PayloadParamEntry = {
  id: string;
  path: string;
  valueType: PayloadParamValueType;
  value: string;
};

export type PayloadModelEntry = {
  id: string;
  name: string;
  protocol?: string;
};

export type PayloadRule = {
  id: string;
  models: PayloadModelEntry[];
  params: PayloadParamEntry[];
};

export type PayloadFilterRule = {
  id: string;
  models: PayloadModelEntry[];
  params: string[];
};

export interface StreamingConfig {
  keepaliveSeconds: string;
  bootstrapRetries: string;
  nonstreamKeepaliveInterval: string;
}

export interface VisualHealthCheckProviderPolicy {
  id: string;
  provider: string;
  unauthorizedAction: 'default' | 'delete' | 'disable' | 'ignore';
  zeroQuotaAction: 'default' | 'disable' | 'ignore';
}

export interface VisualHealthCheckNotifications {
  barkEnabled: boolean;
  barkUrl: string;
  barkGroup: string;
  emailEnabled: boolean;
  emailSmtpHost: string;
  emailSmtpPort: string;
  emailUsername: string;
  emailPassword: string;
  emailFrom: string;
  emailTo: string[];
  emailSubjectPrefix: string;
}

export type VisualConfigValues = {
  host: string;
  port: string;
  tlsEnable: boolean;
  tlsCert: string;
  tlsKey: string;
  rmAllowRemote: boolean;
  rmSecretKey: string;
  rmDisableControlPanel: boolean;
  rmPanelRepo: string;
  authDir: string;
  apiKeysText: string;
  debug: boolean;
  commercialMode: boolean;
  loggingToFile: boolean;
  logsMaxTotalSizeMb: string;
  usageStatisticsEnabled: boolean;
  proxyUrl: string;
  forceModelPrefix: boolean;
  requestRetry: string;
  maxRetryCredentials: string;
  maxRetryInterval: string;
  quotaSwitchProject: boolean;
  quotaSwitchPreviewModel: boolean;
  routingStrategy: 'round-robin' | 'fill-first';
  wsAuth: boolean;
  payloadDefaultRules: PayloadRule[];
  payloadDefaultRawRules: PayloadRule[];
  payloadOverrideRules: PayloadRule[];
  payloadOverrideRawRules: PayloadRule[];
  payloadFilterRules: PayloadFilterRule[];
  healthCheck: {
    enabled: boolean;
    intervalSeconds: string;
    parallelism: string;
    unauthorizedAction: 'delete' | 'disable' | 'ignore';
    zeroQuotaAction: 'disable' | 'ignore';
    providerPolicies: VisualHealthCheckProviderPolicy[];
    notifications: VisualHealthCheckNotifications;
  };
  streaming: StreamingConfig;
};

export const makeClientId = () => {
  if (typeof globalThis.crypto?.randomUUID === 'function') return globalThis.crypto.randomUUID();
  return `${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 10)}`;
};

export const DEFAULT_VISUAL_VALUES: VisualConfigValues = {
  host: '',
  port: '',
  tlsEnable: false,
  tlsCert: '',
  tlsKey: '',
  rmAllowRemote: false,
  rmSecretKey: '',
  rmDisableControlPanel: false,
  rmPanelRepo: '',
  authDir: '',
  apiKeysText: '',
  debug: false,
  commercialMode: false,
  loggingToFile: false,
  logsMaxTotalSizeMb: '',
  usageStatisticsEnabled: false,
  proxyUrl: '',
  forceModelPrefix: false,
  requestRetry: '',
  maxRetryCredentials: '',
  maxRetryInterval: '',
  quotaSwitchProject: true,
  quotaSwitchPreviewModel: true,
  routingStrategy: 'round-robin',
  wsAuth: false,
  payloadDefaultRules: [],
  payloadDefaultRawRules: [],
  payloadOverrideRules: [],
  payloadOverrideRawRules: [],
  payloadFilterRules: [],
  healthCheck: {
    enabled: false,
    intervalSeconds: '',
    parallelism: '',
    unauthorizedAction: 'disable',
    zeroQuotaAction: 'disable',
    providerPolicies: [],
    notifications: {
      barkEnabled: false,
      barkUrl: '',
      barkGroup: '',
      emailEnabled: false,
      emailSmtpHost: '',
      emailSmtpPort: '',
      emailUsername: '',
      emailPassword: '',
      emailFrom: '',
      emailTo: [],
      emailSubjectPrefix: '',
    },
  },
  streaming: {
    keepaliveSeconds: '',
    bootstrapRetries: '',
    nonstreamKeepaliveInterval: '',
  },
};
