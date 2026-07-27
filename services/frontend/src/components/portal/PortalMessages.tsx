import { AlertTriangle, Info } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useMarkMessageRead, usePortalMessages } from "@/components/portal/usePortalMessages";

// Banner de avisos no topo do portal - avisos urgentes destacados. Cada
// aviso some ao tocar em "Ok" (marca lido no backend). Responsivo por
// padrao: empilha e quebra texto longo em qualquer largura.
export function PortalMessages() {
  const messages = usePortalMessages();
  const markRead = useMarkMessageRead();

  if (!messages.data?.length) return null;

  return (
    <div className="space-y-2">
      {messages.data.map((message) => (
        <div
          key={message.id}
          className={`rounded-lg border p-3 ${
            message.urgent ? "border-destructive/40 bg-destructive/10" : "bg-muted/30"
          }`}
        >
          <div className="flex items-start gap-2">
            {message.urgent ? (
              <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-destructive" />
            ) : (
              <Info className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground" />
            )}
            <div className="min-w-0 flex-1">
              {message.title && <p className="text-sm font-semibold">{message.title}</p>}
              <p className="whitespace-pre-wrap break-words text-sm text-muted-foreground">{message.body}</p>
            </div>
          </div>
          <div className="mt-2 flex justify-end">
            <Button
              size="sm"
              variant="outline"
              onClick={() => markRead.mutate(message.id)}
              disabled={markRead.isPending}
            >
              Ok
            </Button>
          </div>
        </div>
      ))}
    </div>
  );
}
