import { z } from "zod";
import type { CertificateValidityUnit, IssueCertificateRequest } from "@/components/certificates/certificate-types";

export const certificateIssueFormSchema = z.object({
  domains: z.string().trim().min(1, "Informe ao menos um domínio ou IP"),
  validityQuantity: z
    .string()
    .trim()
    .refine((value) => /^\d+$/.test(value) && Number(value) > 0, "Informe uma quantidade positiva"),
  validityUnit: z.enum(["days", "weeks", "months", "years"]),
});

export type CertificateIssueFormValues = z.infer<typeof certificateIssueFormSchema>;

export const emptyCertificateIssueForm: CertificateIssueFormValues = {
  domains: "",
  validityQuantity: "2",
  validityUnit: "years" as CertificateValidityUnit,
};

// parseDomainsInput separa por vírgula ou quebra de linha, remove espaços em
// branco e duplicatas - permite emitir um único certificado cobrindo um
// domínio curinga (ex.: *.mydomain) e vários subdomínios explícitos juntos.
function parseDomainsInput(value: string): string[] {
  const seen = new Set<string>();
  const result: string[] = [];
  for (const raw of value.split(/[,\n]/)) {
    const trimmed = raw.trim();
    if (trimmed && !seen.has(trimmed)) {
      seen.add(trimmed);
      result.push(trimmed);
    }
  }
  return result;
}

export function formValuesToIssueCertificateRequest(values: CertificateIssueFormValues): IssueCertificateRequest {
  return {
    domains: parseDomainsInput(values.domains),
    validityQuantity: Number(values.validityQuantity),
    validityUnit: values.validityUnit,
  };
}
