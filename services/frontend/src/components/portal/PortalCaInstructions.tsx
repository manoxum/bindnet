import { Monitor, Smartphone, Apple, Chrome } from "lucide-react";

interface Platform {
  id: string;
  label: string;
  icon: typeof Monitor;
  steps: string[];
}

// Passos por sistema, escritos para quem não é técnico: cada um começa
// por onde clicar, não pelo conceito. O ficheiro chega sempre com o nome
// bindnet-local-ca.crt (Content-Disposition em serveCertificate,
// services/backend/internal/cert/ca_routes.go).
const PLATFORMS: Platform[] = [
  {
    id: "android",
    label: "Android",
    icon: Smartphone,
    steps: [
      "Toque em “Descarregar certificado” acima e guarde o ficheiro.",
      "Abra Definições → Segurança → Mais definições de segurança → Encriptação e credenciais.",
      "Escolha “Instalar a partir do armazenamento” e selecione bindnet-local-ca.crt.",
      "Quando pedir o tipo, escolha “Certificado de CA”.",
      "Confirme com o PIN ou padrão do telemóvel.",
    ],
  },
  {
    id: "ios",
    label: "iPhone / iPad",
    icon: Apple,
    steps: [
      "Toque em “Descarregar certificado” acima usando o Safari (outros navegadores não funcionam).",
      "Abra Definições — vai aparecer “Perfil descarregado” no topo. Toque aí e depois em “Instalar”.",
      "Volte a Definições → Geral → Acerca → Definições de confiança em certificados.",
      "Ligue o interruptor ao lado de “Bindnet”. Este passo é obrigatório — sem ele o certificado fica instalado mas não é confiado.",
    ],
  },
  {
    id: "windows",
    label: "Windows",
    icon: Monitor,
    steps: [
      "Descarregue o ficheiro e faça duplo clique nele.",
      "Clique em “Instalar Certificado…” e escolha “Máquina Local”.",
      "Selecione “Colocar todos os certificados no seguinte arquivo” e escolha “Autoridades de Certificação de Raiz Fidedignas”.",
      "Conclua e reinicie o navegador.",
    ],
  },
  {
    id: "linux",
    label: "Linux",
    icon: Chrome,
    steps: [
      "Descarregue o ficheiro para a sua pasta pessoal.",
      "sudo cp bindnet-local-ca.crt /usr/local/share/ca-certificates/",
      "sudo update-ca-certificates",
      "O Firefox e o Chrome têm arquivos próprios — pode ser preciso importar também nas definições do navegador.",
    ],
  },
];

// Instruções de instalação da CA local, por sistema operativo. Público:
// vive no portal cativo, sem sessão — quem acabou de se ligar ao Wi-Fi
// precisa de instalar a CA ANTES de conseguir navegar sem avisos de
// certificado, e nessa altura ainda não tem conta nenhuma.
export function PortalCaInstructions() {
  return (
    <div className="space-y-4">
      {PLATFORMS.map((platform) => {
        const Icon = platform.icon;
        return (
          <details
            key={platform.id}
            className="rounded-2xl border border-[#bfd7c8] bg-white p-4 shadow-sm transition-colors open:border-[#4d8b67] open:bg-[#f4fbf6]"
          >
            <summary className="flex cursor-pointer items-center gap-2 text-sm font-semibold marker:text-[#326b4a]">
              <Icon className="h-4 w-4 text-[#326b4a]" />
              <span className="text-[#14251d]">{platform.label}</span>
            </summary>
            <ol className="mt-4 list-decimal space-y-2.5 pl-5 text-sm leading-6 text-[#284338] marker:font-semibold marker:text-[#326b4a]">
              {platform.steps.map((step) => (
                <li key={step}>{step}</li>
              ))}
            </ol>
          </details>
        );
      })}
    </div>
  );
}
