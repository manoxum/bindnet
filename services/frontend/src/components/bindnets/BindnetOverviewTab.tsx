import { Clock, Fingerprint, Globe2, Network, Radar, Tag, type LucideIcon } from "lucide-react";
import { formatSeen, nodeLabel, type BindnetNode, type metricCards } from "@/lib/mesh";

interface BindnetOverviewTabProps {
  node: BindnetNode;
  metrics: ReturnType<typeof metricCards>;
}

function OverviewItem({ icon: Icon, label, value }: { icon: LucideIcon; label: string; value: string }) {
  return (
    <div className="flex items-center gap-3 rounded-lg border border-border/60 bg-muted/30 px-3 py-2.5">
      <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-primary/10 text-primary">
        <Icon className="h-4 w-4" />
      </div>
      <div className="min-w-0">
        <p className="text-xs text-muted-foreground">{label}</p>
        <p className="truncate text-sm font-semibold">{value}</p>
      </div>
    </div>
  );
}

export function BindnetOverviewTab({ node, metrics }: BindnetOverviewTabProps) {
  const domains = node.domains?.length ? node.domains.join(", ") : "sem domínio anunciado";
  const fingerprint = node.fingerprint || "ainda não informado";
  const endpoint = node.port ? `${node.host ?? node.address}:${node.port}` : node.host || node.address;

  return (
    <div className="space-y-4">
      <div className="grid gap-3 sm:grid-cols-3">
        {metrics.map(({ label, value, icon: Icon }) => (
          <div key={label} className="rounded-lg border bg-card px-4 py-3 shadow-sm">
            <Icon className="mb-2 h-4 w-4 text-muted-foreground" />
            <div className="text-xl font-semibold leading-none">{value}</div>
            <div className="mt-1 text-xs text-muted-foreground">{label}</div>
          </div>
        ))}
      </div>

      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <OverviewItem icon={Tag} label="Tipo" value={nodeLabel(node.kind)} />
        <OverviewItem icon={Radar} label="Origem" value={node.source} />
        <OverviewItem icon={Clock} label="Última leitura" value={formatSeen(node.lastSeenAt)} />
        <OverviewItem icon={Network} label="IP / porta" value={endpoint} />
        <OverviewItem icon={Fingerprint} label="Fingerprint" value={fingerprint} />
        <OverviewItem icon={Globe2} label="Domínios" value={domains} />
      </div>
    </div>
  );
}
