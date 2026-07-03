import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { api, ApiError } from "@/lib/api";
import type { ConfigForm } from "@/components/hotspot/hotspot-schema";

interface UseHotspotMutationsOptions {
  onSaveSuccess: () => void;
  onRecoverSuccess: () => void;
}

export function useHotspotMutations({ onSaveSuccess, onRecoverSuccess }: UseHotspotMutationsOptions) {
  const queryClient = useQueryClient();

  const save = useMutation({
    mutationFn: (data: ConfigForm) => api.patch("/hotspot/config", data),
    onSuccess: () => {
      toast.success("Configuração salva. Clique em 'Aplicar' para reiniciar o hotspot com os novos valores.");
      queryClient.invalidateQueries({ queryKey: ["hotspot", "config"] });
      onSaveSuccess();
    },
    onError: (error) => toast.error(error instanceof ApiError ? error.message : "Falha ao salvar"),
  });

  const apply = useMutation({
    mutationFn: () => api.post("/hotspot/apply"),
    onSuccess: () => {
      toast.success("Hotspot recriado com a configuração atual.");
      queryClient.invalidateQueries({ queryKey: ["hotspot"] });
    },
    onError: (error) => toast.error(error instanceof ApiError ? error.message : "Falha ao aplicar"),
  });

  const start = useMutation({
    mutationFn: () => api.post("/hotspot/start"),
    onSuccess: () => {
      toast.success("Hotspot iniciado.");
      queryClient.invalidateQueries({ queryKey: ["hotspot"] });
    },
    onError: (error) => toast.error(error instanceof ApiError ? error.message : "Falha ao iniciar"),
  });

  const stop = useMutation({
    mutationFn: () => api.post("/hotspot/stop"),
    onSuccess: () => {
      toast.success("Hotspot parado.");
      queryClient.invalidateQueries({ queryKey: ["hotspot"] });
    },
    onError: (error) => toast.error(error instanceof ApiError ? error.message : "Falha ao parar"),
  });

  const recoverWifi = useMutation({
    mutationFn: () => api.post("/hotspot/recover-wifi"),
    onSuccess: () => {
      toast.success("Adaptador Wi-Fi recuperado.");
      queryClient.invalidateQueries({ queryKey: ["hotspot"] });
      onRecoverSuccess();
    },
    onError: (error) => toast.error(error instanceof ApiError ? error.message : "Falha ao recuperar Wi-Fi"),
  });

  const identify = useMutation({
    mutationFn: (mac: string) => api.post(`/hotspot/clients/${encodeURIComponent(mac)}/identify`),
    onSuccess: () => {
      toast.success("Cliente identificado.");
      queryClient.invalidateQueries({ queryKey: ["hotspot", "clients"] });
    },
    onError: (error) => toast.error(error instanceof ApiError ? error.message : "Falha ao identificar cliente"),
  });

  const block = useMutation({
    mutationFn: (mac: string) => api.post("/hotspot/blocklist", { mac }),
    onSuccess: () => {
      toast.success("Cliente bloqueado.");
      queryClient.invalidateQueries({ queryKey: ["hotspot", "clients"] });
      queryClient.invalidateQueries({ queryKey: ["hotspot", "blocklist"] });
    },
    onError: (error) => toast.error(error instanceof ApiError ? error.message : "Falha ao bloquear cliente"),
  });

  const unblock = useMutation({
    mutationFn: (mac: string) => api.del(`/hotspot/blocklist/${encodeURIComponent(mac)}`),
    onSuccess: () => {
      toast.success("Cliente desbloqueado.");
      queryClient.invalidateQueries({ queryKey: ["hotspot", "clients"] });
      queryClient.invalidateQueries({ queryKey: ["hotspot", "blocklist"] });
    },
    onError: (error) => toast.error(error instanceof ApiError ? error.message : "Falha ao desbloquear cliente"),
  });

  return { save, apply, start, stop, recoverWifi, identify, block, unblock };
}
