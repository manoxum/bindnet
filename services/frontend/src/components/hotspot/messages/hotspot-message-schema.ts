import { z } from "zod";
import type { HotspotMessageCreateRequest } from "@/components/hotspot/messages/hotspot-message-types";

export const hotspotMessageFormSchema = z.object({
  title: z.string().max(200, "Título muito longo").optional(),
  body: z.string().trim().min(1, "Escreva a mensagem").max(2000, "Mensagem muito longa"),
  targetMac: z.string().optional(),
  urgent: z.boolean(),
  // datetime-local (hora local do operador); vazio = sem expiração.
  expiresAt: z.string().optional(),
});

export type HotspotMessageFormValues = z.infer<typeof hotspotMessageFormSchema>;

export const emptyHotspotMessageForm: HotspotMessageFormValues = {
  title: "",
  body: "",
  targetMac: "",
  urgent: false,
  expiresAt: "",
};

// formValuesToCreateRequest normaliza o formulario para o corpo da API:
// campos vazios viram ausentes e o expiresAt local vira RFC3339 (UTC).
export function formValuesToCreateRequest(values: HotspotMessageFormValues): HotspotMessageCreateRequest {
  const request: HotspotMessageCreateRequest = {
    body: values.body.trim(),
    urgent: values.urgent,
  };
  const title = values.title?.trim();
  if (title) request.title = title;
  const targetMac = values.targetMac?.trim();
  if (targetMac) request.targetMac = targetMac;
  const expiresAt = values.expiresAt?.trim();
  if (expiresAt) request.expiresAt = new Date(expiresAt).toISOString();
  return request;
}
