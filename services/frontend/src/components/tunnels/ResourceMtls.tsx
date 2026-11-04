import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { useTunnelApi, useTunnelAccount } from "./useTunnelAccount";
import { useTunnelMutation } from "./useTunnels";
import { ConfirmTunnelAction, type TunnelConfirmation } from "./ConfirmTunnelAction";
export function ResourceMtls({ resourceId }: { resourceId: number }) {
  const api = useTunnelApi();
  const account = useTunnelAccount();
  const [name, setName] = useState("");
  const [certificate, setCertificate] = useState("");
  const [confirmation, setConfirmation] = useState<TunnelConfirmation | null>(null);
  const certificates = useQuery({
    queryKey: ["tunnels", account, "mtls", resourceId],
    queryFn: () =>
      api.get<{ certificates: { mtlsCertificateId: number; name: string; fingerprint: string }[] }>(
        `/tunnels/resources/${resourceId}/mtls`,
      ),
    retry: false,
  });
  const add = useTunnelMutation(
    () => api.post(`/tunnels/resources/${resourceId}/mtls`, { name, certificate }),
    "CA pública cadastrada no Pangolin.",
  );
  const remove = useTunnelMutation(
    (id: number) => api.del(`/tunnels/resources/${resourceId}/mtls/${id}`),
    "CA removida.",
  );
  return (
    <section className="space-y-4 rounded-lg border p-4">
      <h3 className="font-medium">Certificados de cliente (mTLS)</h3>
      <p className="text-xs text-muted-foreground">
        Cadastre somente o certificado público da CA que assina os certificados dos clientes. Nunca cole uma
        chave privada.
      </p>
      {certificates.isError && (
        <p role="alert" className="text-sm text-destructive">
          {certificates.error.message}
        </p>
      )}
      {certificates.data?.certificates?.map((value) => (
        <div key={value.mtlsCertificateId} className="flex items-center justify-between gap-3">
          <span className="break-all text-sm">{value.name || `CA #${value.mtlsCertificateId}`}</span>
          <Button
            variant="outline"
            size="sm"
            disabled={remove.isPending}
            onClick={() =>
              setConfirmation({
                title: "Remover CA de confiança?",
                description:
                  "Clientes que dependem desta CA podem perder o acesso. Remover a última CA pode desativar mTLS no Pangolin.",
                action: () => remove.mutate(value.mtlsCertificateId),
              })
            }
          >
            Remover
          </Button>
        </div>
      ))}
      <form
        className="space-y-3"
        onSubmit={(event) => {
          event.preventDefault();
          add.mutate(undefined, {
            onSuccess: () => {
              setName("");
              setCertificate("");
            },
          });
        }}
      >
        <div className="space-y-2">
          <Label htmlFor="mtls-name">Nome da CA</Label>
          <Input id="mtls-name" value={name} onChange={(event) => setName(event.target.value)} />
        </div>
        <div className="space-y-2">
          <Label htmlFor="mtls-cert">Certificado público PEM</Label>
          <Textarea
            id="mtls-cert"
            rows={4}
            value={certificate}
            onChange={(event) => setCertificate(event.target.value)}
            placeholder="-----BEGIN CERTIFICATE-----"
          />
        </div>
        <Button variant="outline" disabled={add.isPending || !certificate.trim()}>
          Cadastrar CA pública
        </Button>
      </form>
      <ConfirmTunnelAction value={confirmation} close={() => setConfirmation(null)} />
    </section>
  );
}
