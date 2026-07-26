import { z } from "zod";
import { hotspotLimitsFormSchema, limitsToFormValues, formValuesToLimits } from "@/components/hotspot/hotspot-device-limits-schema";
import { GIGABYTE, bytesToGB } from "@/components/hotspot/hotspot-limits-types";
import { optionalPositiveInt } from "@/components/hotspot/hotspot-number-schema";
import type { HotspotProfile, HotspotProfileRequest } from "@/components/hotspot/hotspot-profile-types";

// Estende o mesmo schema de taxa/tipo/cota do limite de dispositivo
// (hotspot-device-limits-schema.ts) com nome + politica de recarga de
// credito - um perfil e um bundle dos dois. "enabled" nao existe mais
// aqui: o proprio limitType (herdado do extend) decide se a politica de
// credito abaixo esta em uso, ver HotspotLimitTypeFields.tsx.
export const hotspotProfileFormSchema = hotspotLimitsFormSchema.extend({
  name: z.string().trim().min(1, "Informe um nome"),
  rechargeAmountGB: optionalPositiveInt,
  rechargePeriod: z.enum(["daily", "weekly", "monthly"]),
  plafondGB: optionalPositiveInt,
  // Politica do tipo "time" (minutos na UI, convertidos para segundos na
  // API). Deadline vem/volta como valor de <input type="datetime-local">.
  timeMode: z.enum(["budget", "deadline"]),
  timeBudgetMinutes: optionalPositiveInt,
  timeRechargePeriod: z.enum(["daily", "weekly", "monthly"]),
  timePlafondMinutes: optionalPositiveInt,
  timeDeadlineAt: z.string().optional(),
});

export type HotspotProfileFormValues = z.infer<typeof hotspotProfileFormSchema>;

// Converte um instante ISO (RFC3339 da API) para o formato aceito por
// <input type="datetime-local"> ("YYYY-MM-DDTHH:mm", hora local) e vice-
// versa. new Date(local).toISOString() devolve UTC com Z, que o Go parseia
// como RFC3339 sem problema.
function isoToDatetimeLocal(iso: string): string {
  const d = new Date(iso);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

export function profileToFormValues(profile: HotspotProfile): HotspotProfileFormValues {
  return {
    ...limitsToFormValues(profile),
    name: profile.name,
    rechargeAmountGB: profile.creditRechargeAmountBytes
      ? String(Math.round(bytesToGB(profile.creditRechargeAmountBytes)))
      : "",
    rechargePeriod: profile.creditRechargePeriod ?? "daily",
    plafondGB: profile.creditPlafondBytes ? String(Math.round(bytesToGB(profile.creditPlafondBytes))) : "",
    timeMode: profile.timeMode ?? "budget",
    timeBudgetMinutes: profile.timeRechargeSeconds ? String(Math.round(profile.timeRechargeSeconds / 60)) : "",
    timeRechargePeriod: profile.timeRechargePeriod ?? "daily",
    timePlafondMinutes: profile.timePlafondSeconds ? String(Math.round(profile.timePlafondSeconds / 60)) : "",
    timeDeadlineAt: profile.timeDeadlineAt ? isoToDatetimeLocal(profile.timeDeadlineAt) : "",
  };
}

export function formValuesToProfile(values: HotspotProfileFormValues): HotspotProfileRequest {
  const budget = values.timeMode === "budget";
  return {
    ...formValuesToLimits(values),
    name: values.name.trim(),
    creditRechargeAmountBytes: values.rechargeAmountGB ? Number(values.rechargeAmountGB) * GIGABYTE : null,
    creditRechargePeriod: values.rechargeAmountGB ? values.rechargePeriod : null,
    creditPlafondBytes: values.plafondGB ? Number(values.plafondGB) * GIGABYTE : null,
    timeMode: values.timeMode,
    timeRechargeSeconds: budget && values.timeBudgetMinutes ? Number(values.timeBudgetMinutes) * 60 : null,
    timeRechargePeriod: budget && values.timeBudgetMinutes ? values.timeRechargePeriod : null,
    timePlafondSeconds: budget && values.timePlafondMinutes ? Number(values.timePlafondMinutes) * 60 : null,
    timeDeadlineAt: !budget && values.timeDeadlineAt ? new Date(values.timeDeadlineAt).toISOString() : null,
  };
}
