import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { HotspotRateQuotaFields } from "@/components/hotspot/HotspotRateQuotaFields";
import { HotspotCreditConfigFields } from "@/components/hotspot/HotspotCreditConfigFields";
import {
  hotspotProfileFormSchema,
  profileToFormValues,
  formValuesToProfile,
  type HotspotProfileFormValues,
} from "@/components/hotspot/hotspot-profile-schema";
import type { HotspotProfile, HotspotProfileRequest } from "@/components/hotspot/hotspot-profile-types";

interface HotspotProfileFormProps {
  value: HotspotProfile;
  onSubmit: (profile: HotspotProfileRequest) => void;
  pending: boolean;
}

// Formulario de perfil - mesmo shape de campos de taxa/cota/throttle do
// HotspotLimitsForm (via HotspotRateQuotaFields) + politica de credito
// (via HotspotCreditConfigFields, mesma usada por DeviceCreditCard) +
// nome. Um perfil combina os dois num so formulario/submit.
export function HotspotProfileForm({ value, onSubmit, pending }: HotspotProfileFormProps) {
  const {
    register,
    handleSubmit,
    watch,
    formState: { errors, isDirty },
  } = useForm<HotspotProfileFormValues>({
    resolver: zodResolver(hotspotProfileFormSchema),
    values: profileToFormValues(value),
  });
  const enabled = watch("enabled");

  return (
    <form className="space-y-6" onSubmit={handleSubmit((values) => onSubmit(formValuesToProfile(values)))}>
      <div className="space-y-2">
        <Label htmlFor="name">Nome</Label>
        <Input id="name" placeholder="ex.: Convidados" {...register("name")} disabled={value.isDefault} />
        {errors.name && <p className="text-sm text-destructive">{errors.name.message}</p>}
      </div>

      <HotspotRateQuotaFields register={register} errors={errors} />
      <HotspotCreditConfigFields register={register} enabled={enabled} />

      <Button type="submit" disabled={!isDirty || pending}>
        Salvar
      </Button>
    </form>
  );
}
