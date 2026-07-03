import type { FieldErrors, UseFormHandleSubmit, UseFormRegister } from "react-hook-form";
import { Eye, EyeOff } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { SelectNative } from "@/components/ui/select-native";
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

// Formulário de configuração do hotspot, agrupado por assunto (rede, interfaces, rádio) para facilitar a leitura.
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
    <form className="space-y-6" onSubmit={handleSubmit(onSave)}>
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
        </div>
      </fieldset>

      <div className="flex items-end gap-2">
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
