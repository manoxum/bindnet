import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { Download, HardDriveDownload, Radar, ShieldCheck } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { api, ApiError } from "@/lib/api";
import { type BindnetNode, peerHost } from "@/lib/mesh";

const defaultBackendPort = "8090";

function downloadTextFile(filename: string, contents: string) {
  const blob = new Blob([contents], { type: "text/plain;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  link.click();
  URL.revokeObjectURL(url);
}

export function RemoteCaCard({ node }: { node: BindnetNode }) {
  const [port, setPort] = useState(defaultBackendPort);
  const [certificatePem, setCertificatePem] = useState<string | null>(null);

  const fetchCa = useMutation({
    mutationFn: async () => {
      const url = `http://${peerHost(node.address)}:${port}/api/mesh/ca`;
      const response = await fetch(url);
      if (!response.ok) throw new Error(`Servidor respondeu ${response.status}`);
      return response.text();
    },
    onSuccess: (pem) => setCertificatePem(pem),
    onError: () =>
      toast.error("Não foi possível buscar a CA deste servidor. Confira o endereço/porta e se ele está acessível nesta rede."),
  });

  const installLocal = useMutation({
    mutationFn: () =>
      api.post<{ path: string }>("/certificates/ca/install-local", { certificatePem }),
    onSuccess: (result) => toast.success(`CA de ${node.name} instalada em ${result.path}`),
    onError: (error) =>
      toast.error(error instanceof ApiError ? error.message : "Falha ao instalar CA deste servidor"),
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle>Autoridade certificadora de {node.name}</CardTitle>
        <CardDescription>
          Busca a CA pública deste servidor remoto (sem autenticação, é um certificado público) e permite baixar ou
          instalar neste computador.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
          <div className="flex-1 space-y-2">
            <Label htmlFor="remote-ca-port">Porta do painel ({peerHost(node.address)})</Label>
            <Input id="remote-ca-port" value={port} onChange={(event) => setPort(event.target.value)} />
          </div>
          <Button variant="outline" onClick={() => fetchCa.mutate()} disabled={fetchCa.isPending}>
            <Radar className="h-4 w-4" />
            {fetchCa.isPending ? "Buscando..." : "Buscar CA"}
          </Button>
        </div>

        {certificatePem && (
          <div className="flex flex-col gap-3 rounded-md border bg-muted/30 p-4 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex gap-3">
              <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-md border bg-background">
                <ShieldCheck className="h-5 w-5 text-primary" />
              </div>
              <p className="text-sm text-muted-foreground">CA encontrada. Baixe ou instale neste computador.</p>
            </div>
            <div className="flex gap-2">
              <Button
                variant="outline"
                onClick={() => downloadTextFile(`bindnet-ca-${peerHost(node.address)}.crt`, certificatePem)}
              >
                <Download className="h-4 w-4" />
                Baixar
              </Button>
              <Button onClick={() => installLocal.mutate()} disabled={installLocal.isPending}>
                <HardDriveDownload className="h-4 w-4" />
                {installLocal.isPending ? "Instalando..." : "Instalar localmente"}
              </Button>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
