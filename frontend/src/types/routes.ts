export const ROUTES = {
  HOME: "/",
} as const;

/** Any path the router may receive. */
export type Route = (typeof ROUTES)[keyof typeof ROUTES] | string;
