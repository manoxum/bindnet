import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { api, ApiError } from "@/lib/api";
import type {
  ContentCategory,
  ContentPlan,
  ContentPlanDetail,
  ContentPlanRequest,
  ContentRuleRequest,
} from "@/components/hotspot/hotspot-content-types";

const enc = encodeURIComponent;

export function useContentPlans() {
  return useQuery<ContentPlan[]>({
    queryKey: ["hotspot", "content", "plans"],
    queryFn: () => api.get<ContentPlan[]>("/hotspot/content-plans"),
  });
}

export function useContentPlan(id: string | null) {
  return useQuery<ContentPlanDetail>({
    queryKey: ["hotspot", "content", "plan", id],
    queryFn: () => api.get<ContentPlanDetail>(`/hotspot/content-plans/${enc(id!)}`),
    enabled: !!id,
  });
}

export function useContentCategories() {
  return useQuery<ContentCategory[]>({
    queryKey: ["hotspot", "content", "categories"],
    queryFn: () => api.get<ContentCategory[]>("/hotspot/content-categories"),
  });
}

function onContentError(error: unknown) {
  toast.error(error instanceof ApiError ? error.message : "Falha ao salvar conteúdo");
}

export function useHotspotContentMutations() {
  const queryClient = useQueryClient();
  const invalidate = () => queryClient.invalidateQueries({ queryKey: ["hotspot", "content"] });

  const createPlan = useMutation({
    mutationFn: (req: ContentPlanRequest) => api.post<ContentPlan>("/hotspot/content-plans", req),
    onSuccess: () => {
      toast.success("Plano criado.");
      invalidate();
    },
    onError: onContentError,
  });

  const updatePlan = useMutation({
    mutationFn: ({ id, req }: { id: string; req: ContentPlanRequest }) =>
      api.patch(`/hotspot/content-plans/${enc(id)}`, req),
    onSuccess: () => {
      toast.success("Plano salvo.");
      invalidate();
    },
    onError: onContentError,
  });

  const deletePlan = useMutation({
    mutationFn: (id: string) => api.del(`/hotspot/content-plans/${enc(id)}`),
    onSuccess: () => {
      toast.success("Plano removido.");
      invalidate();
    },
    onError: onContentError,
  });

  const addRule = useMutation({
    mutationFn: ({ planId, req }: { planId: string; req: ContentRuleRequest }) =>
      api.post(`/hotspot/content-plans/${enc(planId)}/rules`, req),
    onSuccess: () => {
      toast.success("Regra adicionada.");
      invalidate();
    },
    onError: onContentError,
  });

  const removeRule = useMutation({
    mutationFn: (ruleId: string) => api.del(`/hotspot/content-rules/${enc(ruleId)}`),
    onSuccess: () => {
      toast.success("Regra removida.");
      invalidate();
    },
    onError: onContentError,
  });

  const setCategoryEnabled = useMutation({
    mutationFn: ({ slug, enabled }: { slug: string; enabled: boolean }) =>
      api.patch(`/hotspot/content-categories/${enc(slug)}`, { enabled }),
    onSuccess: () => invalidate(),
    onError: onContentError,
  });

  const syncCategory = useMutation({
    mutationFn: (slug: string) => api.post(`/hotspot/content-categories/${enc(slug)}/sync`),
    onSuccess: () => toast.success("Sincronização iniciada (roda em segundo plano)."),
    onError: onContentError,
  });

  return { createPlan, updatePlan, deletePlan, addRule, removeRule, setCategoryEnabled, syncCategory };
}
