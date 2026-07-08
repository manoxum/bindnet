import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { SelectNative } from "@/components/ui/select-native";
import { RateUnitOptions } from "@/components/hotspot/RateUnitOptions";
import {
  hotspotVoucherIssueFormSchema,
  emptyVoucherIssueForm,
  formValuesToVoucherIssueRequest,
  type HotspotVoucherIssueFormValues,
} from "@/components/hotspot/hotspot-voucher-schema";
import type { HotspotVoucherIssueRequest } from "@/components/hotspot/hotspot-voucher-types";

interface HotspotVoucherIssueFormProps {
  onSubmit: (request: HotspotVoucherIssueRequest) => void;
  pending: boolean;
}

export function HotspotVoucherIssueForm({ onSubmit, pending }: HotspotVoucherIssueFormProps) {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<HotspotVoucherIssueFormValues>({
    resolver: zodResolver(hotspotVoucherIssueFormSchema),
    defaultValues: emptyVoucherIssueForm,
  });

  return (
    <form className="space-y-4" onSubmit={handleSubmit((values) => onSubmit(formValuesToVoucherIssueRequest(values)))}>
      <div className="space-y-2">
        <Label htmlFor="amount">Valor por voucher</Label>
        <div className="flex gap-2">
          <Input id="amount" type="number" min={1} className="flex-1" {...register("amount")} />
          <SelectNative id="amountUnit" className="w-24" {...register("amountUnit")}>
            <RateUnitOptions />
          </SelectNative>
        </div>
        {errors.amount && <p className="text-sm text-destructive">{errors.amount.message}</p>}
      </div>

      <div className="space-y-2">
        <Label htmlFor="quantity">Quantidade</Label>
        <Input id="quantity" type="number" min={1} max={100} {...register("quantity")} />
        {errors.quantity && <p className="text-sm text-destructive">{errors.quantity.message}</p>}
      </div>

      <div className="space-y-2">
        <Label htmlFor="note">Nota (opcional)</Label>
        <Input id="note" placeholder="ex.: lote para revenda" {...register("note")} />
        {errors.note && <p className="text-sm text-destructive">{errors.note.message}</p>}
      </div>

      <Button type="submit" disabled={pending}>
        Emitir
      </Button>
    </form>
  );
}
