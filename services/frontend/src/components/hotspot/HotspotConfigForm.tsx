import type { FieldErrors, UseFormHandleSubmit, UseFormRegister } from "react-hook-form";
import { Eye, EyeOff } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { SelectNative } from "@/components/ui/select-native";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import type { ConfigForm } from "@/components/hotspot/hotspot-schema";

interface NetworkInterface {
  name: string;
  type: "wifi" | "other";
  state: string;
  speedMbps?: number;
}

interface HotspotConfigFormProps {
  register: UseFormRegister<ConfigForm>;
  errors: FieldErrors<ConfigForm>;
  handleSubmit: UseFormHandleSubmit<ConfigForm>;
  onSave: (data: ConfigForm) => void;
  onApply: () => void;
  isDirty: boolean;
  savePending: boolean;
  applyPending: boolean;
  showPassword: boolean;
  onToggleShowPassword: () => void;
  wifiInterfaces: NetworkInterface[];
  networkInterfaces: NetworkInterface[];
}

function interfaceLabel(i: NetworkInterface) {
  const speed = i.speedMbps ? `, ${i.speedMbps}Mbps` : "";
  return `${i.name} (${i.type}, ${i.state}${speed})`;
}

// Formulário de configuração do hotspot, separado em abas para manter o modal compacto.
export function HotspotConfigForm({
  register,
  errors,
  handleSubmit,
  onSave,
  onApply,
  isDirty,
  savePending,
  applyPending,
  showPassword,
  onToggleShowPassword,
  wifiInterfaces,
  networkInterfaces,
}: HotspotConfigFormProps) {
  return (
    <form className="space-y-5" onSubmit={handleSubmit(onSave)}>
      <Tabs defaultValue="wifi" className="space-y-4">
        <TabsList className="grid h-auto w-full grid-cols-2 gap-1 sm:grid-cols-5">
          <TabsTrigger value="wifi">Wi-Fi</TabsTrigger>
          <TabsTrigger value="interfaces">Interfaces</TabsTrigger>
          <TabsTrigger value="radio">Rádio</TabsTrigger>
          <TabsTrigger value="network">Rede</TabsTrigger>
          <TabsTrigger value="uplink">Uplink</TabsTrigger>
        </TabsList>

        <TabsContent value="wifi" className="mt-0">
          <fieldset className="space-y-4">
            <legend className="text-sm font-medium text-muted-foreground">Rede Wi-Fi</legend>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="WIFI_SSID">SSID</Label>
                <Input id="WIFI_SSID" {...register("WIFI_SSID")} />
                {errors.WIFI_SSID && <p className="text-sm text-destructive">{errors.WIFI_SSID.message}</p>}
              </div>
              <div className="space-y-2">
                <Label htmlFor="WIFI_PASSWORD">Senha</Label>
                <div className="relative">
                  <Input id="WIFI_PASSWORD" type={showPassword ? "text" : "password"} {...register("WIFI_PASSWORD")} />
                  <button
                    type="button"
                    className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground"
                    onClick={onToggleShowPassword}
                  >
                    {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                  </button>
                </div>
                {errors.WIFI_PASSWORD && <p className="text-sm text-destructive">{errors.WIFI_PASSWORD.message}</p>}
              </div>
            </div>
          </fieldset>
        </TabsContent>

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
            </div>
          </fieldset>
        </TabsContent>

        <TabsContent value="radio" className="mt-0">
          <fieldset className="space-y-4">
            <legend className="text-sm font-medium text-muted-foreground">Rádio</legend>
            <div className="grid gap-4 sm:grid-cols-3">
              <div className="space-y-2">
                <Label htmlFor="WIFI_COUNTRY">País (código Wi-Fi)</Label>
                <Input id="WIFI_COUNTRY" maxLength={2} {...register("WIFI_COUNTRY")} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="WIFI_FREQ_BAND">Banda</Label>
                <SelectNative id="WIFI_FREQ_BAND" {...register("WIFI_FREQ_BAND")}>
                  <option value="auto">Automática</option>
                  <option value="2.4">2.4GHz</option>
                  <option value="5">5GHz</option>
                </SelectNative>
              </div>
              <div className="space-y-2">
                <Label htmlFor="WIFI_CHANNEL">Canal</Label>
                <Input id="WIFI_CHANNEL" placeholder="auto" {...register("WIFI_CHANNEL")} />
              </div>
              <div className="space-y-2 sm:col-span-3">
                <Label htmlFor="WIFI_CHANNEL_CANDIDATES">Canais candidatos</Label>
                <Input id="WIFI_CHANNEL_CANDIDATES" placeholder="1,6,11" {...register("WIFI_CHANNEL_CANDIDATES")} />
              </div>
            </div>
          </fieldset>
        </TabsContent>

        <TabsContent value="network" className="mt-0">
          <fieldset className="space-y-4">
            <legend className="text-sm font-medium text-muted-foreground">Rede do hotspot</legend>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="HOTSPOT_GATEWAY">Gateway</Label>
                <Input id="HOTSPOT_GATEWAY" placeholder="192.168.12.1" {...register("HOTSPOT_GATEWAY")} />
                {errors.HOTSPOT_GATEWAY && (
                  <p className="text-sm text-destructive">{errors.HOTSPOT_GATEWAY.message}</p>
                )}
              </div>
              <div className="space-y-2">
                <Label htmlFor="HOTSPOT_CIDR">Faixa CIDR</Label>
                <Input id="HOTSPOT_CIDR" placeholder="192.168.12.0/24" {...register("HOTSPOT_CIDR")} />
                {errors.HOTSPOT_CIDR && <p className="text-sm text-destructive">{errors.HOTSPOT_CIDR.message}</p>}
              </div>
              <div className="space-y-2 sm:col-span-2">
                <Label htmlFor="HOTSPOT_DNS_FALLBACKS">DNS públicos de fallback</Label>
                <Input
                  id="HOTSPOT_DNS_FALLBACKS"
                  placeholder="1.1.1.1,8.8.8.8"
                  {...register("HOTSPOT_DNS_FALLBACKS")}
                />
              </div>
            </div>
          </fieldset>
        </TabsContent>

        <TabsContent value="uplink" className="mt-0">
          <fieldset className="space-y-4">
            <legend className="text-sm font-medium text-muted-foreground">Uplink</legend>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="BINDNET_UPLINK_INTERFACE">Interface virtual</Label>
                <Input id="BINDNET_UPLINK_INTERFACE" placeholder="bn-uplink" {...register("BINDNET_UPLINK_INTERFACE")} />
                {errors.BINDNET_UPLINK_INTERFACE && (
                  <p className="text-sm text-destructive">{errors.BINDNET_UPLINK_INTERFACE.message}</p>
                )}
              </div>
              <div className="space-y-2">
                <Label htmlFor="UPLINK_MONITOR_INTERVAL">Intervalo do monitor</Label>
                <Input id="UPLINK_MONITOR_INTERVAL" placeholder="10" {...register("UPLINK_MONITOR_INTERVAL")} />
                {errors.UPLINK_MONITOR_INTERVAL && (
                  <p className="text-sm text-destructive">{errors.UPLINK_MONITOR_INTERVAL.message}</p>
                )}
              </div>
            </div>
          </fieldset>
        </TabsContent>
      </Tabs>

      <div className="flex flex-col-reverse gap-2 border-t pt-4 sm:flex-row sm:items-center sm:justify-end">
        <Button type="submit" disabled={!isDirty || savePending}>
          Salvar
        </Button>
        <Button type="button" variant="outline" onClick={onApply} disabled={applyPending}>
          Aplicar e reiniciar
        </Button>
      </div>
    </form>
  );
}
