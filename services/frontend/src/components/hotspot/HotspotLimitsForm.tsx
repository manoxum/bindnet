import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@/components/ui/button";
import {
  hotspotLimitsFormSchema,
  limitsToFormValues,
  formValuesToLimits,
  type HotspotLimitsFormValues,
} from "@/components/hotspot/hotspot-limits-schema";
import type { HotspotLimits } from "@/components/hotspot/hotspot-limits-types";
import { HotspotRateQuotaFields } from "@/components/hotspot/HotspotRateQuotaFields";

interface HotspotLimitsFormProps {
  value: HotspotLimits;
  onSubmit: (limits: HotspotLimits) => void;
  pending: boolean;
}

// Formulário genérico de limite (taxa valor+unidade + cota GB/período
// + taxa de throttle pós-cota), reusado tanto pelo limite global
// quanto pelo limite por dispositivo - só muda o que o container faz
// com o valor enviado em onSubmit.
export function HotspotLimitsForm({ value, onSubmit, pending }: HotspotLimitsFormProps) {
  const {
    register,
    handleSubmit,
    formState: { errors, isDirty },
  } = useForm<HotspotLimitsFormValues>({
    resolver: zodResolver(hotspotLimitsFormSchema),
    values: limitsToFormValues(value),
  });

  return (
    <form className="space-y-6" onSubmit={handleSubmit((values) => onSubmit(formValuesToLimits(values)))}>
      <HotspotRateQuotaFields register={register} errors={errors} />

      <Button type="submit" disabled={!isDirty || pending}>
        Salvar
      </Button>
    </form>
  );
}
