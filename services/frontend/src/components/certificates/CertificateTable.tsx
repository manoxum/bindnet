import { Download, Trash2 } from "lucide-react";
import { Button, buttonVariants } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { allCertificateDomains, type Certificate } from "@/components/certificates/certificate-types";

export interface CertificateListProps {
  certificates: Certificate[];
  emptyMessage: string;
  revoked?: boolean;
  revokePending?: boolean;
  permanentDeletePending?: boolean;
  onRevoke?: (id: string) => void;
  onPermanentDelete?: (id: string) => void;
}

export function CertificateTable({
  certificates,
  emptyMessage,
  revoked = false,
  revokePending = false,
  permanentDeletePending = false,
  onRevoke,
  onPermanentDelete,
}: CertificateListProps) {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Domínio</TableHead>
          <TableHead>{revoked ? "Revogado em" : "Expira em"}</TableHead>
          <TableHead>Status</TableHead>
          <TableHead className="text-right">Ações</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {certificates.map((cert) => {
          const extraDomains = allCertificateDomains(cert).filter((name) => name !== cert.domain);
          return (
            <TableRow key={cert.id}>
              <TableCell>
                <div>{cert.domain}</div>
                {extraDomains.length > 0 && (
                  <div className="truncate text-xs text-muted-foreground" title={extraDomains.join(", ")}>
                    + {extraDomains.join(", ")}
                  </div>
                )}
              </TableCell>
              <TableCell>{new Date(revoked ? cert.revokedAt! : cert.expiresAt).toLocaleDateString()}</TableCell>
              <TableCell>
                <Badge variant={revoked ? "secondary" : "success"}>{revoked ? "revogado" : "válido"}</Badge>
              </TableCell>
              <TableCell className="flex justify-end gap-2">
                <a
                  href={`/api/certificates/${cert.id}/download`}
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
                    onClick={() => onPermanentDelete?.(cert.id)}
                  >
                    <Trash2 className="h-4 w-4" />
                    Eliminar
                  </Button>
                ) : (
                  <Button variant="destructive" size="sm" disabled={revokePending} onClick={() => onRevoke?.(cert.id)}>
                    <Trash2 className="h-4 w-4" />
                  </Button>
                )}
              </TableCell>
            </TableRow>
          );
        })}
        {certificates.length === 0 && (
          <TableRow>
            <TableCell colSpan={4} className="text-center text-muted-foreground">
              {emptyMessage}
            </TableCell>
          </TableRow>
        )}
      </TableBody>
    </Table>
  );
}
