import { z } from "zod";
import {
  quotaValueToBytes,
  bytesToQuotaValue,
  type HotspotLimits,
  type LimitType,
  type RateUnit,
} from "@/components/hotspot/hotspot-limits-types";
import { optionalPositiveDecimal, parseDecimal } from "@/components/hotspot/hotspot-number-schema";

const rateUnit = z.enum(["kbit", "mbit", "gbit", "kbyte", "mbyte", "gbyte"]);

// Schema do shape novo (tipo unico ilimitado/credito/cota[/customizado]) -
// usado por dispositivo (override) e perfil, ver HotspotLimitTypeFields.tsx.
// "custom" so e oferecido como opcao no formulario de perfil
// (HotspotLimitTypeToggle com includeCustom) - o schema aceita o valor
// nos dois casos por simplicidade (um so enum), ja que o formulario de
// dispositivo nunca produz esse valor.
// Taxa e cota aceitam fracao (1.5GB de cota, 17.5KB/s de taxa) - ver
// optionalPositiveDecimal em hotspot-number-schema.ts.
export const hotspotLimitsFormSchema = z.object({
  downloadRateValue: optionalPositiveDecimal,
  downloadRateUnit: rateUnit,
  uploadRateValue: optionalPositiveDecimal,
  uploadRateUnit: rateUnit,
  limitType: z.enum(["unlimited", "credit", "quota", "custom"]),
  dailyQuotaValue: optionalPositiveDecimal,
  dailyQuotaUnit: rateUnit,
  weeklyQuotaValue: optionalPositiveDecimal,
  weeklyQuotaUnit: rateUnit,
  monthlyQuotaValue: optionalPositiveDecimal,
  monthlyQuotaUnit: rateUnit,
});

export type HotspotLimitsFormValues = z.infer<typeof hotspotLimitsFormSchema>;

// Sem Math.round ao reidratar: a cota e gravada em bytes, entao um teto
// de 1.5GB voltava do banco como 1.5 e o arredondamento o mostrava como
// "2" no formulario - reabrir e salvar (sem tocar no campo) trocava a
// cota do perfil de 1.5GB pra 2GB silenciosamente. String() ja imprime
// a forma curta de um float ("1.5", "3"), sem casa decimal inventada.
export function limitsToFormValues(limits: HotspotLimits): HotspotLimitsFormValues {
  return {
    downloadRateValue: limits.downloadRateValue?.toString() ?? "",
    downloadRateUnit: limits.downloadRateUnit,
    uploadRateValue: limits.uploadRateValue?.toString() ?? "",
    uploadRateUnit: limits.uploadRateUnit,
    limitType: limits.limitType,
    dailyQuotaValue: limits.dailyQuotaBytes
      ? String(bytesToQuotaValue(limits.dailyQuotaBytes, limits.dailyQuotaUnit))
      : "",
    dailyQuotaUnit: limits.dailyQuotaUnit,
    weeklyQuotaValue: limits.weeklyQuotaBytes
      ? String(bytesToQuotaValue(limits.weeklyQuotaBytes, limits.weeklyQuotaUnit))
      : "",
    weeklyQuotaUnit: limits.weeklyQuotaUnit,
    monthlyQuotaValue: limits.monthlyQuotaBytes
      ? String(bytesToQuotaValue(limits.monthlyQuotaBytes, limits.monthlyQuotaUnit))
      : "",
    monthlyQuotaUnit: limits.monthlyQuotaUnit,
  };
}

// parseDecimal (nao Number) porque o campo aceita virgula como
// separador decimal - ver hotspot-number-schema.ts.
export function formValuesToLimits(values: HotspotLimitsFormValues): HotspotLimits {
  return {
    downloadRateValue: values.downloadRateValue ? parseDecimal(values.downloadRateValue) : null,
    downloadRateUnit: values.downloadRateUnit as RateUnit,
    uploadRateValue: values.uploadRateValue ? parseDecimal(values.uploadRateValue) : null,
    uploadRateUnit: values.uploadRateUnit as RateUnit,
    limitType: values.limitType as LimitType,
    dailyQuotaBytes: values.dailyQuotaValue
      ? quotaValueToBytes(parseDecimal(values.dailyQuotaValue), values.dailyQuotaUnit as RateUnit)
      : null,
    dailyQuotaUnit: values.dailyQuotaUnit as RateUnit,
    weeklyQuotaBytes: values.weeklyQuotaValue
      ? quotaValueToBytes(parseDecimal(values.weeklyQuotaValue), values.weeklyQuotaUnit as RateUnit)
      : null,
    weeklyQuotaUnit: values.weeklyQuotaUnit as RateUnit,
    monthlyQuotaBytes: values.monthlyQuotaValue
      ? quotaValueToBytes(parseDecimal(values.monthlyQuotaValue), values.monthlyQuotaUnit as RateUnit)
      : null,
    monthlyQuotaUnit: values.monthlyQuotaUnit as RateUnit,
  };
}
