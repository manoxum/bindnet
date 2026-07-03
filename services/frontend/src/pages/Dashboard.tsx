import { useQuery } from "@tanstack/react-query";
import { Wifi, Globe, Server, Database, HardDrive, KeyRound, Copy } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { toast } from "sonner";
import { api } from "@/lib/api";
import { usePageHeader } from "@/hooks/usePageHeader";

interface ServiceStatus {
  key: string;
  running: boolean;
  status: string;
  startedAt?: string;
}

interface NginxUIInstallSecret {
  secret: string | null;
}

const serviceInfo: Record<string, { title: string; icon: typeof Wifi; link?: string }> = {
  hotspot: { title: "Hotspot Wi-Fi", icon: Wifi },
  dns: { title: "DNS split-horizon", icon: Globe },
  nginxUi: { title: "nginx-ui", icon: Server, link: `http://${window.location.hostname}:9080` },
  postgres: { title: "Postgres", icon: Database },
  mongo: { title: "Mongo", icon: Database },
  minio: { title: "MinIO", icon: HardDrive },
};

export function DashboardPage() {
  usePageHeader({ title: "Visão geral", description: "Status dos serviços do stack bindnet." });

  const { data, isLoading } = useQuery<ServiceStatus[]>({
    queryKey: ["dashboard"],
    queryFn: () => api.get<ServiceStatus[]>("/dashboard"),
    refetchInterval: 5000,
  });

  const installSecret = useQuery<NginxUIInstallSecret>({
    queryKey: ["nginx-ui", "install-secret"],
    queryFn: () => api.get<NginxUIInstallSecret>("/nginx-ui/install-secret"),
    refetchInterval: 10000,
  });

  return (
    <div className="space-y-6">
      {installSecret.data?.secret && (
        <Card className="border-primary/50">
          <CardHeader className="flex flex-row items-center gap-2 space-y-0 pb-2">
            <KeyRound className="h-4 w-4 text-primary" />
            <CardTitle className="text-sm font-medium">Segredo de instalação do nginx-ui</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="mb-2 text-xs text-muted-foreground">
              Use este valor na tela inicial de configuração do nginx-ui. Este card some sozinho assim que a
              configuração inicial for concluída.
            </p>
            <div className="flex items-center gap-2">
              <code className="flex-1 overflow-x-auto rounded-md bg-muted px-2 py-1 text-xs">
                {installSecret.data.secret}
              </code>
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() => {
                  navigator.clipboard.writeText(installSecret.data!.secret!);
                  toast.success("Segredo copiado.");
                }}
              >
                <Copy className="h-4 w-4" />
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {(data ?? Object.keys(serviceInfo).map((key) => ({ key, running: false, status: "" }))).map((service) => {
          const info = serviceInfo[service.key];
          if (!info) return null;
          const Icon = info.icon;
          return (
            <Card key={service.key}>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">{info.title}</CardTitle>
                <Icon className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                {isLoading ? (
                  <span className="text-sm text-muted-foreground">Carregando...</span>
                ) : (
                  <Badge variant={service.running ? "success" : "secondary"}>
                    {service.running ? "rodando" : service.status || "parado"}
                  </Badge>
                )}
                {info.link && (
                  <a href={info.link} target="_blank" rel="noreferrer" className="mt-2 block text-xs text-primary underline">
                    Abrir nginx-ui
                  </a>
                )}
              </CardContent>
            </Card>
          );
        })}
      </div>
    </div>
  );
}
