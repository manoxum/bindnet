import { Download, ShieldCheck, ShieldX, Trash2 } from "lucide-react";
import { Button, buttonVariants } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { allCertificateDomains, type Certificate } from "@/components/certificates/certificate-types";
import type { CertificateListProps } from "@/components/certificates/CertificateTable";

interface CertificateCardProps {
  certificate: Certificate;
  revoked?: boolean;
  revokePending?: boolean;
  permanentDeletePending?: boolean;
  onRevoke?: (id: string) => void;
  onPermanentDelete?: (id: string) => void;
}

function CertificateCard({
  certificate,
  revoked = false,
  revokePending = false,
  permanentDeletePending = false,
  onRevoke,
  onPermanentDelete,
}: CertificateCardProps) {
  const extraDomains = allCertificateDomains(certificate).filter((name) => name !== certificate.domain);
  return (
    <div className="rounded-lg border bg-card p-4 text-card-foreground shadow-sm transition hover:-translate-y-0.5 hover:border-foreground/20 hover:shadow-md">
      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 items-center gap-3">
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md border bg-muted/30">
            {revoked ? (
              <ShieldX className="h-4 w-4 text-muted-foreground" />
            ) : (
              <ShieldCheck className="h-4 w-4 text-primary" />
            )}
          </div>
          <p className="truncate font-medium">{certificate.domain}</p>
        </div>
        <Badge variant={revoked ? "secondary" : "success"}>{revoked ? "revogado" : "válido"}</Badge>
      </div>
      {extraDomains.length > 0 && (
        <p className="mt-2 truncate text-xs text-muted-foreground" title={extraDomains.join(", ")}>
          + {extraDomains.join(", ")}
        </p>
      )}
      <p className="mt-3 text-xs text-muted-foreground">
        {revoked ? "Revogado em" : "Expira em"}{" "}
        {new Date(revoked ? certificate.revokedAt! : certificate.expiresAt).toLocaleDateString()}
      </p>
      <div className="mt-4 flex justify-end gap-2">
        <a
          href={`/api/certificates/${certificate.id}/download`}
          download
          className={buttonVariants({ variant: "outline", size: "sm" })}
        >
          <Download className="h-4 w-4" />
        </a>
        {revoked ? (
          <Button
            variant="destructive"
            size="sm"
            disabled={permanentDeletePending}
            onClick={() => onPermanentDelete?.(certificate.id)}
          >
            <Trash2 className="h-4 w-4" />
            Eliminar
          </Button>
        ) : (
          <Button variant="destructive" size="sm" disabled={revokePending} onClick={() => onRevoke?.(certificate.id)}>
            <Trash2 className="h-4 w-4" />
          </Button>
        )}
      </div>
    </div>
  );
}

export function CertificateGrid({ certificates, emptyMessage, ...rest }: CertificateListProps) {
  if (certificates.length === 0) {
    return (
      <div className="rounded-md border border-dashed bg-muted/20 px-4 py-8 text-center text-sm text-muted-foreground">
        {emptyMessage}
      </div>
    );
  }

  return (
    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
      {certificates.map((certificate) => (
        <CertificateCard key={certificate.id} certificate={certificate} {...rest} />
      ))}
    </div>
  );
}
