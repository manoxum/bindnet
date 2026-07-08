import { z } from "zod";
import { hotspotLimitsFormSchema, limitsToFormValues, formValuesToLimits } from "@/components/hotspot/hotspot-limits-schema";
import { GIGABYTE, bytesToGB } from "@/components/hotspot/hotspot-limits-types";
import type { HotspotProfile, HotspotProfileRequest } from "@/components/hotspot/hotspot-profile-types";

const optionalPositiveInt = z
  .string()
  .trim()
  .refine((value) => value === "" || (/^\d+$/.test(value) && Number(value) > 0), "Deve ser um número positivo");

// Estende o mesmo schema de taxa/cota/throttle do limite (HotspotLimitsForm)
// com nome + politica de credito - um perfil e um bundle dos dois.
export const hotspotProfileFormSchema = hotspotLimitsFormSchema.extend({
  name: z.string().trim().min(1, "Informe um nome"),
  enabled: z.boolean(),
  rechargeAmountGB: optionalPositiveInt,
  rechargePeriod: z.enum(["daily", "weekly", "monthly"]),
  plafondGB: optionalPositiveInt,
});

export type HotspotProfileFormValues = z.infer<typeof hotspotProfileFormSchema>;

export function profileToFormValues(profile: HotspotProfile): HotspotProfileFormValues {
  return {
    ...limitsToFormValues(profile),
    name: profile.name,
    enabled: profile.creditEnabled,
    rechargeAmountGB: profile.creditRechargeAmountBytes
      ? String(Math.round(bytesToGB(profile.creditRechargeAmountBytes)))
      : "",
    rechargePeriod: profile.creditRechargePeriod ?? "daily",
    plafondGB: profile.creditPlafondBytes ? String(Math.round(bytesToGB(profile.creditPlafondBytes))) : "",
  };
}

export function formValuesToProfile(values: HotspotProfileFormValues): HotspotProfileRequest {
  return {
    ...formValuesToLimits(values),
    name: values.name.trim(),
    creditEnabled: values.enabled,
    creditRechargeAmountBytes: values.rechargeAmountGB ? Number(values.rechargeAmountGB) * GIGABYTE : null,
    creditRechargePeriod: values.rechargeAmountGB ? values.rechargePeriod : null,
    creditPlafondBytes: values.plafondGB ? Number(values.plafondGB) * GIGABYTE : null,
  };
}
