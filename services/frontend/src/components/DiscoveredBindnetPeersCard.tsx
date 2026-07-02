import { useQuery } from "@tanstack/react-query";
import { Server } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { api } from "@/lib/api";

interface DiscoveredPeer {
  address: string;
  nodeName: string;
  source: string;
  lastSeenAt: string;
}

// Servidores Bindnet encontrados por multicast na mesma rede local -
// diferente de DiscoveredServersCard (services locais anunciados pelo
// nginx-ui deste no). Uma rede diferente (ex.: outro site remoto)
// nunca aparece aqui - so via configuração manual em DISCOVER_PEERS.
export function DiscoveredBindnetPeersCard() {
  const peers = useQuery<DiscoveredPeer[]>({
    queryKey: ["dns", "peers"],
    queryFn: () => api.get<DiscoveredPeer[]>("/dns/peers"),
    refetchInterval: 10000,
  });

  const rows = peers.data ?? [];

  return (
    <Card>
      <CardHeader className="flex flex-row items-start gap-3 space-y-0">
        <Server className="mt-1 h-4 w-4 text-muted-foreground" />
        <div>
          <CardTitle>Servidores Bindnet na rede</CardTitle>
          <CardDescription>
            Outros servidores Bindnet encontrados automaticamente na mesma rede local (multicast). Servidores em
            outra rede não aparecem aqui - configure-os manualmente em DISCOVER_PEERS, na malha de descoberta acima.
          </CardDescription>
        </div>
      </CardHeader>
      <CardContent>
        {rows.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            {peers.isLoading ? "Carregando..." : "Nenhum servidor Bindnet encontrado na rede local ainda."}
          </p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Nome</TableHead>
                <TableHead>Endereço</TableHead>
                <TableHead>Origem</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((peer) => (
                <TableRow key={peer.address}>
                  <TableCell className="font-medium">{peer.nodeName}</TableCell>
                  <TableCell>{peer.address}</TableCell>
                  <TableCell>
                    <Badge variant="secondary">{peer.source}</Badge>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}
