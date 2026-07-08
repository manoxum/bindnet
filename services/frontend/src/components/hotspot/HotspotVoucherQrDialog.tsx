import { QRCodeSVG } from "qrcode.react";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";

interface HotspotVoucherQrDialogProps {
  code: string | null;
  onOpenChange: (open: boolean) => void;
}

// QR code de um unico voucher - encoda so o texto do codigo, o mesmo
// que o dispositivo digitaria manualmente no portal (ver
// PortalVoucherQrScanner.tsx, que le esse mesmo formato).
export function HotspotVoucherQrDialog({ code, onOpenChange }: HotspotVoucherQrDialogProps) {
  return (
    <Dialog open={code !== null} onOpenChange={(open) => !open && onOpenChange(false)}>
      <DialogContent className="sm:max-w-xs">
        <DialogHeader>
          <DialogTitle>QR code do voucher</DialogTitle>
        </DialogHeader>
        {code && (
          <div className="flex flex-col items-center gap-3 py-2">
            <div className="rounded-lg bg-white p-4">
              <QRCodeSVG value={code} size={192} level="M" />
            </div>
            <p className="font-mono text-sm">{code}</p>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
