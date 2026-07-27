import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { api, ApiError } from "@/lib/api";
import type { HotspotMessage, HotspotMessageCreateRequest } from "@/components/hotspot/messages/hotspot-message-types";

function onMessageError(error: unknown) {
  toast.error(error instanceof ApiError ? error.message : "Falha ao processar aviso");
}

export function useHotspotMessages() {
  return useQuery<HotspotMessage[]>({
    queryKey: ["hotspot", "messages"],
    queryFn: () => api.get<HotspotMessage[]>("/hotspot/messages"),
  });
}

export function useHotspotMessageMutations() {
  const queryClient = useQueryClient();
  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["hotspot", "messages"] });

  const create = useMutation({
    mutationFn: (request: HotspotMessageCreateRequest) => api.post<HotspotMessage>("/hotspot/messages", request),
    onSuccess: () => {
      toast.success("Aviso enviado aos dispositivos.");
      invalidate();
    },
    onError: onMessageError,
  });

  const remove = useMutation({
    mutationFn: (id: string) => api.del(`/hotspot/messages/${id}`),
    onSuccess: () => {
      toast.success("Aviso removido.");
      invalidate();
    },
    onError: onMessageError,
  });

  return { create, remove };
}
