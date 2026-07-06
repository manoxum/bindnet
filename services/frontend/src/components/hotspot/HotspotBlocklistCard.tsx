import { Undo2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import type { HotspotBlockMode } from "@/components/hotspot/useHotspotMutations";

export interface HotspotBlockedDevice {
  macAddress: string;
  note?: string;
  mode: HotspotBlockMode;
  blockedAt: string;
}

const MODE_LABELS: Record<HotspotBlockMode, string> = {
  deauth: "Desconectado do Wi-Fi",
  traffic: "Só tráfego cortado",
};

interface HotspotBlocklistCardProps {
  devices: HotspotBlockedDevice[];
  unblockPendingMac?: string;
  onUnblock: (mac: string) => void;
}

export function HotspotBlocklistCard({ devices, unblockPendingMac, onUnblock }: HotspotBlocklistCardProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Clientes bloqueados ({devices.length})</CardTitle>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>MAC</TableHead>
              <TableHead>Nota</TableHead>
              <TableHead>Modo</TableHead>
              <TableHead>Bloqueado em</TableHead>
              <TableHead className="text-right">Ações</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {devices.map((device) => (
              <TableRow key={device.macAddress}>
                <TableCell className="font-mono text-xs">{device.macAddress}</TableCell>
                <TableCell>{device.note || "sem nota"}</TableCell>
                <TableCell>
                  <Badge variant={device.mode === "traffic" ? "outline" : "destructive"}>
                    {MODE_LABELS[device.mode]}
                  </Badge>
                </TableCell>
                <TableCell>{new Date(device.blockedAt).toLocaleString()}</TableCell>
                <TableCell>
                  <div className="flex justify-end">
                    <Button
                      variant="secondary"
                      size="sm"
                      disabled={unblockPendingMac === device.macAddress}
                      onClick={() => onUnblock(device.macAddress)}
                    >
                      <Undo2 className="h-4 w-4" />
                      Desbloquear
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))}
            {devices.length === 0 && (
              <TableRow>
                <TableCell colSpan={5} className="text-center text-muted-foreground">
                  Nenhum MAC bloqueado.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  );
}
