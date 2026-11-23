export const PORTAL_HOSTNAME = "bindnet.local.com";
export const ADMIN_HOSTNAME = "admin.bindnet.local.com";

export type AppSurface = "portal" | "admin" | "unknown";

const developmentAdminHosts = new Set(["localhost", "127.0.0.1", "::1"]);

// Hostname efetivo: o Host visto pelo servidor (injetado pelo nginx) tem
// prioridade sobre o do browser, que pode ser um domínio público de túnel.
export function effectiveHostname(): string {
  const injected = document
    .querySelector('meta[name="bindnet-server-host"]')
    ?.getAttribute("content")
    ?.trim()
    .toLowerCase();
  if (injected && !injected.startsWith("__") && injected !== "_") {
    const known = [PORTAL_HOSTNAME, ADMIN_HOSTNAME];
    if (known.includes(injected)) return injected;
  }
  return window.location.hostname;
}

export function appSurfaceForHostname(hostname: string): AppSurface {
  const normalizedHostname = hostname.trim().toLowerCase();

  if (normalizedHostname === PORTAL_HOSTNAME) return "portal";
  if (normalizedHostname === ADMIN_HOSTNAME) return "admin";
  if (developmentAdminHosts.has(normalizedHostname)) return "admin";
  return "unknown";
}
