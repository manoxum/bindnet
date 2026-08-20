import { Link } from "react-router-dom";
import { ArrowLeft, Download, ShieldCheck } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { buttonVariants } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { PortalCaInstructions } from "@/components/portal/PortalCaInstructions";

// Página pública de instalação da CA local, acessível a partir do
// portal cativo em bindnet.local.com. Sem sessão de propósito: quem acabou de se
// ligar ao Wi-Fi precisa de instalar a CA *antes* de conseguir navegar
// nos sites internos sem aviso de certificado, e nessa altura não tem
// conta nenhuma no painel.
//
// O download é um <a href> direto para GET /api/mesh/ca — a única rota
// de certificado que já era pública por desenho (ver comentário em
// services/backend/internal/cert/certificates_routes.go). Devolve só o
// certificado PÚBLICO da CA, nunca a chave privada, e já vem com
// Content-Disposition: attachment, por isso não é preciso fetch nem
// blob no browser.
export function PortalCaPage() {
  return (
    <div className="portal-surface portal-grid flex min-h-screen items-start justify-center bg-[#06120f] p-4 py-8 text-[#e9fff6] sm:py-14">
      <Card className="w-full max-w-lg rounded-[1.75rem] border-white/10 bg-[#0b211b]/95 shadow-[0_24px_80px_rgba(0,0,0,0.35)]">
        <CardHeader>
          <div className="flex items-center gap-2">
            <span className="flex h-10 w-10 items-center justify-center rounded-2xl bg-[#a8ee63]/10">
              <ShieldCheck className="h-5 w-5 text-[#b8f24a]" />
            </span>
            <CardTitle>Certificado de segurança</CardTitle>
          </div>
          <CardDescription className="leading-6 text-[#c5d9d1]">
            Instale este certificado uma vez para abrir os sites internos desta rede sem avisos de segurança.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="rounded-2xl border border-white/10 bg-[#061713]/75 p-4 text-sm leading-6 text-[#d0e1da]">
            Esta rede usa uma autoridade de certificação própria. Sem ela instalada, o seu navegador mostra “ligação
            não privada” ao abrir os endereços internos. O ficheiro contém apenas a parte <strong>pública</strong> do
            certificado.
          </div>

          {/* <a> e nao <Button>: o Button deste repo renderiza sempre
              um <button> (nao suporta asChild), e o download precisa de
              ser uma navegacao de verdade. Caminho relativo para
              funcionar em qualquer dominio pelo qual o portal seja
              servido, sem fixar host nenhum. */}
          <a href="/api/mesh/ca" download className={cn(buttonVariants(), "h-11 w-full bg-[#b8f24a] text-[#102017] hover:bg-[#c9ff78]")}>
            <Download className="h-4 w-4" />
            Descarregar certificado
          </a>

          <div className="space-y-3 rounded-2xl border border-[#9bc8aa] bg-[#dff1e6] p-4">
            <p className="text-sm font-bold text-[#13291e]">Como instalar</p>
            <PortalCaInstructions />
          </div>

          <Link
            to="/"
            className={cn(
              buttonVariants({ variant: "ghost" }),
              "h-11 w-full bg-[#b8f24a] !text-[#102017] shadow-none hover:bg-[#c9ff78] hover:!text-[#102017]",
            )}
          >
            <ArrowLeft className="h-4 w-4" />
            <span className="text-[#e8fff4]">Voltar</span>
          </Link>
        </CardContent>
      </Card>
    </div>
  );
}
