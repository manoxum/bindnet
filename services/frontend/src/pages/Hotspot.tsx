import { useEffect, useRef, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Tabs } from "@/components/ui/tabs";
import { HotspotDialogs } from "@/components/hotspot/HotspotDialogs";
import { HotspotTabsList } from "@/components/hotspot/HotspotTabsList";
import { HotspotTabsContent } from "@/components/hotspot/HotspotTabsContent";
import { HotspotSummaryCard } from "@/components/hotspot/HotspotSummaryCard";
import { interfaceLabel } from "@/components/hotspot/HotspotInterfacesTab";
import { configSchema, type ConfigForm } from "@/components/hotspot/hotspot-schema";
import { hotspotConfigDefaults } from "@/components/hotspot/hotspot-config-defaults";
import { useHotspotQueries } from "@/components/hotspot/useHotspotQueries";
import { useHotspotMutations } from "@/components/hotspot/useHotspotMutations";
import { usePageHeader } from "@/hooks/usePageHeader";
import { useUrlTab } from "@/hooks/useUrlTab";

export function HotspotPage() {
  const [showPassword, setShowPassword] = useState(false);
  const [configOpen, setConfigOpen] = useState(false);
  const [confirmRecoverOpen, setConfirmRecoverOpen] = useState(false);
  const autoPromptedRef = useRef(false);
  const [tab, setTab] = useUrlTab("connected");

  const queries = useHotspotQueries();
  const { status, config, interfaces, clients, blocklist } = queries;
  const mutations = useHotspotMutations({
    onSaveSuccess: () => setConfigOpen(false),
    onRecoverSuccess: () => setConfirmRecoverOpen(false),
  });
  const { saveAndApply, switchUplink, start, stop, recoverWifi } = mutations;

  const {
    register,
    handleSubmit,
    reset,
    watch,
    setValue,
    formState: { errors, isDirty },
  } = useForm<ConfigForm>({
    resolver: zodResolver(configSchema),
  });
  const wifiOpen = watch("WIFI_OPEN") === "true";

  const wifiInterfaces = interfaces.data?.filter((i) => i.type === "wifi") ?? [];
  const networkInterfaces = interfaces.data ?? [];
  const wifiInterfaceOptions = wifiInterfaces.map((i) => ({ value: i.name, label: `${i.name} (${i.state})` }));
  const internetInterfaceOptions = [
    { value: "auto", label: "Automática (melhor disponível)" },
    ...networkInterfaces.map((i) => ({ value: i.name, label: interfaceLabel(i) })),
  ];

  // Troca rapida de interface pelo card de resumo (sem abrir o dialog
  // inteiro de "Alterar configuracao") - a escolha do usuario no
  // dropdown ja e a confirmacao, igual clicar em "Salvar e aplicar" no
  // dialog. INTERNET_INTERFACE tem caminho proprio (POST
  // /api/hotspot/uplink): so a fonte de uplink muda, ao vivo, sem
  // reiniciar o hotspot nem derrubar os clientes conectados; os demais
  // campos continuam no salvar+aplicar completo (reinicia o AP).
  const handleQuickSwitchInterface = (field: "WIFI_INTERFACE" | "INTERNET_INTERFACE" | "WIFI_OPEN", value: string) => {
    if (!config.data) return;
    if (field === "INTERNET_INTERFACE") {
      switchUplink.mutate(value);
      return;
    }
    saveAndApply.mutate({ ...config.data, [field]: value } as ConfigForm);
  };

  // Preenche o formulario assim que config+interfaces carregarem. Quando
  // ainda nao configurado (instalacao nova), sugere valores inteligentes
  // em vez de deixar em branco: interface Wi-Fi unica ja vem selecionada e
  // uma senha aleatoria ja vem pronta pro operador aceitar ou trocar. Se
  // SSID/interface ainda estiverem vazios, abre o dialogo de configuracao
  // automaticamente uma unica vez, em vez de depender do operador saber
  // clicar em "Alterar configuracao".
  useEffect(() => {
    if (!config.data || !interfaces.data) return;
    const needsSetup = !config.data.WIFI_SSID || !config.data.WIFI_INTERFACE;
    reset(hotspotConfigDefaults(config.data, wifiInterfaces));
    if (needsSetup && !autoPromptedRef.current) {
      autoPromptedRef.current = true;
      setConfigOpen(true);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [config.data, interfaces.data, reset]);

  const connectedCount = clients.data?.length ?? 0;
  const blockedCount = blocklist.data?.length ?? 0;
  const blockedMacs = new Set(blocklist.data?.map((device) => device.macAddress) ?? []);

  // "starting": serviço vivo mas AP ainda não no ar (ou caído e a ser
  // retentado). Distinto de ligado — ver status_service em
  // services/worker/hotspot/entrypoint.sh.
  const starting = status.data?.status === "starting";

  usePageHeader({
    title: "Hotspot Wi-Fi",
    description: starting
      ? "A iniciar…"
      : status.data?.running
        ? `Rodando · canal ${status.data.channel ?? "?"} · banda ${status.data.band ?? "?"}GHz`
        : "Parado",
  });

  return (
    <div className="space-y-6">
      <HotspotSummaryCard
        config={config.data ?? {}}
        running={!!status.data?.running}
        starting={starting}
        currentBand={status.data?.band}
        currentChannel={status.data?.channel}
        currentInternetInterface={status.data?.internetInterface}
        wifiInterfaceOptions={wifiInterfaceOptions}
        internetInterfaceOptions={internetInterfaceOptions}
        onQuickSwitchInterface={handleQuickSwitchInterface}
        quickSwitchPending={saveAndApply.isPending || switchUplink.isPending}
        startPending={start.isPending}
        stopPending={stop.isPending}
        recoverPending={recoverWifi.isPending}
        onStart={() => start.mutate()}
        onStop={() => stop.mutate()}
        onRecover={() => {
          if (status.data?.running) {
            setConfirmRecoverOpen(true);
            return;
          }
          recoverWifi.mutate();
        }}
        onEdit={() => setConfigOpen(true)}
      />

      <Tabs value={tab} onValueChange={setTab} className="space-y-4">
        <HotspotTabsList connectedCount={connectedCount} blockedCount={blockedCount} />
        <HotspotTabsContent queries={queries} mutations={mutations} blockedMacs={blockedMacs} />
      </Tabs>

      <HotspotDialogs
        configOpen={configOpen}
        onConfigOpenChange={setConfigOpen}
        register={register}
        setValue={setValue}
        watch={watch}
        errors={errors}
        handleSubmit={handleSubmit}
        onSave={(data) => saveAndApply.mutate(data)}
        isDirty={isDirty}
        savePending={saveAndApply.isPending}
        showPassword={showPassword}
        onToggleShowPassword={() => setShowPassword((v) => !v)}
        wifiOpen={wifiOpen}
        onWifiOpenChange={(open) => setValue("WIFI_OPEN", open ? "true" : "false", { shouldDirty: true, shouldValidate: true })}
        wifiInterfaces={wifiInterfaces}
        networkInterfaces={networkInterfaces}
        confirmRecoverOpen={confirmRecoverOpen}
        onConfirmRecoverOpenChange={setConfirmRecoverOpen}
        recoverPending={recoverWifi.isPending}
        onConfirmRecover={() => recoverWifi.mutate()}
      />
    </div>
  );
}
