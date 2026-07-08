import { useEffect, useRef, useState } from "react";
import QrScanner from "qr-scanner";
import { ExternalLink } from "lucide-react";
import { buttonVariants } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from "@/components/ui/dialog";

interface PortalVoucherQrScannerProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onScan: (code: string) => void;
}

type ScanError = "insecure-context" | "camera-unavailable" | null;

// Leitor de QR code do cartao de recarga via camera do dispositivo -
// usado como alternativa a digitar o codigo manualmente na pagina de
// autoatendimento (ver src/pages/Portal.tsx). O QR encoda so o texto
// do codigo, o mesmo que vai no campo de resgate.
//
// navigator.mediaDevices so existe em "contexto seguro" (HTTPS ou
// localhost) - o portal cativo e servido propositalmente em HTTP puro
// (RULE.md, secao do portal cativo), entao a camera nunca funciona
// aqui quando acessado pelo IP do gateway do hotspot. Detectamos isso
// antes de tentar (em vez de deixar o erro generico do getUserMedia
// aparecer) e oferecemos abrir client.bindnet.local via HTTPS num
// navegador completo - dominio com certificado da CA local, exposto
// pelo nginx-ui (ver /etc/nginx/sites-available/client.bindnet.local
// no container nginx-ui) e resolvido automaticamente pelo
// dns-provider por cair em DNS_LOCAL_TLDS (RULE.md, secao
// dns-provider). O usuario precisa aceitar manualmente o aviso de
// certificado nao confiavel uma vez; dai em diante a camera funciona.
export function PortalVoucherQrScanner({ open, onOpenChange, onScan }: PortalVoucherQrScannerProps) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const scannerRef = useRef<QrScanner | null>(null);
  const [error, setError] = useState<ScanError>(null);

  useEffect(() => {
    if (!open || !videoRef.current) return;

    if (!window.isSecureContext || !navigator.mediaDevices?.getUserMedia) {
      setError("insecure-context");
      return;
    }

    setError(null);
    const scanner = new QrScanner(
      videoRef.current,
      (result) => {
        onScan(result.data.trim().toUpperCase());
        onOpenChange(false);
      },
      { highlightScanRegion: true, highlightCodeOutline: true },
    );
    scannerRef.current = scanner;
    scanner.start().catch(() => setError("camera-unavailable"));

    return () => {
      scanner.stop();
      scanner.destroy();
      scannerRef.current = null;
    };
  }, [open, onScan, onOpenChange]);

  const secureUrl = "https://client.bindnet.local/portal";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Ler QR code do cartão de recarga</DialogTitle>
          <DialogDescription>Aponte a câmera para o QR code impresso no cartão.</DialogDescription>
        </DialogHeader>
        {error === "insecure-context" && (
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground">
              Esta conexão não permite usar a câmera. Abra a página num navegador para continuar.
            </p>
            <a href={secureUrl} target="_blank" rel="noopener noreferrer" className={buttonVariants({ className: "w-full" })}>
              <ExternalLink className="h-4 w-4" />
              Abrir no navegador
            </a>
          </div>
        )}
        {error === "camera-unavailable" && (
          <p className="text-sm text-destructive">Não foi possível acessar a câmera. Verifique a permissão do navegador.</p>
        )}
        {!error && (
          <video ref={videoRef} className="aspect-square w-full rounded-lg bg-black object-cover" muted playsInline />
        )}
      </DialogContent>
    </Dialog>
  );
}
