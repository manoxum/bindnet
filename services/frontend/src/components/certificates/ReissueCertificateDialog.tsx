import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { IssueCertificateForm } from "@/components/certificates/IssueCertificateForm";
import type { Certificate, ReissueCertificateRequest } from "@/components/certificates/certificate-types";

interface ReissueCertificateDialogProps {
  certificate: Certificate | null;
  pending: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (id: string, request: ReissueCertificateRequest) => void;
}

export function ReissueCertificateDialog({
  certificate,
  pending,
  onOpenChange,
  onSubmit,
}: ReissueCertificateDialogProps) {
  return (
    <Dialog open={certificate !== null} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>Editar e reemitir certificado</DialogTitle>
          <DialogDescription>
            Uma nova chave e um novo certificado serão gerados. A versão atual será movida para Revogados.
          </DialogDescription>
        </DialogHeader>
        {certificate ? (
          <IssueCertificateForm
            pending={pending}
            submitLabel="Reemitir certificado"
            lockName
            defaultValues={{
              name: certificate.name,
              domains: [...(certificate.dnsNames ?? []), ...(certificate.ipAddresses ?? [])].join("\n"),
              validityQuantity: String(certificate.validityQuantity),
              validityUnit: certificate.validityUnit,
            }}
            onSubmit={({ domains, validityQuantity, validityUnit }) =>
              onSubmit(certificate.id, { domains, validityQuantity, validityUnit })
            }
          />
        ) : null}
      </DialogContent>
    </Dialog>
  );
}
