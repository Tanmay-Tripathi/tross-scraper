/** Runtime configuration, read from Vite env vars at build time. */
export const Config = {
  appName: import.meta.env.VITE_APP_NAME || "Tross Scraper",
  /** Base URL of the Go API. Empty means same-origin. */
  apiHost: import.meta.env.VITE_API_HOST || "",
  logErrors: import.meta.env.VITE_LOG_ERRORS === "true",
} as const;

/** Builds an absolute URL for an API path such as `/public/v1/health`. */
export function apiUrl(path: string): string {
  const base = Config.apiHost.replace(/\/$/, "");
  return `${base}${path.startsWith("/") ? path : `/${path}`}`;
}
