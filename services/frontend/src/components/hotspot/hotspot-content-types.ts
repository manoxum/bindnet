// Tipos do bloqueio/permissao de conteudo (planos + regras + categorias).
// Espelham os shapes do backend (services/backend/internal/hotspot/
// hotspot_content*.go).

export type ContentAction = "allow" | "block";
export type ContentRuleKind = "domain" | "ip" | "category";

export interface ContentPlan {
  id: string;
  name: string;
  description: string;
  defaultAction: ContentAction;
}

export interface ContentRule {
  id: string;
  planId: string;
  kind: ContentRuleKind;
  value: string;
  action: ContentAction;
}

export interface ContentPlanDetail extends ContentPlan {
  rules: ContentRule[];
}

export interface ContentCategory {
  slug: string;
  name: string;
  sourceUrls: string;
  enabled: boolean;
  lastSyncedAt: string | null;
  domainCount: number;
}

export interface ContentPlanRequest {
  name: string;
  description: string;
  defaultAction: ContentAction;
}

export interface ContentRuleRequest {
  kind: ContentRuleKind;
  value: string;
  action: ContentAction;
}

export const CONTENT_KIND_LABELS: Record<ContentRuleKind, string> = {
  domain: "Domínio",
  ip: "IP/CIDR",
  category: "Categoria",
};
