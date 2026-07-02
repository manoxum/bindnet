import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Download, Trash2 } from "lucide-react";
import { Button, buttonVariants } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { api, ApiError } from "@/lib/api";

interface Certificate {
  id: string;
  domain: string;
  commonName: string;
  issuedAt: string;
  expiresAt: string;
  revokedAt?: string;
}

const issueSchema = z.object({
  domain: z.string().min(1, "Informe um domínio ou IP"),
});
type IssueForm = z.infer<typeof issueSchema>;

export function CertificatesPage() {
  const queryClient = useQueryClient();

  const certificates = useQuery<Certificate[]>({
    queryKey: ["certificates"],
    queryFn: () => api.get<Certificate[]>("/certificates"),
  });

  const { register, handleSubmit, reset, formState: { errors } } = useForm<IssueForm>({
    resolver: zodResolver(issueSchema),
  });

  const issue = useMutation({
    mutationFn: (data: IssueForm) => api.post("/certificates", data),
    onSuccess: () => {
      toast.success("Certificado emitido.");
      reset();
      queryClient.invalidateQueries({ queryKey: ["certificates"] });
    },
    onError: (error) => toast.error(error instanceof ApiError ? error.message : "Falha ao emitir certificado"),
  });

  const revoke = useMutation({
    mutationFn: (id: string) => api.del(`/certificates/${id}`),
    onSuccess: () => {
      toast.success("Certificado revogado.");
      queryClient.invalidateQueries({ queryKey: ["certificates"] });
    },
    onError: (error) => toast.error(error instanceof ApiError ? error.message : "Falha ao revogar"),
  });

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold">Certificados (CA local)</h1>
      <p className="text-sm text-muted-foreground">
        Emita, liste, revogue e baixe certificados assinados pela CA local do painel. Nada escuta mais nas portas
        80/443 - a emissão agora é sempre uma ação explícita aqui.
      </p>

      <Card>
        <CardHeader>
          <CardTitle>Autoridade certificadora</CardTitle>
          <CardDescription>Certificado raiz usado para assinar os certificados abaixo.</CardDescription>
        </CardHeader>
        <CardContent className="flex items-center justify-between">
          <span className="text-sm text-muted-foreground">
            Importe este certificado nos dispositivos que devem confiar nos certificados emitidos abaixo.
          </span>
          <a href="/api/certificates/ca" download className={buttonVariants({ variant: "outline" })}>
            <Download className="mr-2 h-4 w-4" />
            Baixar CA
          </a>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Emitir certificado</CardTitle>
          <CardDescription>Emite um novo certificado assinado pela CA local para o domínio/IP informado.</CardDescription>
        </CardHeader>
        <CardContent>
          <form className="flex items-end gap-2" onSubmit={handleSubmit((data) => issue.mutate(data))}>
            <div className="flex-1 space-y-2">
              <Label htmlFor="domain">Domínio ou IP</Label>
              <Input id="domain" placeholder="ex.: painel.local" {...register("domain")} />
              {errors.domain && <p className="text-sm text-destructive">{errors.domain.message}</p>}
            </div>
            <Button type="submit" disabled={issue.isPending}>
              Emitir
            </Button>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Certificados emitidos</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Domínio</TableHead>
                <TableHead>Expira em</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Ações</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(certificates.data ?? []).map((cert) => (
                <TableRow key={cert.id}>
                  <TableCell>{cert.domain}</TableCell>
                  <TableCell>{new Date(cert.expiresAt).toLocaleDateString()}</TableCell>
                  <TableCell>
                    <Badge variant={cert.revokedAt ? "secondary" : "success"}>
                      {cert.revokedAt ? "revogado" : "válido"}
                    </Badge>
                  </TableCell>
                  <TableCell className="flex justify-end gap-2">
                    <a
                      href={`/api/certificates/${cert.id}/download`}
                      download
                      className={buttonVariants({ variant: "outline", size: "sm" })}
                    >
                      <Download className="h-4 w-4" />
                    </a>
                    <Button
                      variant="destructive"
                      size="sm"
                      disabled={!!cert.revokedAt || revoke.isPending}
                      onClick={() => revoke.mutate(cert.id)}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
              {(certificates.data ?? []).length === 0 && (
                <TableRow>
                  <TableCell colSpan={4} className="text-center text-muted-foreground">
                    Nenhum certificado emitido ainda.
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
}
