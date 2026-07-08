import { UserCog } from "lucide-react";
import { SelectNative } from "@/components/ui/select-native";
import { useHotspotProfiles } from "@/components/hotspot/useHotspotProfileQueries";
import { useAssignDeviceProfile } from "@/components/hotspot/useHotspotProfileMutations";

// Dropdown de perfil vinculado ao dispositivo - mesmo layout de
// OverviewItem (DeviceOverviewTab.tsx), so troca o valor estático por
// um <SelectNative> editável. Perfil "Padrão" (isDefault=true) sempre
// aparece na lista mesmo sem profileId explícito, ja que é o valor
// default aplicado pelo backend quando o dispositivo nunca foi
// vinculado a nenhum outro perfil.
export function DeviceProfileSelect({ mac, profileId }: { mac: string; profileId?: string }) {
  const profiles = useHotspotProfiles();
  const assignProfile = useAssignDeviceProfile(mac);
  const defaultProfile = profiles.data?.find((profile) => profile.isDefault);

  return (
    <div className="flex items-center gap-3 rounded-lg border border-border/60 bg-muted/30 px-3 py-2.5">
      <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md bg-primary/10 text-primary">
        <UserCog className="h-4 w-4" />
      </div>
      <div className="min-w-0 flex-1">
        <p className="text-xs text-muted-foreground">Perfil</p>
        <SelectNative
          className="mt-1"
          value={profileId || defaultProfile?.id || ""}
          disabled={assignProfile.isPending || !profiles.data}
          onChange={(event) => assignProfile.mutate(event.target.value)}
        >
          {(profiles.data ?? []).map((profile) => (
            <option key={profile.id} value={profile.id}>
              {profile.name}
            </option>
          ))}
        </SelectNative>
      </div>
    </div>
  );
}
