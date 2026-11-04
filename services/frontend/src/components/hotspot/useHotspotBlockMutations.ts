import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { api, ApiError } from "@/lib/api";

// "deauth" desconecta e impede reconexao; "traffic" conserva a conexao Wi-Fi.
export type HotspotBlockMode = "deauth" | "traffic";

export function useHotspotBlockMutations() {
  const queryClient = useQueryClient();
  const refreshDevices = () => Promise.all([
    queryClient.invalidateQueries({ queryKey: ["hotspot", "clients"] }),
    queryClient.invalidateQueries({ queryKey: ["hotspot", "blocklist"] }),
    queryClient.invalidateQueries({ queryKey: ["hotspot", "devices", "known"] }),
  ]);

  const block = useMutation({
    mutationFn: ({ mac, mode }: { mac: string; mode: HotspotBlockMode }) =>
      api.post("/hotspot/blocklist", { mac, mode }),
    onSuccess: async (_, { mode }) => {
      toast.success(mode === "traffic" ? "Tráfego do cliente cortado." : "Cliente bloqueado.");
      await refreshDevices();
    },
    onError: (error) => toast.error(error instanceof ApiError ? error.message : "Falha ao bloquear cliente"),
  });

  const unblock = useMutation({
    mutationFn: (mac: string) => api.del(`/hotspot/blocklist/${encodeURIComponent(mac)}`),
    onSuccess: async () => {
      toast.success("Cliente desbloqueado.");
      await refreshDevices();
    },
    onError: (error) => toast.error(error instanceof ApiError ? error.message : "Falha ao desbloquear cliente"),
  });

  return { block, unblock };
}
