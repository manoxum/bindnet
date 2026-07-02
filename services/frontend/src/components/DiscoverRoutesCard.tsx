import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Route, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { api, ApiError } from "@/lib/api";

interface DiscoverRoute {
  domain: string;
  owner: string;
  nextHop: string;
  distance: number;
  source: string;
  state: string;
  lastSeenAt: string;
}

export function DiscoverRoutesCard() {
  const queryClient = useQueryClient();

  const routes = useQuery<DiscoverRoute[]>({
    queryKey: ["dns", "routes"],
    queryFn: () => api.get<DiscoverRoute[]>("/dns/routes"),
    refetchInterval: 10000,
  });

  const forget = useMutation({
    mutationFn: (domain: string) => api.del(`/dns/routes/${domain}`),
    onSuccess: () => {
      toast.success("Rota removida.");
      queryClient.invalidateQueries({ queryKey: ["dns", "routes"] });
    },
    onError: (error) => toast.error(error instanceof ApiError ? error.message : "Falha ao remover rota"),
  });

  const rows = routes.data ?? [];

  return (
    <Card>
      <CardHeader className="flex flex-row items-start gap-3 space-y-0">
        <Route className="mt-1 h-4 w-4 text-muted-foreground" />
        <div>
          <CardTitle>Rotas da malha (discover mode)</CardTitle>
          <CardDescription>
            Domínios remotos aprendidos dos servidores vizinhos (DISCOVER_PEERS), com o próximo salto conhecido.
          </CardDescription>
        </div>
      </CardHeader>
      <CardContent>
        {rows.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            {routes.isLoading ? "Carregando..." : "Nenhuma rota remota conhecida ainda."}
          </p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Domínio</TableHead>
                <TableHead>Dono</TableHead>
                <TableHead>Próximo salto</TableHead>
                <TableHead>Distância</TableHead>
                <TableHead>Origem</TableHead>
                <TableHead>Estado</TableHead>
                <TableHead className="text-right">Ações</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((route) => (
                <TableRow key={route.domain}>
                  <TableCell className="font-medium">{route.domain}</TableCell>
                  <TableCell>{route.owner}</TableCell>
                  <TableCell>{route.nextHop}</TableCell>
                  <TableCell>{route.distance}</TableCell>
                  <TableCell className="text-muted-foreground">{route.source}</TableCell>
                  <TableCell>
                    <Badge variant={route.state === "ok" ? "success" : "secondary"}>{route.state}</Badge>
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      variant="destructive"
                      size="sm"
                      disabled={forget.isPending}
                      onClick={() => forget.mutate(route.domain)}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
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
