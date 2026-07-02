import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { api, ApiError } from "@/lib/api";

// Compartilha a mesma queryKey de Dns.tsx ("dns","config") de proposito:
// os dois campos (DOMAINS/DISCOVER_*) fazem parte da mesma secao "dns"
// do .env, so o card de edicao e separado (um assunto por componente).
export function DiscoverMeshConfigCard() {
  const queryClient = useQueryClient();
  const [domains, setDomains] = useState("");
  const [nodeName, setNodeName] = useState("");
  const [peers, setPeers] = useState("");

  const config = useQuery<Record<string, string>>({
    queryKey: ["dns", "config"],
    queryFn: () => api.get<Record<string, string>>("/dns/config"),
  });

  useEffect(() => {
    if (config.data) {
      setDomains(config.data.DOMAINS ?? "");
      setNodeName(config.data.DISCOVER_NODE_NAME ?? "");
      setPeers(config.data.DISCOVER_PEERS ?? "");
    }
  }, [config.data]);

  const save = useMutation({
    mutationFn: () =>
      api.patch("/dns/config", { DOMAINS: domains, DISCOVER_NODE_NAME: nodeName, DISCOVER_PEERS: peers }),
    onSuccess: () => {
      toast.success("Malha de descoberta salva. Clique em 'Aplicar' para reiniciar o DNS com os novos valores.");
      queryClient.invalidateQueries({ queryKey: ["dns", "config"] });
    },
    onError: (error) => toast.error(error instanceof ApiError ? error.message : "Falha ao salvar"),
  });

  const apply = useMutation({
    mutationFn: () => api.post("/dns/apply"),
    onSuccess: () => toast.success("DNS recriado com a configuração atual."),
    onError: (error) => toast.error(error instanceof ApiError ? error.message : "Falha ao aplicar"),
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle>Malha de descoberta (discover mode)</CardTitle>
        <CardDescription>
          Domínios que participam do roteamento por próximo salto entre servidores Bindnet. Nomes locais
          (anunciados pelo nginx-ui) resolvem normalmente; nomes de outro servidor são encaminhados para o
          próximo salto conhecido.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor="domains">DOMAINS (zonas da malha)</Label>
            <Input id="domains" placeholder="ex.: dev" value={domains} onChange={(e) => setDomains(e.target.value)} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="nodeName">Nome deste servidor</Label>
            <Input
              id="nodeName"
              placeholder="ex.: servidor-a"
              value={nodeName}
              onChange={(e) => setNodeName(e.target.value)}
            />
          </div>
          <div className="space-y-2 sm:col-span-2">
            <Label htmlFor="peers">Servidores vizinhos (DISCOVER_PEERS)</Label>
            <Input
              id="peers"
              placeholder="ex.: 10.0.0.2:8531, 10.0.0.3:8531"
              value={peers}
              onChange={(e) => setPeers(e.target.value)}
            />
          </div>
        </div>
        <div className="flex gap-2">
          <Button onClick={() => save.mutate()} disabled={save.isPending}>
            Salvar
          </Button>
          <Button variant="outline" onClick={() => apply.mutate()} disabled={apply.isPending}>
            Aplicar e reiniciar
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
