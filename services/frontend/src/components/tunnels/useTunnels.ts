import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { useTunnelApi, useTunnelAccount } from "./useTunnelAccount";
import type { TunnelConfig, TunnelDomain, TunnelResource, TunnelSite } from "./types";

export function useTunnelMutation<T = void>(action: (value: T) => Promise<unknown>, message: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: action,
    onSuccess: () => toast.success(message),
    onError: (error: Error) => toast.error(error.message),
    // Recarrega também após falha parcial de provisionamento.
    onSettled: () => {
      void client.invalidateQueries({ queryKey: ["tunnels"] });
    },
  });
}

export function useTunnels() {
  const api = useTunnelApi();
  const account = useTunnelAccount();
  const config = useQuery({
    queryKey: ["tunnels", account, "config"],
    queryFn: () => api.get<TunnelConfig>("/tunnels/config"),
  });
  const enabled = config.data?.configured === true;
  const sites = useQuery({
    queryKey: ["tunnels", account, "sites"],
    queryFn: () => api.get<TunnelSite[]>("/tunnels/sites"),
    enabled,
    refetchInterval: 15000,
    retry: false,
  });
  const resources = useQuery({
    queryKey: ["tunnels", account, "resources"],
    queryFn: () => api.get<TunnelResource[]>("/tunnels/resources"),
    enabled,
    retry: false,
  });
  const domains = useQuery({
    queryKey: ["tunnels", account, "domains"],
    queryFn: () => api.get<TunnelDomain[]>("/tunnels/domains"),
    enabled,
    retry: false,
  });
  return { config, sites, resources, domains };
}
