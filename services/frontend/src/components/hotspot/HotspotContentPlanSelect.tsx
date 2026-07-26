import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { api, ApiError } from "@/lib/api";
import { Label } from "@/components/ui/label";
import { SelectNative } from "@/components/ui/select-native";
import { useContentPlans } from "@/components/hotspot/useHotspotContent";

interface LinkResponse {
  planId: string | null;
}

// Seletor "Plano de conteúdo" reutilizável por perfil e dispositivo. O
// vínculo mora em endpoint próprio (PATCH .../content-plan), separado do
// submit do formulário do perfil/limites - por isso este componente
// carrega e salva sozinho, via as rotas de link.
export function HotspotContentPlanSelect({ scope, id }: { scope: "profile" | "device"; id: string }) {
  const base = scope === "profile" ? `/hotspot/profiles/${encodeURIComponent(id)}/content-plan` : `/hotspot/devices/${encodeURIComponent(id)}/content-plan`;
  const queryClient = useQueryClient();
  const plans = useContentPlans();

  const link = useQuery<LinkResponse>({
    queryKey: ["hotspot", "content", "link", scope, id],
    queryFn: () => api.get<LinkResponse>(base),
    enabled: !!id,
  });

  const setLink = useMutation({
    mutationFn: (planId: string | null) => api.patch(base, { planId }),
    onSuccess: () => {
      toast.success("Plano de conteúdo atualizado.");
      queryClient.invalidateQueries({ queryKey: ["hotspot", "content", "link", scope, id] });
    },
    onError: (error) => toast.error(error instanceof ApiError ? error.message : "Falha ao vincular plano"),
  });

  return (
    <div className="space-y-2">
      <Label htmlFor="contentPlan">Plano de conteúdo (bloqueio de sites/serviços)</Label>
      <SelectNative
        id="contentPlan"
        value={link.data?.planId ?? ""}
        disabled={setLink.isPending || link.isLoading}
        onChange={(e) => setLink.mutate(e.target.value === "" ? null : e.target.value)}
      >
        <option value="">Nenhum</option>
        {(plans.data ?? []).map((plan) => (
          <option key={plan.id} value={plan.id}>
            {plan.name}
          </option>
        ))}
      </SelectNative>
      <p className="text-xs text-muted-foreground">Salvo na hora, separado do restante do formulário.</p>
    </div>
  );
}
