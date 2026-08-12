import type { FieldErrors, UseFormHandleSubmit, UseFormRegister, UseFormSetValue, UseFormWatch } from "react-hook-form";
import { Button } from "@/components/ui/button";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import type { ConfigForm } from "@/components/hotspot/hotspot-schema";
import { HotspotWifiTab } from "@/components/hotspot/HotspotWifiTab";
import { HotspotInterfacesTab } from "@/components/hotspot/HotspotInterfacesTab";
import { HotspotRadioTab } from "@/components/hotspot/HotspotRadioTab";
import { HotspotNetworkTab } from "@/components/hotspot/HotspotNetworkTab";
import { HotspotUplinkTab } from "@/components/hotspot/HotspotUplinkTab";
import { HotspotServiceTab } from "@/components/hotspot/HotspotServiceTab";

interface NetworkInterface {
  name: string;
  type: "wifi" | "other";
  state: string;
  speedMbps?: number;
}

interface HotspotConfigFormProps {
  register: UseFormRegister<ConfigForm>;
  setValue: UseFormSetValue<ConfigForm>;
  watch: UseFormWatch<ConfigForm>;
  errors: FieldErrors<ConfigForm>;
  handleSubmit: UseFormHandleSubmit<ConfigForm>;
  onSave: (data: ConfigForm) => void;
  isDirty: boolean;
  savePending: boolean;
  showPassword: boolean;
  onToggleShowPassword: () => void;
  wifiOpen: boolean;
  onWifiOpenChange: (open: boolean) => void;
  wifiInterfaces: NetworkInterface[];
  networkInterfaces: NetworkInterface[];
  // false esconde o botão Salvar e aplicar - usado pelo assistente de
  // configuração inicial, que só salva/aplica tudo no último passo.
  showActions?: boolean;
  // false esconde a aba "Serviço" - o assistente de configuração
  // inicial só grava tudo no fim, e o interruptor de arranque
  // automático dessa aba grava sozinho, na hora (ver
  // HotspotServiceTab): misturar os dois ali confundiria o passo a
  // passo.
  showServiceTab?: boolean;
}

// Formulário de configuração do hotspot, separado em abas (uma por
// arquivo em components/hotspot/Hotspot*Tab.tsx) para manter este shell
// compacto.
export function HotspotConfigForm({
  register,
  setValue,
  watch,
  errors,
  handleSubmit,
  onSave,
  isDirty,
  savePending,
  showPassword,
  onToggleShowPassword,
  wifiOpen,
  onWifiOpenChange,
  wifiInterfaces,
  networkInterfaces,
  showActions = true,
  showServiceTab = true,
}: HotspotConfigFormProps) {
  return (
    <form className="space-y-5" onSubmit={handleSubmit(onSave)}>
      <Tabs defaultValue="wifi" className="space-y-4">
        <TabsList className={`grid h-auto w-full grid-cols-2 gap-1 ${showServiceTab ? "sm:grid-cols-6" : "sm:grid-cols-5"}`}>
          <TabsTrigger value="wifi">Wi-Fi</TabsTrigger>
          <TabsTrigger value="interfaces">Interfaces</TabsTrigger>
          <TabsTrigger value="radio">Rádio</TabsTrigger>
          <TabsTrigger value="network">Rede</TabsTrigger>
          <TabsTrigger value="uplink">Uplink</TabsTrigger>
          {showServiceTab && <TabsTrigger value="service">Serviço</TabsTrigger>}
        </TabsList>

        <HotspotWifiTab
          register={register}
          errors={errors}
          showPassword={showPassword}
          onToggleShowPassword={onToggleShowPassword}
          wifiOpen={wifiOpen}
          onWifiOpenChange={onWifiOpenChange}
        />
        <HotspotInterfacesTab
          register={register}
          wifiInterfaces={wifiInterfaces}
          networkInterfaces={networkInterfaces}
        />
        <HotspotRadioTab register={register} setValue={setValue} watch={watch} />
        <HotspotNetworkTab register={register} errors={errors} />
        <HotspotUplinkTab register={register} errors={errors} />
        {showServiceTab && <HotspotServiceTab />}
      </Tabs>

      {showActions && (
        <div className="flex flex-col-reverse gap-2 border-t pt-4 sm:flex-row sm:items-center sm:justify-end">
          <Button type="submit" disabled={!isDirty || savePending}>
            {savePending ? "Salvando e aplicando..." : "Salvar e aplicar"}
          </Button>
        </div>
      )}
    </form>
  );
}
