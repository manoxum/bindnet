import { useState } from "react";
import { Check, Copy } from "lucide-react";
import { Button } from "@/components/ui/button";

interface InstallCaQuickCommandPanelProps {
  command: string;
  note: string;
}

// Opção 2: um comando único para colar no terminal - baixa e instala a
// CA automaticamente, sem precisar baixar nada pelo navegador antes.
export function InstallCaQuickCommandPanel({ command, note }: InstallCaQuickCommandPanelProps) {
  const [copied, setCopied] = useState(false);

  async function copy() {
    await navigator.clipboard.writeText(command);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  return (
    <div className="space-y-3 text-sm">
      <p className="font-medium">Opção 2 · Comando de instalação rápida</p>
      <p className="text-muted-foreground">{note}</p>
      <div className="flex items-start gap-2">
        <pre className="flex-1 overflow-x-auto rounded-md bg-muted p-3 text-xs">{command}</pre>
        <Button type="button" variant="outline" size="sm" onClick={copy}>
          {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
          {copied ? "Copiado" : "Copiar"}
        </Button>
      </div>
    </div>
  );
}
