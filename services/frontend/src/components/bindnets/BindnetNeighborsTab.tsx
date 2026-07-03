import { Link } from "react-router-dom";
import { CircleDot } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { EmptyState } from "@/components/bindnets/EmptyState";
import { cn } from "@/lib/utils";
import { nodeLabel, nodePath, type neighborRows } from "@/lib/mesh";

interface BindnetNeighborsTabProps {
  neighbors: ReturnType<typeof neighborRows>;
}

export function BindnetNeighborsTab({ neighbors }: BindnetNeighborsTabProps) {
  if (neighbors.length === 0) {
    return <EmptyState label="Nenhum vizinho conhecido para este nó." />;
  }

  return (
    <div className="space-y-2">
      {neighbors.map((neighbor) => (
        <Link
          key={neighbor.id}
          to={nodePath(neighbor.id)}
          className="flex items-center justify-between rounded-md border px-3 py-2 transition hover:bg-accent"
        >
          <div className="flex min-w-0 items-center gap-2">
            <CircleDot className={cn("h-3.5 w-3.5", neighbor.kind === "direct" ? "text-sky-500" : "text-amber-500")} />
            <span className="truncate text-sm font-medium">{neighbor.name}</span>
          </div>
          <Badge variant="outline">{nodeLabel(neighbor.kind)}</Badge>
        </Link>
      ))}
    </div>
  );
}
