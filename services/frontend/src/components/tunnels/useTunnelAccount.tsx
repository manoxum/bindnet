import { createContext, useContext, useMemo, type ReactNode } from "react";
import { api } from "@/lib/api";
const TunnelAccount = createContext("");
export function TunnelAccountProvider({ accountId, children }: { accountId: string; children: ReactNode }) {
  return <TunnelAccount.Provider value={accountId}>{children}</TunnelAccount.Provider>;
}
export function useTunnelAccount() {
  return useContext(TunnelAccount);
}
export function useTunnelApi() {
  const account = useTunnelAccount();
  return useMemo(() => {
    const path = (value: string) =>
      `/tunnels/accounts/${encodeURIComponent(account)}${value.replace(/^\/tunnels/, "")}`;
    return {
      get: <T,>(value: string) => api.get<T>(path(value), { timeoutMs: 25000 }),
      post: <T,>(value: string, body?: unknown) =>
        api.post<T>(path(value), body, { timeoutMs: value === "/tunnels/sites" ? 195000 : 90000 }),
      put: <T,>(value: string, body?: unknown) => api.put<T>(path(value), body),
      patch: <T,>(value: string, body?: unknown) => api.patch<T>(path(value), body),
      del: <T,>(value: string) => api.del<T>(path(value)),
    };
  }, [account]);
}
