import { QRCodeSVG } from "qrcode.react";
import { Wifi } from "lucide-react";

interface HotspotWifiQrProps {
  ssid: string;
  password: string;
}

function wifiQrValue(ssid: string, password: string) {
  const escape = (value: string) => value.replace(/([\\;,:"])/g, "\\$1");
  return `WIFI:T:WPA;S:${escape(ssid)};P:${escape(password)};;`;
}

// Cartão do QR de conexão Wi-Fi: moldura em degradê na cor da marca em volta
// de um miolo branco (o branco é o que garante boa leitura pela câmera em
// qualquer tema, então nunca deve seguir o fundo escuro do dark mode).
export function HotspotWifiQr({ ssid, password }: HotspotWifiQrProps) {
  return (
    <div className="flex h-full shrink-0 flex-col items-center gap-3 rounded-2xl bg-gradient-to-br from-primary/30 via-primary/10 to-transparent p-[3px] shadow-elevated transition-transform duration-300 hover:scale-[1.02]">
      <div className="flex h-full flex-col items-center justify-center gap-4 rounded-[calc(1rem-1px)] bg-white px-6 py-5">
        <div className="relative">
          <QRCodeSVG value={wifiQrValue(ssid, password)} size={176} fgColor="#065f46" bgColor="#ffffff" level="M" />
          <div className="absolute inset-0 flex items-center justify-center">
            <div className="flex h-10 w-10 items-center justify-center rounded-full border-2 border-white bg-emerald-600 shadow-sm">
              <Wifi className="h-5 w-5 text-white" />
            </div>
          </div>
        </div>
        <p className="text-xs font-medium text-emerald-700">Escaneie para conectar</p>
      </div>
    </div>
  );
}
