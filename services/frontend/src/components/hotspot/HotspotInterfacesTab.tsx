import type { UseFormRegister } from "react-hook-form";
import { Label } from "@/components/ui/label";
import { SelectNative } from "@/components/ui/select-native";
import { TabsContent } from "@/components/ui/tabs";
import type { ConfigForm } from "@/components/hotspot/hotspot-schema";

interface NetworkInterface {
  name: string;
  type: "wifi" | "other";
  state: string;
  speedMbps?: number;
}

interface HotspotInterfacesTabProps {
  register: UseFormRegister<ConfigForm>;
  wifiInterfaces: NetworkInterface[];
  networkInterfaces: NetworkInterface[];
}

export function interfaceLabel(i: NetworkInterface) {
  const speed = i.speedMbps ? `, ${i.speedMbps}Mbps` : "";
  return `${i.name} (${i.type}, ${i.state}${speed})`;
}

export function HotspotInterfacesTab({ register, wifiInterfaces, networkInterfaces }: HotspotInterfacesTabProps) {
  return (
    <TabsContent value="interfaces" className="mt-0">
      <fieldset className="space-y-4">
        <legend className="text-sm font-medium text-muted-foreground">Interfaces</legend>
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor="WIFI_INTERFACE">Interface Wi-Fi</Label>
            <SelectNative id="WIFI_INTERFACE" {...register("WIFI_INTERFACE")}>
              <option value="">Selecione...</option>
              {wifiInterfaces.map((i) => (
                <option key={i.name} value={i.name}>
                  {i.name} ({i.state})
                </option>
              ))}
            </SelectNative>
          </div>
          <div className="space-y-2">
            <Label htmlFor="INTERNET_INTERFACE">Interface de internet</Label>
            <SelectNative id="INTERNET_INTERFACE" {...register("INTERNET_INTERFACE")}>
              <option value="">Selecione...</option>
              <option value="auto">Automática (melhor disponível)</option>
              {networkInterfaces.map((i) => (
                <option key={i.name} value={i.name}>
                  {interfaceLabel(i)}
                </option>
              ))}
            </SelectNative>
          </div>
          <div className="space-y-2 sm:col-span-2">
            <Label htmlFor="WIFI_AP_MODE">Modo da placa Wi-Fi</Label>
            <SelectNative id="WIFI_AP_MODE" {...register("WIFI_AP_MODE")}>
              <option value="auto">Automático — placa dedicada quando não há Wi-Fi ligado</option>
              <option value="virtual">Virtual — manter sempre a placa disponível no sistema</option>
            </SelectNative>
            <p className="text-xs text-muted-foreground">
              Em <strong>Automático</strong>, se a máquina não estiver ligada a nenhuma rede Wi-Fi quando o hotspot
              arranca, o AP toma a placa inteira e ela desaparece do menu de rede até o hotspot parar. Em{" "}
              <strong>Virtual</strong>, o AP sobe numa interface separada (<code>ap0</code>) e a placa continua
              gerida pelo sistema, dando para ligar-se a uma rede Wi-Fi depois — desde que ela esteja no mesmo canal
              do hotspot.
            </p>
          </div>
        </div>
        <p className="rounded-lg border border-border/60 bg-muted/30 p-3 text-xs text-muted-foreground">
          <span className="font-medium text-foreground">Ordem importa nesta máquina.</span> Para manter a conexão
          Wi-Fi cliente ligada com o hotspot no ar, ligue-se à rede Wi-Fi <em>antes</em> de iniciar o hotspot: um
          rádio único só transmite numa frequência por vez, então o AP é travado no canal da estação. Com o hotspot
          já no ar, a máquina só consegue associar-se a redes que estejam nesse mesmo canal. O log do hotspot mostra
          se a placa aceita mais de um canal simultâneo.
        </p>
      </fieldset>
    </TabsContent>
  );
}
