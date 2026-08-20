import { useState } from "react";
import { Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import type { DnsRecord } from "@/components/dns/useDnsQueries";
import type { useDnsMutations } from "@/components/dns/useDnsMutations";

interface DnsRecordsCardProps {
  records: DnsRecord[];
  mutations: ReturnType<typeof useDnsMutations>;
}

export function DnsRecordsCard({ records, mutations }: DnsRecordsCardProps) {
  const { addRecord, removeRecord, clearRecords } = mutations;
  const [newHostname, setNewHostname] = useState("");
  const [newAddress, setNewAddress] = useState("");
  const [confirmClearOpen, setConfirmClearOpen] = useState(false);

  function submitAdd() {
    const hostname = newHostname.trim().toLowerCase();
    if (!hostname) return;
    addRecord.mutate({ hostname, address: newAddress.trim() || undefined }, { onSuccess: () => { setNewHostname(""); setNewAddress(""); } });
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between gap-4">
          <div>
            <CardTitle>Zonas DNS locais ({records.length})</CardTitle>
            <CardDescription>
              Cada zona cobre o próprio domínio e todos os seus subdomínios. Ex.: <span className="font-mono">empresa.local.</span>
              resolve <span className="font-mono">app.empresa.local.</span> no mesmo IPv4. O ponto final é aceito e normalizado. Sem IP, o split-horizon reserva um loopback para a view do host.
            </CardDescription>
          </div>
          <Button
            variant="outline"
            size="sm"
            disabled={records.length === 0}
            onClick={() => setConfirmClearOpen(true)}
          >
            <Trash2 className="h-4 w-4" />
            Limpar tudo
          </Button>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-2 sm:grid-cols-[1fr_12rem_auto]">
          <Input
            placeholder="Zona, ex.: empresa.local."
            value={newHostname}
            onChange={(e) => setNewHostname(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && (e.preventDefault(), submitAdd())}
          />
          <Input
            placeholder="IPv4 opcional, ex.: 192.168.1.10"
            value={newAddress}
            onChange={(e) => setNewAddress(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && (e.preventDefault(), submitAdd())}
          />
          <Button type="button" onClick={submitAdd} disabled={!newHostname.trim() || addRecord.isPending}>
            Adicionar
          </Button>
        </div>

        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Zona</TableHead>
              <TableHead>Endereço</TableHead>
              <TableHead className="hidden md:table-cell">Tipo</TableHead>
              <TableHead className="hidden sm:table-cell">Criado em</TableHead>
              <TableHead className="text-right">Ações</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {records.map((record) => (
              <TableRow key={record.hostname}>
                <TableCell className="font-mono text-xs">{record.hostname}</TableCell>
                <TableCell className="font-mono text-xs">{record.address}</TableCell>
                <TableCell className="hidden md:table-cell">{record.configuredAddress ? "IP configurado" : "Loopback automático"}</TableCell>
                <TableCell className="hidden sm:table-cell">{new Date(record.createdAt).toLocaleString()}</TableCell>
                <TableCell>
                  <div className="flex justify-end">
                    <Button
                      variant="secondary"
                      size="sm"
                      aria-label="Remover"
                      disabled={removeRecord.isPending && removeRecord.variables === record.hostname}
                      onClick={() => removeRecord.mutate(record.hostname)}
                    >
                      <Trash2 className="h-4 w-4" />
                      <span className="hidden sm:inline">Remover</span>
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))}
            {records.length === 0 && (
              <TableRow>
                <TableCell colSpan={5} className="text-center text-muted-foreground">
                  Nenhuma zona configurada ainda.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </CardContent>

      <ConfirmDialog
        open={confirmClearOpen}
        onOpenChange={setConfirmClearOpen}
        title="Limpar todos os registros?"
        description="Todos os hostnames resolvidos perdem o IP de loopback reservado. Eles ganham um novo IP na próxima consulta."
        confirmLabel="Limpar tudo"
        variant="destructive"
        pending={clearRecords.isPending}
        onConfirm={() => clearRecords.mutate(undefined, { onSuccess: () => setConfirmClearOpen(false) })}
      />
    </Card>
  );
}
