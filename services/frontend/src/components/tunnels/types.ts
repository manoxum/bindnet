export interface TunnelConfig {
  id: string;
  name: string;
  apiUrl: string;
  endpoint: string;
  orgId: string;
  configured: boolean;
}
export interface TunnelSite {
  siteId: number;
  name: string;
  type: string;
  online: boolean;
  managed: boolean;
}
export interface TunnelTarget {
  targetId: number;
  siteId: number;
  ip: string;
  port: number;
  method: string;
  enabled: boolean;
}
export interface TunnelResource {
  resourceId: number;
  localDomain?: string;
  name: string;
  mode: string;
  fullDomain: string;
  proxyPort: number | null;
  enabled: boolean;
  sso: boolean;
  managed: boolean;
  targets: TunnelTarget[] | null;
}
export interface TunnelDomain {
  domainId: string;
  baseDomain: string;
  verified: boolean;
  type: string;
}
export interface ConnectorStatus {
  running: boolean;
  status: string;
  image: string;
  health: string;
}
