import { useState } from "react";
import { Copy } from "lucide-react";
import { api } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
export function TokenIssueDialog({
  onClose,
  onIssued,
}: {
  onClose: () => void;
  onIssued: () => Promise<void>;
}) {
  const [name, setName] = useState("");
  const [days, setDays] = useState(90);
  const [token, setToken] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [copied, setCopied] = useState(false);
  async function issue(event: React.FormEvent) {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      const result = await api.post<{ token: string }>("/integrations/tokens", {
        name,
        validityDays: days,
      });
      setToken(result.token);
      await onIssued();
    } catch (e) {
      setError(
        e instanceof Error ? e.message : "Não foi possível emitir o token.",
      );
    } finally {
      setBusy(false);
    }
  }
  return (
    <Dialog
      open
      onOpenChange={(v) => {
        if (!v && !busy) onClose();
      }}
    >
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>
            {token ? "Token emitido" : "Emitir API Token"}
          </DialogTitle>
          <DialogDescription>
            {token
              ? "Copie agora. Este token não será mostrado novamente."
              : "Identifique a aplicação e escolha a validade do acesso."}
          </DialogDescription>
        </DialogHeader>
        {token ? (
          <div className="space-y-3">
            <code className="block break-all rounded-xl border bg-muted p-4 text-xs">
              {token}
            </code>
            <Button
              variant="outline"
              onClick={() =>
                void navigator.clipboard
                  .writeText(token)
                  .then(() => setCopied(true))
                  .catch(() =>
                    setError("Selecione o token e copie manualmente."),
                  )
              }
            >
              <Copy size={15} className="mr-2" />
              {copied ? "Copiado" : "Copiar token"}
            </Button>
          </div>
        ) : (
          <form onSubmit={(e) => void issue(e)} className="space-y-4">
            <div>
              <Label htmlFor="token-name">Nome da aplicação</Label>
              <Input
                id="token-name"
                required
                maxLength={100}
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </div>
            <div>
              <Label htmlFor="token-days">Validade em dias</Label>
              <Input
                id="token-days"
                type="number"
                min={1}
                max={365}
                required
                value={days}
                onChange={(e) => setDays(Number(e.target.value))}
              />
            </div>
            <Button className="w-full" disabled={busy}>
              Emitir token
            </Button>
          </form>
        )}
        {error && (
          <p role="alert" className="text-sm text-destructive">
            {error}
          </p>
        )}
      </DialogContent>
    </Dialog>
  );
}
