import { Badge } from "@/components/ui/badge";
import { EmptyState } from "@/components/bindnets/EmptyState";
import type { serviceRowsForNode } from "@/lib/mesh";

interface BindnetServicesTabProps {
  services: ReturnType<typeof serviceRowsForNode>;
}

export function BindnetServicesTab({ services }: BindnetServicesTabProps) {
  if (services.length === 0) {
    return <EmptyState label="Nenhum serviço anunciado por este nó." />;
  }

  return (
    <div className="divide-y rounded-md border">
      {services.map((service) => (
        <div key={`${service.name}:${service.via}`} className="grid gap-3 p-3 sm:grid-cols-[1fr_auto_auto] sm:items-center">
          <div className="min-w-0">
            <p className="truncate font-medium">{service.name}</p>
            <p className="truncate text-xs text-muted-foreground">{service.via}</p>
          </div>
          <span className="text-sm text-muted-foreground">{service.detail}</span>
          <Badge variant={service.state === "ok" || service.state === "local" ? "success" : "secondary"}>
            {service.state}
          </Badge>
        </div>
      ))}
    </div>
  );
}
