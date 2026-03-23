import { DEFAULT_API_PORT, MANAGEMENT_API_PREFIX } from './constants';

const FRONTEND_DEV_PORTS = new Set(['4173', '5173', '5174', '4174']);

export const normalizeApiBase = (input: string): string => {
  let base = (input || '').trim();
  if (!base) return '';
  base = base.replace(/\/?v0\/management\/?$/i, '');
  base = base.replace(/\/+$/i, '');
  if (!/^https?:\/\//i.test(base)) {
    base = `http://${base}`;
  }
  try {
    const url = new URL(base);
    const hostname = url.hostname.toLowerCase();
    if ((hostname === '127.0.0.1' || hostname === 'localhost') && FRONTEND_DEV_PORTS.has(url.port)) {
      url.port = String(DEFAULT_API_PORT);
      base = url.toString().replace(/\/+$/i, '');
    }
  } catch {
    // Keep the original normalized value when URL parsing fails.
  }
  return base;
};

export const computeApiUrl = (base: string): string => {
  const normalized = normalizeApiBase(base);
  if (!normalized) return '';
  return `${normalized}${MANAGEMENT_API_PREFIX}`;
};

export const detectApiBaseFromLocation = (): string => {
  try {
    const { protocol, hostname, port } = window.location;
    const resolvedPort =
      port && FRONTEND_DEV_PORTS.has(port) ? String(DEFAULT_API_PORT) : port;
    const normalizedPort = resolvedPort ? `:${resolvedPort}` : '';
    return normalizeApiBase(`${protocol}//${hostname}${normalizedPort}`);
  } catch (error) {
    console.warn('Failed to detect api base from location, fallback to default', error);
    return normalizeApiBase(`http://localhost:${DEFAULT_API_PORT}`);
  }
};

export const isLocalhost = (hostname: string): boolean => {
  const value = (hostname || '').toLowerCase();
  return value === 'localhost' || value === '127.0.0.1' || value === '[::1]';
};
