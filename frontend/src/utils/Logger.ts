import { Config } from "@/base/config";

export function debugLog(...args: unknown[]) {
  if (Config.logErrors) console.log(...args);
}

export function debugWarn(...args: unknown[]) {
  if (Config.logErrors) console.warn(...args);
}

export function debugError(...args: unknown[]) {
  console.error(...args);
}
