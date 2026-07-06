import { z } from "zod";

export const configSchema = z.object({
  WIFI_SSID: z.string().min(1, "Informe o SSID"),
  WIFI_PASSWORD: z.string().min(8, "Mínimo de 8 caracteres (WPA2)"),
  WIFI_INTERFACE: z.string().min(1, "Selecione a interface Wi-Fi"),
  INTERNET_INTERFACE: z.string().min(1, "Selecione a interface de internet"),
  WIFI_COUNTRY: z.string().min(2).max(2),
  WIFI_CHANNEL: z.string().min(1),
  WIFI_FREQ_BAND: z.string().min(1),
  WIFI_CHANNEL_CANDIDATES: z.string(),
  HOTSPOT_GATEWAY: z.string().min(1, "Informe o gateway do hotspot"),
  HOTSPOT_CIDR: z.string().min(1, "Informe a faixa CIDR do hotspot"),
  HOTSPOT_DNS_FALLBACKS: z.string(),
  BINDNET_UPLINK_INTERFACE: z.string().min(1, "Informe o uplink virtual"),
  UPLINK_MONITOR_INTERVAL: z.string().min(1, "Informe o intervalo"),
});
export type ConfigForm = z.infer<typeof configSchema>;
