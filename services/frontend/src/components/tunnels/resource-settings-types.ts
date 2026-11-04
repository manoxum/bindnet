export interface HeaderValue {
  name: string;
  value: string;
}
export type SettingValue = string | number | boolean | null | HeaderValue[];
export type ResourceSettings = Record<string, SettingValue>;
export interface SettingField {
  key: string;
  label: string;
  type: "text" | "textarea" | "toggle" | "select" | "number" | "headers";
  options?: { value: string; label: string }[];
  description?: string;
}
export const generalFields: SettingField[] = [
  { key: "name", label: "Nome do recurso", type: "text" },
  { key: "niceId", label: "Identificador", type: "text" },
  {
    key: "resourcePolicyId",
    label: "ID da política compartilhada (opcional)",
    type: "number",
    description:
      "Vazio remove a política compartilhada e utiliza a política própria. Os IDs podem ser consultados no Pangolin.",
  },
];
export const proxyFields: SettingField[] = [
  { key: "ssl", label: "HTTPS público", type: "toggle" },
  { key: "stickySession", label: "Sessões persistentes", type: "toggle" },
  { key: "forwardIdentityHeaders", label: "Encaminhar cabeçalhos de identidade", type: "toggle" },
  {
    key: "loadBalancingStrategy",
    label: "Estratégia de balanceamento",
    type: "select",
    options: [
      { value: "wrr", label: "Round robin" },
      { value: "p2c", label: "Duas escolhas" },
      { value: "hrw", label: "Hash do cliente" },
      { value: "leasttime", label: "Menor tempo" },
    ],
  },
  {
    key: "requestHeaders",
    label: "Cabeçalhos de requisição",
    type: "headers",
    description: "Um por linha: Nome: valor. O Host é definido pelo domínio local.",
  },
  { key: "responseHeaders", label: "Cabeçalhos de resposta", type: "headers" },
];
export const authFields: SettingField[] = [
  {
    key: "sso",
    label: "Exigir SSO da plataforma",
    type: "toggle",
    description: "Desativar SSO pode tornar o serviço público se não houver outro método de autenticação.",
  },
  { key: "blockAccess", label: "Bloquear acesso ao recurso", type: "toggle" },
  { key: "emailWhitelistEnabled", label: "Ativar lista de e-mails permitidos", type: "toggle" },
  {
    key: "mtlsEnabled",
    label: "Exigir certificado de cliente (mTLS)",
    type: "toggle",
    description: "Cadastre uma CA pública antes de ativar.",
  },
];
export const maintenanceFields: SettingField[] = [
  { key: "maintenanceModeEnabled", label: "Ativar página de manutenção", type: "toggle" },
  {
    key: "maintenanceModeType",
    label: "Tipo de manutenção",
    type: "select",
    options: [
      { value: "automatic", label: "Automático — todos os alvos indisponíveis" },
      { value: "forced", label: "Forçado — bloqueia todo o tráfego" },
    ],
  },
  { key: "maintenanceTitle", label: "Título da página", type: "text" },
  { key: "maintenanceMessage", label: "Mensagem de manutenção", type: "textarea" },
  { key: "maintenanceEstimatedTime", label: "Conclusão estimada", type: "text" },
];
