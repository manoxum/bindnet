import { z } from "zod";

export const configSchema = z.object({
  WIFI_SSID: z.string().min(1, "Informe o SSID"),
  WIFI_PASSWORD: z.string().min(8, "Mínimo de 8 caracteres (WPA2)"),
  WIFI_INTERFACE: z.string().min(1, "Selecione a interface Wi-Fi"),
  INTERNET_INTERFACE: z.string().min(1, "Selecione a interface de internet"),
  WIFI_COUNTRY: z.string().min(2).max(2),
  WIFI_CHANNEL: z.string().min(1),
  WIFI_FREQ_BAND: z.string().min(1),
});
export type ConfigForm = z.infer<typeof configSchema>;
