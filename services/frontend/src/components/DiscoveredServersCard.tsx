import { useQuery } from "@tanstack/react-query";
import { Server } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { api } from "@/lib/api";

interface DiscoveredServer {
  name: string;
  zone: string;
  source: string;
  kind: string;
  file?: string;
}

export function DiscoveredServersCard() {
  const servers = useQuery<DiscoveredServer[]>({
    queryKey: ["dns", "discovered-servers"],
    queryFn: () => api.get<DiscoveredServer[]>("/dns/discovered-servers"),
    refetchInterval: 10000,
  });

  const rows = servers.data ?? [];

  return (
    <Card>
      <CardHeader className="flex flex-row items-start gap-3 space-y-0">
        <Server className="mt-1 h-4 w-4 text-muted-foreground" />
        <div>
          <CardTitle>Serviços locais (nginx-ui)</CardTitle>
          <CardDescription>
            Domínios declarados no nginx-ui deste servidor, tratados como DNS local e anunciados aos vizinhos da
            malha de descoberta como pertencentes a este nó.
          </CardDescription>
        </div>
      </CardHeader>
      <CardContent>
        {rows.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            {servers.isLoading ? "Carregando..." : "Nenhum server_name encontrado no nginx-ui."}
          </p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Servidor</TableHead>
                <TableHead>Tipo</TableHead>
                <TableHead>Origem</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((server) => (
                <TableRow key={`${server.source}:${server.name}`}>
                  <TableCell className="font-medium">{server.name}</TableCell>
                  <TableCell>
                    <Badge variant="secondary">{server.kind}</Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground">{server.source}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}
