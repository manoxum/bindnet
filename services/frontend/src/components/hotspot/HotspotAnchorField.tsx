import { useQuery } from "@tanstack/react-query";
import type { UseFormRegister, UseFormSetValue, UseFormWatch } from "react-hook-form";
import { api } from "@/lib/api";
import { Label } from "@/components/ui/label";
import { SelectNative } from "@/components/ui/select-native";
import type { ConfigForm } from "@/components/hotspot/hotspot-schema";

interface WifiNetwork {
  ssid: string;
  channel: number;
  freqMhz: number;
  signal: number;
}

interface HotspotAnchorFieldProps {
  register: UseFormRegister<ConfigForm>;
  setValue: UseFormSetValue<ConfigForm>;
  watch: UseFormWatch<ConfigForm>;
}

// Seletor da "rede âncora": a rede em cujo canal o hotspot deve subir.
// Ao escolher, grava também o canal (WIFI_ANCHOR_CHANNEL) — é ele que o
// hotspot usa, inclusive quando a rede não está no ar no momento de
// arrancar. Ver anchor.sh em services/worker/hotspot.
export function HotspotAnchorField({ register, setValue, watch }: HotspotAnchorFieldProps) {
  // Lê o cache do NetworkManager no worker; nunca força varredura, que
  // com o AP no ar interromperia o beacon.
  const networks = useQuery({
    queryKey: ["hotspot", "wifi-scan"],
    queryFn: () => api.get<WifiNetwork[]>("/hotspot/wifi-scan"),
  });

  const selectedSsid = watch("WIFI_ANCHOR_SSID");
  const selectedChannel = watch("WIFI_ANCHOR_CHANNEL");
  const visible = networks.data ?? [];
  // A rede gravada pode não estar visível agora (router desligado,
  // máquina noutro sítio). Mantém-na na lista para não se perder a
  // seleção só por ela estar fora do ar.
  const missingSelected = selectedSsid && !visible.some((n) => n.ssid === selectedSsid);

  return (
    <div className="space-y-2 sm:col-span-2">
      <Label htmlFor="WIFI_ANCHOR_SSID">Rede a manter acessível (âncora de canal)</Label>
      <SelectNative
        id="WIFI_ANCHOR_SSID"
        {...register("WIFI_ANCHOR_SSID")}
        onChange={(event) => {
          const ssid = event.target.value;
          setValue("WIFI_ANCHOR_SSID", ssid, { shouldDirty: true });
          const match = visible.find((n) => n.ssid === ssid);
          setValue("WIFI_ANCHOR_CHANNEL", match ? String(match.channel) : "", { shouldDirty: true });
        }}
      >
        <option value="">Nenhuma — escolher o canal menos ocupado</option>
        {missingSelected && (
          <option value={selectedSsid}>
            {selectedSsid} (fora do ar{selectedChannel ? `, último canal conhecido: ${selectedChannel}` : ""})
          </option>
        )}
        {visible.map((network) => (
          <option key={network.ssid} value={network.ssid}>
            {network.ssid} — canal {network.channel} · sinal {network.signal}%
          </option>
        ))}
      </SelectNative>
      <input type="hidden" {...register("WIFI_ANCHOR_CHANNEL")} />
      <p className="text-xs text-muted-foreground">
        Um rádio só transmite numa frequência de cada vez, por isso a máquina só se consegue ligar a redes que
        estejam <strong>no mesmo canal do hotspot</strong>. Escolher aqui a rede que costuma usar faz o hotspot subir
        no canal dela. Sem âncora, o hotspot procura o canal mais vazio — que é justamente onde não há rede nenhuma a
        que se ligar. Dá acesso a <strong>uma</strong> rede de cada vez: as que estiverem noutros canais ficam
        inalcançáveis enquanto o hotspot estiver ligado.
        {networks.isError && " (Não foi possível listar as redes visíveis agora.)"}
      </p>
    </div>
  );
}
