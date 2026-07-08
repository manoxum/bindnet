import type { UseFormRegister } from "react-hook-form";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { SelectNative } from "@/components/ui/select-native";

// Fieldset de politica de credito compartilhado por DeviceCreditCard
// (config por dispositivo) e HotspotProfileForm (perfil) - os dois usam
// os mesmos nomes de campo (enabled/rechargeAmountGB/rechargePeriod/
// plafondGB), so muda o que o formulario pai faz no submit. register
// fica frouxamente tipado (UseFormRegister<any>) de proposito, ja que
// este fieldset e reusado por schemas zod diferentes que so
// compartilham esse subconjunto de campos.
export function HotspotCreditConfigFields({
  register,
  enabled,
  idPrefix = "",
}: {
  register: UseFormRegister<any>;
  enabled: boolean;
  idPrefix?: string;
}) {
  return (
    <>
      <div className="flex items-center gap-2">
        <input id={`${idPrefix}enabled`} type="checkbox" className="h-4 w-4" {...register("enabled")} />
        <Label htmlFor={`${idPrefix}enabled`}>Exigir crédito para trafegar</Label>
      </div>

      {enabled && (
        <fieldset className="space-y-4">
          <legend className="text-sm font-medium text-muted-foreground">Recarga automática (opcional)</legend>
          <div className="grid gap-4 sm:grid-cols-3">
            <div className="space-y-2">
              <Label htmlFor={`${idPrefix}rechargeAmountGB`}>Recarga por período (GB)</Label>
              <Input id={`${idPrefix}rechargeAmountGB`} placeholder="só manual" {...register("rechargeAmountGB")} />
            </div>
            <div className="space-y-2">
              <Label htmlFor={`${idPrefix}rechargePeriod`}>Período</Label>
              <SelectNative id={`${idPrefix}rechargePeriod`} {...register("rechargePeriod")}>
                <option value="daily">Diário</option>
                <option value="weekly">Semanal</option>
                <option value="monthly">Mensal</option>
              </SelectNative>
            </div>
            <div className="space-y-2">
              <Label htmlFor={`${idPrefix}plafondGB`}>Plafond - teto do saldo (GB)</Label>
              <Input id={`${idPrefix}plafondGB`} placeholder="sem teto" {...register("plafondGB")} />
            </div>
          </div>
        </fieldset>
      )}
    </>
  );
}
