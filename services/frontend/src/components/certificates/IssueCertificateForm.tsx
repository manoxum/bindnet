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
}

export function IssueCertificateForm({ onSubmit, pending }: IssueCertificateFormProps) {
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<CertificateIssueFormValues>({
    resolver: zodResolver(certificateIssueFormSchema),
    defaultValues: emptyCertificateIssueForm,
  });

  return (
    <form
      className="space-y-4"
      onSubmit={handleSubmit((values) => {
        onSubmit(formValuesToIssueCertificateRequest(values));
        reset(emptyCertificateIssueForm);
      })}
    >
      <div className="space-y-2">
        <Label htmlFor="domains">Domínios ou IPs</Label>
        <Textarea
          id="domains"
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
          <Label htmlFor="validityQuantity">Validade</Label>
          <Input id="validityQuantity" type="number" min={1} className="w-24" {...register("validityQuantity")} />
        </div>
        <div className="space-y-2">
          <Label htmlFor="validityUnit">Período</Label>
          <SelectNative id="validityUnit" className="w-32" {...register("validityUnit")}>
            <option value="days">Dias</option>
            <option value="weeks">Semanas</option>
            <option value="months">Meses</option>
            <option value="years">Anos</option>
          </SelectNative>
        </div>
        <Button type="submit" disabled={pending}>
          Emitir
        </Button>
      </div>
      {errors.validityQuantity && <p className="text-sm text-destructive">{errors.validityQuantity.message}</p>}
    </form>
  );
}
