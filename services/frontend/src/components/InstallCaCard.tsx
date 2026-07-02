import { useState } from "react";
import { Download } from "lucide-react";
import { Button, buttonVariants } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";

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

export function InstallCaCard() {
  const [os, setOs] = useState<Os>("windows");

  return (
    <Card>
      <CardHeader>
        <CardTitle>Instalar CA neste computador</CardTitle>
        <CardDescription>
          Escolha o sistema operacional para ver o passo a passo e baixar um instalador automático quando
          disponível.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
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
              roda <code>update-ca-certificates</code> (Debian/Ubuntu; distribuições baseadas em outro gerenciador
              de certificados podem precisar de um comando equivalente).
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
