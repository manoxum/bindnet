import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Eye, EyeOff } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { SelectNative } from "@/components/ui/select-native";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { LogsPanel } from "@/components/LogsPanel";
import { api, ApiError } from "@/lib/api";

interface HotspotStatus {
  running: boolean;
  status: string;
  channel?: string;
  band?: string;
}
interface NetworkInterface {
  name: string;
  type: "wifi" | "other";
  state: string;
}
interface HotspotClient {
  mac: string;
  ip: string;
  hostname: string;
}

const configSchema = z.object({
  WIFI_SSID: z.string().min(1, "Informe o SSID"),
  WIFI_PASSWORD: z.string().min(8, "Mínimo de 8 caracteres (WPA2)"),
  WIFI_INTERFACE: z.string().min(1, "Selecione a interface Wi-Fi"),
  INTERNET_INTERFACE: z.string().min(1, "Selecione a interface de internet"),
  WIFI_COUNTRY: z.string().min(2).max(2),
  WIFI_CHANNEL: z.string().min(1),
  WIFI_FREQ_BAND: z.string().min(1),
});
type ConfigForm = z.infer<typeof configSchema>;

export function HotspotPage() {
  const queryClient = useQueryClient();
  const [showPassword, setShowPassword] = useState(false);

  const status = useQuery<HotspotStatus>({
    queryKey: ["hotspot", "status"],
    queryFn: () => api.get<HotspotStatus>("/hotspot/status"),
    refetchInterval: 5000,
  });
  const config = useQuery<Record<string, string>>({
    queryKey: ["hotspot", "config"],
    queryFn: () => api.get<Record<string, string>>("/hotspot/config"),
  });
  const interfaces = useQuery<NetworkInterface[]>({
    queryKey: ["hotspot", "interfaces"],
    queryFn: () => api.get<NetworkInterface[]>("/hotspot/interfaces"),
  });
  const clients = useQuery<HotspotClient[]>({
    queryKey: ["hotspot", "clients"],
    queryFn: () => api.get<HotspotClient[]>("/hotspot/clients"),
    refetchInterval: 5000,
    enabled: !!status.data?.running,
  });

  const { register, handleSubmit, reset, formState: { errors, isDirty } } = useForm<ConfigForm>({
    resolver: zodResolver(configSchema),
  });

  useEffect(() => {
    if (config.data) {
      reset({
        WIFI_SSID: config.data.WIFI_SSID ?? "",
        WIFI_PASSWORD: config.data.WIFI_PASSWORD ?? "",
        WIFI_INTERFACE: config.data.WIFI_INTERFACE ?? "",
        INTERNET_INTERFACE: config.data.INTERNET_INTERFACE ?? "",
        WIFI_COUNTRY: config.data.WIFI_COUNTRY ?? "ST",
        WIFI_CHANNEL: config.data.WIFI_CHANNEL ?? "auto",
        WIFI_FREQ_BAND: config.data.WIFI_FREQ_BAND ?? "auto",
      });
    }
  }, [config.data, reset]);

  const save = useMutation({
    mutationFn: (data: ConfigForm) => api.patch("/hotspot/config", data),
    onSuccess: () => {
      toast.success("Configuração salva. Clique em 'Aplicar' para reiniciar o hotspot com os novos valores.");
      queryClient.invalidateQueries({ queryKey: ["hotspot", "config"] });
    },
    onError: (error) => toast.error(error instanceof ApiError ? error.message : "Falha ao salvar"),
  });

  const apply = useMutation({
    mutationFn: () => api.post("/hotspot/apply"),
    onSuccess: () => {
      toast.success("Hotspot recriado com a configuração atual.");
      queryClient.invalidateQueries({ queryKey: ["hotspot"] });
    },
    onError: (error) => toast.error(error instanceof ApiError ? error.message : "Falha ao aplicar"),
  });

  const start = useMutation({
    mutationFn: () => api.post("/hotspot/start"),
    onSuccess: () => {
      toast.success("Hotspot iniciado.");
      queryClient.invalidateQueries({ queryKey: ["hotspot"] });
    },
    onError: (error) => toast.error(error instanceof ApiError ? error.message : "Falha ao iniciar"),
  });

  const stop = useMutation({
    mutationFn: () => api.post("/hotspot/stop"),
    onSuccess: () => {
      toast.success("Hotspot parado.");
      queryClient.invalidateQueries({ queryKey: ["hotspot"] });
    },
    onError: (error) => toast.error(error instanceof ApiError ? error.message : "Falha ao parar"),
  });

  const wifiInterfaces = interfaces.data?.filter((i) => i.type === "wifi") ?? [];
  const networkInterfaces = interfaces.data ?? [];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold">Hotspot Wi-Fi</h1>
          <p className="text-sm text-muted-foreground">
            {status.data?.running ? (
              <>
                Rodando · canal {status.data.channel ?? "?"} · banda {status.data.band ?? "?"}GHz
              </>
            ) : (
              "Parado"
            )}
          </p>
        </div>
        <div className="flex gap-2">
          <Badge variant={status.data?.running ? "success" : "secondary"}>
            {status.data?.running ? "ligado" : "desligado"}
          </Badge>
          <Button onClick={() => start.mutate()} disabled={status.data?.running || start.isPending}>
            Iniciar
          </Button>
          <Button
            variant="destructive"
            onClick={() => stop.mutate()}
            disabled={!status.data?.running || stop.isPending}
          >
            Parar
          </Button>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Configuração</CardTitle>
          <CardDescription>
            Salvar grava no .env; "Aplicar" recria o hotspot para assumir os novos valores (derruba a conexão por
            alguns segundos).
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form className="grid gap-4 sm:grid-cols-2" onSubmit={handleSubmit((data) => save.mutate(data))}>
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
                  onClick={() => setShowPassword((v) => !v)}
                >
                  {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
              {errors.WIFI_PASSWORD && <p className="text-sm text-destructive">{errors.WIFI_PASSWORD.message}</p>}
            </div>
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
                {networkInterfaces.map((i) => (
                  <option key={i.name} value={i.name}>
                    {i.name} ({i.type}, {i.state})
                  </option>
                ))}
              </SelectNative>
            </div>
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

            <div className="flex items-end gap-2 sm:col-span-2">
              <Button type="submit" disabled={!isDirty || save.isPending}>
                Salvar
              </Button>
              <Button type="button" variant="outline" onClick={() => apply.mutate()} disabled={apply.isPending}>
                Aplicar e reiniciar
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Clientes conectados</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>MAC</TableHead>
                <TableHead>IP</TableHead>
                <TableHead>Hostname</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(clients.data ?? []).map((client) => (
                <TableRow key={client.mac}>
                  <TableCell>{client.mac}</TableCell>
                  <TableCell>{client.ip}</TableCell>
                  <TableCell>{client.hostname}</TableCell>
                </TableRow>
              ))}
              {status.data?.running && (clients.data ?? []).length === 0 && (
                <TableRow>
                  <TableCell colSpan={3} className="text-center text-muted-foreground">
                    Nenhum cliente conectado.
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <LogsPanel title="Logs do hotspot" path="/hotspot/logs" />
    </div>
  );
}
