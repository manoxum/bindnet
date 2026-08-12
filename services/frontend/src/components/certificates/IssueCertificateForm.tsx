import { useId } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { SelectNative } from "@/components/ui/select-native";
import { Textarea } from "@/components/ui/textarea";
import {
  certificateIssueFormSchema,
  emptyCertificateIssueForm,
  formValuesToIssueCertificateRequest,
  type CertificateIssueFormValues,
} from "@/components/certificates/certificate-issue-schema";
import type { IssueCertificateRequest } from "@/components/certificates/certificate-types";

interface IssueCertificateFormProps {
  onSubmit: (request: IssueCertificateRequest) => void;
  pending: boolean;
  defaultValues?: CertificateIssueFormValues;
  submitLabel?: string;
  lockName?: boolean;
}

export function IssueCertificateForm({
  onSubmit,
  pending,
  defaultValues = emptyCertificateIssueForm,
  submitLabel = "Emitir",
  lockName = false,
}: IssueCertificateFormProps) {
  const fieldId = useId();
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<CertificateIssueFormValues>({
    resolver: zodResolver(certificateIssueFormSchema),
    defaultValues,
  });

  return (
    <form
      className="space-y-4"
      onSubmit={handleSubmit((values) => {
        onSubmit(formValuesToIssueCertificateRequest(values));
        if (!lockName) reset(emptyCertificateIssueForm);
      })}
    >
      <div className="space-y-2">
        <Label htmlFor={`${fieldId}-certificate-name`}>Nome do certificado</Label>
        <Input
          id={`${fieldId}-certificate-name`}
          placeholder="Portal interno"
          readOnly={lockName}
          aria-describedby={`${fieldId}-certificate-name-help`}
          {...register("name")}
        />
        <p id={`${fieldId}-certificate-name-help`} className="text-xs text-muted-foreground">
          Nome usado para identificar o certificado no Bindnet e no nginx-ui.
        </p>
        {errors.name && <p className="text-sm text-destructive">{errors.name.message}</p>}
      </div>

      <div className="space-y-2">
        <Label htmlFor={`${fieldId}-domains`}>Domínios ou IPs</Label>
        <Textarea
          id={`${fieldId}-domains`}
          rows={3}
          placeholder={"*.mydomain\napp.mydomain\napp2.mydomain"}
          {...register("domains")}
        />
        <p className="text-xs text-muted-foreground">
          Um por linha ou separados por vírgula. Todos entram no mesmo certificado (SAN); aceita domínio curinga (ex.:
          *.mydomain).
        </p>
        {errors.domains && <p className="text-sm text-destructive">{errors.domains.message}</p>}
      </div>

      <div className="flex flex-wrap items-end gap-2">
        <div className="space-y-2">
          <Label htmlFor={`${fieldId}-validity-quantity`}>Validade</Label>
          <Input
            id={`${fieldId}-validity-quantity`}
            type="number"
            min={1}
            className="w-24"
            {...register("validityQuantity")}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor={`${fieldId}-validity-unit`}>Período</Label>
          <SelectNative id={`${fieldId}-validity-unit`} className="w-32" {...register("validityUnit")}>
            <option value="days">Dias</option>
            <option value="weeks">Semanas</option>
            <option value="months">Meses</option>
            <option value="years">Anos</option>
          </SelectNative>
        </div>
        <Button type="submit" disabled={pending}>
          {submitLabel}
        </Button>
      </div>
      {errors.validityQuantity && <p className="text-sm text-destructive">{errors.validityQuantity.message}</p>}
    </form>
  );
}
