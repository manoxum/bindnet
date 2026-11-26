import { Trash2, KeyRound } from "lucide-react";
import { Button } from "@/components/ui/button";
export type IntegrationToken = {
  id: string;
  name: string;
  prefix: string;
  expiresAt: string;
  revokedAt: string | null;
  lastUsedAt: string | null;
  createdAt: string;
};
export function TokenList({
  items,
  busy,
  onRevoke,
}: {
  items: IntegrationToken[];
  busy: boolean;
  onRevoke: (item: IntegrationToken) => void;
}) {
  if (!items.length)
    return (
      <p className="rounded-xl border border-dashed p-8 text-center text-sm text-muted-foreground">
        Nenhum token emitido.
      </p>
    );
  return (
    <div className="space-y-3">
      {items.map((item) => (
        <div
          key={item.id}
          className="flex items-start gap-3 rounded-xl border p-4"
        >
          <KeyRound className="mt-1 shrink-0 text-muted-foreground" size={17} />
          <div className="min-w-0 flex-1">
            <p className="font-semibold">
              {item.name}
              <span className="ml-2 text-xs font-normal text-muted-foreground">
                {item.revokedAt
                  ? "Revogado"
                  : new Date(item.expiresAt).getTime() <= Date.now()
                    ? "Expirado"
                    : "Ativo"}
              </span>
            </p>
            <p className="mt-1 font-mono text-xs text-muted-foreground">
              {item.prefix}…
            </p>
            <p className="mt-1 text-xs text-muted-foreground">
              Validade: {new Date(item.expiresAt).toLocaleDateString()} · Último
              uso:{" "}
              {item.lastUsedAt
                ? new Date(item.lastUsedAt).toLocaleString()
                : "Nunca"}
            </p>
          </div>
          {!item.revokedAt && (
            <Button
              variant="ghost"
              size="icon"
              disabled={busy}
              aria-label={`Revogar ${item.name}`}
              onClick={() => onRevoke(item)}
            >
              <Trash2 size={16} />
            </Button>
          )}
        </div>
      ))}
    </div>
  );
}
