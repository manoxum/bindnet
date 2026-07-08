import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { HotspotProfile } from "@/components/hotspot/hotspot-profile-types";

export function useHotspotProfiles() {
  return useQuery<HotspotProfile[]>({
    queryKey: ["hotspot", "profiles"],
    queryFn: () => api.get<HotspotProfile[]>("/hotspot/profiles"),
  });
}
