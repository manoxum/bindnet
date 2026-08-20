export const PORTAL_HOSTNAME = "bindnet.local.com";
export const ADMIN_HOSTNAME = "admin.bindnet.local.com";

export type AppSurface = "portal" | "admin" | "unknown";

const developmentAdminHosts = new Set(["localhost", "127.0.0.1", "::1"]);

export function appSurfaceForHostname(hostname: string): AppSurface {
  const normalizedHostname = hostname.trim().toLowerCase();

  if (normalizedHostname === PORTAL_HOSTNAME) return "portal";
  if (normalizedHostname === ADMIN_HOSTNAME) return "admin";
  if (developmentAdminHosts.has(normalizedHostname)) return "admin";
  return "unknown";
}
