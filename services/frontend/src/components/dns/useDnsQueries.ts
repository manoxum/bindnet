import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";

export interface DnsRecord {
  hostname: string;
  address: string;
  configuredAddress: string | null;
  source: "manual" | "dynamic";
  originTld: string | null;
  loopbackOffset: number;
  createdAt: string;
}

export interface LocalTldConfig {
  tld: string;
  mode: "automatic" | "static";
  address?: string;
}

export function useDnsQueries() {
  const config = useQuery<Record<string, string>>({
    queryKey: ["dns", "config"],
    queryFn: () => api.get<Record<string, string>>("/dns/config"),
  });

  const records = useQuery<DnsRecord[]>({
    queryKey: ["dns", "records"],
    queryFn: () => api.get<DnsRecord[]>("/dns/records"),
  });

  const localTlds = useQuery<LocalTldConfig[]>({
    queryKey: ["dns", "local-tlds"],
    queryFn: () => api.get<LocalTldConfig[]>("/dns/local-tlds"),
  });

  return { config, records, localTlds };
}
