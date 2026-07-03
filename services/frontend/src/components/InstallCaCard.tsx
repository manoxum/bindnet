import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { Download, HardDriveDownload, ShieldCheck } from "lucide-react";
import { toast } from "sonner";
import { Button, buttonVariants } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { api, ApiError } from "@/lib/api";

type Os = "windows" | "macos" | "linux" | "mobile";

const osLabels: Record<Os, string> = {
  windows: "Windows",
  macos: "macOS",
  linux: "Linux",
  mobile: "Android / iOS",
};

function caUrl() {
  return `${window.location.origin}/api/certificates/ca`;
}

function downloadTextFile(filename: string, contents: string) {
  const blob = new Blob([contents], { type: "text/plain;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  link.click();
  URL.revokeObjectURL(url);
}

function windowsScript() {
  return [
    "@echo off",
    "REM Instala a CA local do Bindnet na loja de certificados confiaveis do Windows.",
    "REM Execute como Administrador (botao direito > Executar como administrador).",
    `certutil -urlcache -f "${caUrl()}" bindnet-ca.crt`,
    'certutil -addstore -f "ROOT" bindnet-ca.crt',
    "del bindnet-ca.crt",
    "pause",
  ].join("\r\n");
}

function linuxScript() {
  return [
    "#!/usr/bin/env bash",
    "set -euo pipefail",
    "# Instala a CA local do Bindnet na loja de certificados confiaveis do sistema.",
    "# Execute com sudo.",
    `curl -fsSL "${caUrl()}" -o /usr/local/share/ca-certificates/bindnet-ca.crt`,
    "update-ca-certificates",
  ].join("\n");
}

type BrowserTrustResult = {
  store: string;
  path: string;
  installed: boolean;
  error?: string;
};

type InstallLocalCAResult = {
  path: string;
  output?: string;
  browserStores?: BrowserTrustResult[];
};

function browserStoresSummary(browserStores?: BrowserTrustResult[]) {
  if (!browserStores || browserStores.length === 0) return "";
  const installed = browserStores.filter((store) => store.installed);
  if (installed.length === 0) return "";
  const stores = [...new Set(installed.map((store) => store.store))];
  return ` e nas stores de ${stores.join(", ")}`;
}

export function InstallCaCard() {
  const [os, setOs] = useState<Os>("windows");
  const installLocal = useMutation({
    mutationFn: () => api.post<InstallLocalCAResult>("/certificates/ca/install-local"),
    onSuccess: (result) =>
      toast.success(`CA instalada em ${result.path}${browserStoresSummary(result.browserStores)}`),
    onError: (error) =>
      toast.error(error instanceof ApiError ? error.message : "Falha ao instalar CA neste computador"),
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle>Instalar CA neste computador</CardTitle>
        <CardDescription>
          Use o worker para confiar nesta CA no servidor Linux atual ou baixe um instalador para outro dispositivo.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex flex-col gap-3 rounded-md border bg-muted/30 p-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex gap-3">
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-md border bg-background">
              <ShieldCheck className="h-5 w-5 text-primary" />
            </div>
            <div className="space-y-1">
              <p className="text-sm font-medium">Instalação local via worker</p>
              <p className="text-sm text-muted-foreground">
                Grava a CA no armazém de confiança Linux deste host, executa update-ca-certificates e importa a CA
                nas stores próprias do Chrome/Chromium e do Firefox do usuário que roda o painel.
              </p>
            </div>
          </div>
          <Button onClick={() => installLocal.mutate()} disabled={installLocal.isPending}>
            <HardDriveDownload className="h-4 w-4" />
            {installLocal.isPending ? "Instalando..." : "Instalar localmente"}
          </Button>
        </div>

        <div className="flex flex-wrap gap-2">
          {(Object.keys(osLabels) as Os[]).map((key) => (
            <Button key={key} size="sm" variant={os === key ? "default" : "outline"} onClick={() => setOs(key)}>
              {osLabels[key]}
            </Button>
          ))}
        </div>

        {os === "windows" && (
          <div className="space-y-3 text-sm">
            <p className="text-muted-foreground">
              Baixe o script abaixo e execute-o como Administrador (botão direito no arquivo &gt; "Executar como
              administrador"). Ele baixa a CA e instala na loja de certificados confiáveis do Windows.
            </p>
            <Button onClick={() => downloadTextFile("instalar-ca-bindnet.cmd", windowsScript())}>
              <Download className="mr-2 h-4 w-4" />
              Baixar script de instalação (.cmd)
            </Button>
          </div>
        )}

        {os === "linux" && (
          <div className="space-y-3 text-sm">
            <p className="text-muted-foreground">
              Baixe o script abaixo e execute-o com <code>sudo bash instalar-ca-bindnet.sh</code>. Ele baixa a CA e
              roda <code>update-ca-certificates</code>. Se este painel está no mesmo servidor, o botão acima faz isso
              pelo worker.
            </p>
            <Button onClick={() => downloadTextFile("instalar-ca-bindnet.sh", linuxScript())}>
              <Download className="mr-2 h-4 w-4" />
              Baixar script de instalação (.sh)
            </Button>
          </div>
        )}

        {os === "macos" && (
          <div className="space-y-3 text-sm">
            <p className="text-muted-foreground">Baixe a CA e depois rode no Terminal:</p>
            <pre className="overflow-x-auto rounded-md bg-muted p-3 text-xs">
              {`sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain bindnet-ca.crt`}
            </pre>
            <a href="/api/certificates/ca" download className={buttonVariants({ variant: "outline" })}>
              <Download className="mr-2 h-4 w-4" />
              Baixar CA
            </a>
          </div>
        )}

        {os === "mobile" && (
          <div className="space-y-2 text-sm text-muted-foreground">
            <p>
              <strong>Android:</strong> baixe a CA abaixo e vá em Configurações &gt; Segurança &gt; Criptografia e
              credenciais &gt; Instalar um certificado &gt; Certificado de CA.
            </p>
            <p>
              <strong>iOS:</strong> baixe a CA abaixo, instale o perfil quando solicitado, depois vá em
              Configurações &gt; Geral &gt; Sobre &gt; Configurações de confiança de certificado e ative confiança
              total para a CA do Bindnet.
            </p>
            <a href="/api/certificates/ca" download className={buttonVariants({ variant: "outline" })}>
              <Download className="mr-2 h-4 w-4" />
              Baixar CA
            </a>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
