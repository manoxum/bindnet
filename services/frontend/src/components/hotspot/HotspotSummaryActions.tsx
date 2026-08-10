import { Play, RefreshCw, Settings2, Square } from "lucide-react";
import { Button } from "@/components/ui/button";

interface HotspotSummaryActionsProps {
  running: boolean;
  startPending: boolean;
  stopPending: boolean;
  recoverPending: boolean;
  onStart: () => void;
  onStop: () => void;
  onRecover: () => void;
  onEdit: () => void;
}

// Barra de ações do card de resumo: ciclo de vida do hotspot
// (iniciar/parar/recuperar) e atalho para o formulário. Extraída do
// HotspotSummaryCard para manter aquele arquivo dentro do limite de
// ~200 linhas (ver CLAUDE.md). O interruptor de arranque automático
// mora na aba "Serviço" da configuração (HotspotServiceTab), não aqui.
export function HotspotSummaryActions({
  running,
  startPending,
  stopPending,
  recoverPending,
  onStart,
  onStop,
  onRecover,
  onEdit,
}: HotspotSummaryActionsProps) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      <Button size="sm" onClick={onStart} disabled={running || startPending || recoverPending}>
        <Play className="h-4 w-4" />
        Iniciar
      </Button>
      <Button size="sm" variant="destructive" onClick={onStop} disabled={!running || stopPending || recoverPending}>
        <Square className="h-4 w-4" />
        Parar
      </Button>
      <Button size="sm" variant="secondary" onClick={onRecover} disabled={recoverPending || startPending || stopPending}>
        <RefreshCw className={recoverPending ? "h-4 w-4 animate-spin" : "h-4 w-4"} />
        Recuperar Wi-Fi
      </Button>
      <Button variant="outline" size="sm" onClick={onEdit}>
        <Settings2 className="h-4 w-4" />
        Alterar configuração
      </Button>
    </div>
  );
}
