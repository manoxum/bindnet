import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Loader2 } from "lucide-react";
import { useTunnelApi, useTunnelAccount } from "./useTunnelAccount";
interface Progress {
  state: string;
  stage: string;
  message: string;
  startedAt: string;
  siteId?: number;
  resourceId?: number;
}
export function PublicationProgress({ active, startedAt }: { active: boolean; startedAt: number }) {
  const api = useTunnelApi();
  const account = useTunnelAccount();
  const [now, setNow] = useState(Date.now());
  useEffect(() => {
    if (!active) return;
    setNow(Date.now());
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, [active]);
  const progress = useQuery({
    queryKey: ["tunnels", account, "publication-status"],
    queryFn: () => api.get<Progress>("/tunnels/publication-status"),
    enabled: active,
    refetchInterval: active ? 2000 : false,
    retry: false,
    gcTime: 0,
  });
  if (!active) return null;
  const current =
    progress.data && Date.parse(progress.data.startedAt) >= startedAt - 1500 ? progress.data : undefined;
  return (
    <div role="status" aria-live="polite" className="space-y-2 rounded-lg border bg-muted/30 p-3 text-sm">
      <p className="flex items-center gap-2">
        <Loader2 className="h-4 w-4 shrink-0 animate-spin" />
        {current?.message ?? "Aguardando confirmação do Pangolin…"}
      </p>
      <p className="text-xs text-muted-foreground">
        Tempo decorrido: {Math.max(0, Math.floor((now - startedAt) / 1000))}s · prazo máximo de espera: 195s.
      </p>
      {current?.siteId && (
        <p className="text-xs text-muted-foreground">
          Site #{current.siteId}
          {current.resourceId ? ` · recurso #${current.resourceId}` : ""}
        </p>
      )}
      {progress.isError && (
        <p className="text-xs text-muted-foreground">
          Não foi possível consultar o progresso neste momento. A resposta da publicação continua em
          andamento.
        </p>
      )}
    </div>
  );
}
