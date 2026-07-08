import type { HotspotTraffic } from "@/components/hotspot/hotspot-limits-types";

// Resposta de GET /api/hotspot/portal/me - reusa os mesmos nomes de
// campo de HotspotTraffic (o backend monta a resposta assim de
// proposito) para que HotspotQuotaProgress renderize sem alteração.
export interface PortalMeResponse extends HotspotTraffic {
  mac: string;
  alias?: string;
  profileName?: string;
  blocked: boolean;
  blockedByCredit: boolean;
  creditEnabled: boolean;
  balanceBytes: number;
  plafondBytes: number | null;
}

export interface PortalCreditHistoryEntry {
  entryType: string;
  amountBytes: number;
  balanceAfterBytes: number;
  createdAt: string;
}
