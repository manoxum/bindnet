import { z } from "zod";
import { quotaValueToBytes, type RateUnit } from "@/components/hotspot/hotspot-limits-types";
import type { HotspotVoucherIssueRequest } from "@/components/hotspot/hotspot-voucher-types";

export const hotspotVoucherIssueFormSchema = z.object({
  amount: z
    .string()
    .trim()
    .refine((value) => /^\d+$/.test(value) && Number(value) > 0, "Informe um valor positivo"),
  amountUnit: z.enum(["kbit", "mbit", "gbit", "kbyte", "mbyte", "gbyte"]),
  quantity: z
    .string()
    .trim()
    .refine((value) => /^\d+$/.test(value) && Number(value) > 0 && Number(value) <= 100, "Entre 1 e 100"),
  note: z.string().trim().max(200, "No máximo 200 caracteres"),
});

export type HotspotVoucherIssueFormValues = z.infer<typeof hotspotVoucherIssueFormSchema>;

export const emptyVoucherIssueForm: HotspotVoucherIssueFormValues = {
  amount: "1",
  amountUnit: "gbyte" as RateUnit,
  quantity: "1",
  note: "",
};

export function formValuesToVoucherIssueRequest(values: HotspotVoucherIssueFormValues): HotspotVoucherIssueRequest {
  return {
    amountBytes: quotaValueToBytes(Number(values.amount), values.amountUnit),
    amountUnit: values.amountUnit,
    quantity: Number(values.quantity),
    note: values.note.trim() || undefined,
  };
}
