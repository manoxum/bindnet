import { Trash2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/bindnets/EmptyState";
import type { DiscoverRoute } from "@/lib/mesh";

interface BindnetRoutesTabProps {
  routes: DiscoverRoute[];
  forgetPending: boolean;
  onForget: (domain: string) => void;
}

export function BindnetRoutesTab({ routes, forgetPending, onForget }: BindnetRoutesTabProps) {
  if (routes.length === 0) {
    return <EmptyState label="Nenhuma rota remota passando por este contexto." />;
  }

  return (
    <div className="grid gap-2 md:grid-cols-2">
      {routes.slice(0, 10).map((route) => (
        <div key={`${route.domain}:${route.owner}`} className="rounded-md border px-3 py-3">
          <div className="flex items-center justify-between gap-3">
            <p className="truncate font-medium">{route.domain}</p>
            <div className="flex items-center gap-2">
              <Badge variant={route.state === "ok" ? "success" : "secondary"}>{route.state}</Badge>
              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6"
                disabled={forgetPending}
                onClick={() => onForget(route.domain)}
                aria-label={`Remover rota ${route.domain}`}
              >
                <Trash2 className="h-3.5 w-3.5" />
              </Button>
            </div>
          </div>
          <div className="mt-2 flex items-center justify-between gap-3 text-xs text-muted-foreground">
            <span className="truncate">dono: {route.owner}</span>
            <span>{route.distance} salto{route.distance === 1 ? "" : "s"}</span>
          </div>
        </div>
      ))}
    </div>
  );
}
