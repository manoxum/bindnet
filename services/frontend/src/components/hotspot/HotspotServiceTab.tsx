import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { TabsContent } from "@/components/ui/tabs";
import { useHotspotAutostart } from "@/components/hotspot/useHotspotAutostart";

// Aba "Serviço" da configuração do hotspot: opções que governam o
// comportamento do serviço em si, não o rádio nem a rede entregue aos
// clientes.
//
// O interruptor grava sozinho, na hora, e NÃO faz parte do
// "Salvar e aplicar" do formulário — de propósito. Salvar o formulário
// dispara POST /hotspot/apply, que reinicia o hotspot e derruba todos
// os clientes conectados; mudar uma preferência de arranque nunca deve
// custar isso. Por isso a chave mora fora de hotspotConfigKeys e tem
// rota própria (ver hotspot_autostart_routes.go no backend).
export function HotspotServiceTab() {
  const autostart = useHotspotAutostart();

  return (
    <TabsContent value="service" className="mt-0">
      <fieldset className="space-y-4">
        <legend className="text-sm font-medium text-muted-foreground">Serviço</legend>
        <div className="flex flex-col gap-3 rounded-lg border p-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="space-y-1">
            <Label htmlFor="hotspot-autostart">Iniciar automaticamente no arranque</Label>
            <p className="text-sm text-muted-foreground">
              Com isto ligado, o hotspot sobe sozinho sempre que o sistema arranca — útil depois de uma falta de
              energia. Desligado, ele só sobe quando você clicar em "Iniciar".
            </p>
            <p className="text-xs text-muted-foreground">
              Guardado de imediato, sem reiniciar o hotspot nem derrubar quem está conectado.
            </p>
          </div>
          <Switch
            id="hotspot-autostart"
            checked={autostart.enabled}
            disabled={autostart.loading || autostart.pending}
            onCheckedChange={autostart.setEnabled}
            aria-label="Iniciar automaticamente no arranque"
          />
        </div>
      </fieldset>
    </TabsContent>
  );
}
