const parsedTimeout = Number(import.meta.env.VITE_API_TIMEOUT_MS ?? 10000)

export const env = {
  apiBaseUrl: import.meta.env.VITE_API_BASE_URL || '/api',
  apiTimeoutMs: Number.isFinite(parsedTimeout) ? parsedTimeout : 10000,
}
