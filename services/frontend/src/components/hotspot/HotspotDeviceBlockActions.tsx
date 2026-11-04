import { Ban, Undo2, WifiOff } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { HotspotBlockMode } from "@/components/hotspot/useHotspotBlockMutations";

interface HotspotDeviceBlockActionsProps {
  mac: string;
  blocked: boolean;
  disabled?: boolean;
  blockPendingMac?: string;
  unblockPendingMac?: string;
  onBlock: (mac: string, mode: HotspotBlockMode) => void;
  onUnblock: (mac: string) => void;
}

export function HotspotDeviceBlockActions({
  mac,
  blocked,
  disabled,
  blockPendingMac,
  unblockPendingMac,
  onBlock,
  onUnblock,
}: HotspotDeviceBlockActionsProps) {
  const pending = disabled || blockPendingMac === mac || unblockPendingMac === mac;

  if (blocked) {
    return (
      <Button
        variant="secondary"
        size="sm"
        aria-label="Desbloquear"
        title="Remover bloqueio manual"
        disabled={pending}
        onClick={() => onUnblock(mac)}
      >
        <Undo2 className="h-4 w-4" />
        <span className="hidden sm:inline">Desbloquear</span>
      </Button>
    );
  }

  return (
    <>
      <Button
        variant="outline"
        size="sm"
        aria-label="Cortar tráfego"
        title="Cortar tráfego sem desconectar do Wi-Fi"
        disabled={pending}
        onClick={() => onBlock(mac, "traffic")}
      >
        <WifiOff className="h-4 w-4" />
        <span className="hidden sm:inline">Cortar tráfego</span>
      </Button>
      <Button
        variant="destructive"
        size="sm"
        aria-label="Bloquear"
        title="Bloquear e desconectar do Wi-Fi"
        disabled={pending}
        onClick={() => onBlock(mac, "deauth")}
      >
        <Ban className="h-4 w-4" />
        <span className="hidden sm:inline">Bloquear</span>
      </Button>
    </>
  );
}
