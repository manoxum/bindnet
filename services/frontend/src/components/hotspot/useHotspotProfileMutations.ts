import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { api, ApiError } from "@/lib/api";
import type { HotspotProfileRequest } from "@/components/hotspot/hotspot-profile-types";

function onProfileError(error: unknown) {
  toast.error(error instanceof ApiError ? error.message : "Falha ao salvar perfil");
}

export function useHotspotProfileMutations() {
  const queryClient = useQueryClient();
  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ["hotspot", "profiles"] });
    queryClient.invalidateQueries({ queryKey: ["hotspot", "clients"] });
  };

  const create = useMutation({
    mutationFn: (profile: HotspotProfileRequest) => api.post("/hotspot/profiles", profile),
    onSuccess: () => {
      toast.success("Perfil criado.");
      invalidate();
    },
    onError: onProfileError,
  });

  const update = useMutation({
    mutationFn: ({ id, profile }: { id: string; profile: HotspotProfileRequest }) =>
      api.patch(`/hotspot/profiles/${encodeURIComponent(id)}`, profile),
    onSuccess: () => {
      toast.success("Perfil salvo.");
      invalidate();
    },
    onError: onProfileError,
  });

  const remove = useMutation({
    mutationFn: (id: string) => api.del(`/hotspot/profiles/${encodeURIComponent(id)}`),
    onSuccess: () => {
      toast.success("Perfil removido.");
      invalidate();
    },
    onError: onProfileError,
  });

  return { create, update, remove };
}

export function useAssignDeviceProfile(mac: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (profileId: string) => api.patch(`/hotspot/devices/${encodeURIComponent(mac)}/profile`, { profileId }),
    onSuccess: () => {
      toast.success("Perfil do dispositivo atualizado.");
      queryClient.invalidateQueries({ queryKey: ["hotspot", "clients"] });
      queryClient.invalidateQueries({ queryKey: ["hotspot", "devices", mac] });
    },
    onError: onProfileError,
  });
}
