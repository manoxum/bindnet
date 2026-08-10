import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { api, ApiError } from "@/lib/api";

interface HotspotAutostart {
  enabled: boolean;
}

// Interruptor "iniciar automaticamente no arranque". Rota própria
// (GET/PATCH /api/hotspot/autostart) e não o formulário de configuração:
// salvar a configuração dispara POST /hotspot/apply, que REINICIA o
// hotspot e derruba todos os clientes — gravar uma preferência de
// arranque nunca deve fazer isso. Ver RegisterHotspotAutostartRoutes em
// services/backend/internal/hotspot/hotspot_autostart_routes.go.
export function useHotspotAutostart() {
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: ["hotspot", "autostart"],
    queryFn: () => api.get<HotspotAutostart>("/hotspot/autostart"),
  });

  const update = useMutation({
    mutationFn: (enabled: boolean) => api.patch("/hotspot/autostart", { enabled }),
    onSuccess: (_data, enabled) => {
      toast.success(
        enabled
          ? "O hotspot passará a subir sozinho quando o sistema arrancar."
          : "O hotspot deixará de subir sozinho no arranque.",
      );
      queryClient.invalidateQueries({ queryKey: ["hotspot", "autostart"] });
    },
    onError: (error) =>
      toast.error(error instanceof ApiError ? error.message : "Falha ao salvar o arranque automático"),
  });

  return {
    enabled: query.data?.enabled ?? false,
    // Enquanto a leitura inicial não chega, desabilita o controle em vez
    // de mostrar "não" como se fosse a resposta do servidor.
    loading: query.isPending,
    pending: update.isPending,
    setEnabled: (enabled: boolean) => update.mutate(enabled),
  };
}
