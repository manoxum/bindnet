// ca-install-commands.ts monta o comando "de uma linha só" (Opção 2 -
// instalação rápida) e os passos manuais (Opção 1) para cada sistema
// operacional na tela "Instalar CA neste computador" (InstallCaCard).
// Os comandos usam GET /api/mesh/ca (sem sessão, só o certificado
// público da CA) em vez de GET /api/certificates/ca - um curl ou
// Invoke-WebRequest colado num terminal não carrega o cookie de sessão
// do navegador.

function publicCaUrl(): string {
  return `${window.location.origin}/api/mesh/ca`;
}

export function windowsQuickCommand(): string {
  const tmp = "$env:TEMP\\bindnet-ca.crt";
  return [
    `Invoke-WebRequest -Uri "${publicCaUrl()}" -OutFile "${tmp}"`,
    `Import-Certificate -FilePath "${tmp}" -CertStoreLocation Cert:\\LocalMachine\\Root`,
    `Remove-Item "${tmp}"`,
  ].join("; ");
}

export function macosQuickCommand(): string {
  const tmp = "/tmp/bindnet-ca.crt";
  return [
    `curl -fsSL "${publicCaUrl()}" -o ${tmp}`,
    `sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain ${tmp}`,
    `rm ${tmp}`,
  ].join(" && ");
}

export function linuxQuickCommand(): string {
  return [
    `sudo curl -fsSL "${publicCaUrl()}" -o /usr/local/share/ca-certificates/bindnet-ca.crt`,
    "sudo update-ca-certificates",
  ].join(" && ");
}

export const manualInstallSteps: Record<"windows" | "macos" | "linux", string[]> = {
  windows: [
    'Dê duplo clique no arquivo baixado e clique em "Instalar Certificado...".',
    'Escolha "Máquina Local" (pede permissão de administrador) e avance.',
    'Selecione "Colocar todos os certificados no repositório a seguir", clique em "Procurar" e escolha "Autoridades de Certificação Raiz Confiáveis".',
    "Avance e conclua o assistente.",
  ],
  macos: [
    'Dê duplo clique no arquivo baixado para abrir o "Acesso às Chaves" (Keychain Access).',
    'Adicione-o ao chaveiro "Sistema" quando solicitado.',
    'Dê duplo clique no certificado importado, expanda "Confiar" e defina "Ao usar este certificado" como "Sempre Confiar".',
  ],
  linux: [
    "Copie o arquivo baixado para /usr/local/share/ca-certificates/ (requer sudo).",
    "Rode sudo update-ca-certificates.",
  ],
};
