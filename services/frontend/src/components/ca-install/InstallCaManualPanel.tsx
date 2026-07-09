import { Download } from "lucide-react";
import { buttonVariants } from "@/components/ui/button";

interface InstallCaManualPanelProps {
  steps: string[];
}

// Opção 1: baixa o certificado (autenticado, na mesma sessão do
// navegador) e mostra o passo a passo para instalar manualmente na
// loja de confiança do sistema operacional.
export function InstallCaManualPanel({ steps }: InstallCaManualPanelProps) {
  return (
    <div className="space-y-3 text-sm">
      <p className="font-medium">Opção 1 · Baixar certificado e instalar manualmente</p>
      <a href="/api/certificates/ca" download className={buttonVariants({ variant: "outline", size: "sm" })}>
        <Download className="mr-2 h-4 w-4" />
        Baixar certificado da CA
      </a>
      <ol className="list-decimal space-y-1 pl-5 text-muted-foreground">
        {steps.map((step) => (
          <li key={step}>{step}</li>
        ))}
      </ol>
    </div>
  );
}
