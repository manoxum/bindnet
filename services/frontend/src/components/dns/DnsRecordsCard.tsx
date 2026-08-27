import { Fragment, useState } from "react";
import { Search, Trash2, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import type { DnsRecord, LocalTldConfig } from "@/components/dns/useDnsQueries";
import type { useDnsMutations } from "@/components/dns/useDnsMutations";

interface DnsRecordsCardProps {
  records: DnsRecord[];
  localTlds: LocalTldConfig[];
  mutations: ReturnType<typeof useDnsMutations>;
}

function mostSpecificTld(hostname: string, tlds: LocalTldConfig[]): string | null {
  const normalizedHostname = hostname.replace(/\.$/, "").toLowerCase();
  return tlds
    .map(({ tld }) => tld.toLowerCase())
    .filter((tld) => normalizedHostname === tld || normalizedHostname.endsWith(`.${tld}`))
    .sort((left, right) => right.length - left.length)[0] ?? null;
}

function tldForRecord(record: DnsRecord, tlds: LocalTldConfig[]) {
  return mostSpecificTld(record.hostname, tlds) ?? record.originTld;
}

export function DnsRecordsCard({ records, localTlds, mutations }: DnsRecordsCardProps) {
  const { addRecord, removeRecord, clearRecords } = mutations;
  const [newHostname, setNewHostname] = useState("");
  const [newAddress, setNewAddress] = useState("");
  const [search, setSearch] = useState("");
  const [confirmClearOpen, setConfirmClearOpen] = useState(false);

  const normalizedSearch = search.trim().toLowerCase();
  const visibleRecords = normalizedSearch
    ? records.filter((record) =>
        [record.hostname, record.address, record.configuredAddress ?? ""].some((value) => value.toLowerCase().includes(normalizedSearch)),
      )
    : records;
  const dynamicByTld = new Map<string, DnsRecord[]>();
  const manualRecords = visibleRecords.filter((record) => record.source === "manual");
  for (const record of visibleRecords.filter((record) => record.source === "dynamic")) {
    const key = tldForRecord(record, localTlds) ?? "";
    dynamicByTld.set(key, [...(dynamicByTld.get(key) ?? []), record]);
  }
  const recordGroups = [
    ...(manualRecords.length ? [{ label: "Definidos manualmente", records: manualRecords }] : []),
    ...[...dynamicByTld.entries()].sort(([left], [right]) => left.localeCompare(right)).map(([originTld, groupedRecords]) => ({
      label: originTld ? `Gerados pelo TLD .${originTld}` : "Gerados dinamicamente por outras zonas",
      records: groupedRecords,
    })),
  ];

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
            Esvaziar listagem
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

        <div className="relative max-w-xl">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            aria-label="Pesquisar zonas DNS locais"
            className="pl-9 pr-9"
            placeholder="Pesquisar por DNS ou IP"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
          />
          {search && (
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="absolute right-0 top-0 h-9 w-9"
              aria-label="Limpar pesquisa"
              onClick={() => setSearch("")}
            >
              <X className="h-4 w-4" />
            </Button>
          )}
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
            {recordGroups.map((group) => (
              <Fragment key={group.label}>
                <TableRow className="bg-muted/30 hover:bg-muted/30">
                  <TableCell colSpan={5} className="py-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">
                    {group.label} <span className="normal-case">({group.records.length})</span>
                  </TableCell>
                </TableRow>
                {group.records.map((record) => (
                  <TableRow key={record.hostname}>
                    <TableCell className="font-mono text-xs">{record.hostname}</TableCell>
                    <TableCell className="font-mono text-xs">{record.address}</TableCell>
                    <TableCell className="hidden md:table-cell">{record.source === "manual" ? "Definido manualmente" : `Dinâmico${tldForRecord(record, localTlds) ? ` · .${tldForRecord(record, localTlds)}` : ""}`}</TableCell>
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
              </Fragment>
            ))}
            {records.length === 0 && (
              <TableRow>
                <TableCell colSpan={5} className="text-center text-muted-foreground">
                  Nenhuma zona configurada ainda.
                </TableCell>
              </TableRow>
            )}
            {records.length > 0 && visibleRecords.length === 0 && (
              <TableRow>
                <TableCell colSpan={5} className="text-center text-muted-foreground">
                  Nenhuma zona corresponde a “{search}”.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </CardContent>

      <ConfirmDialog
        open={confirmClearOpen}
        onOpenChange={setConfirmClearOpen}
        title="Esvaziar registros DNS dinâmicos?"
        description="Apenas os IPs resolvidos automaticamente serão removidos. Registros definidos manualmente serão preservados."
        confirmLabel="Esvaziar listagem"
        variant="destructive"
        pending={clearRecords.isPending}
        onConfirm={() => clearRecords.mutate(undefined, { onSuccess: () => setConfirmClearOpen(false) })}
      />
    </Card>
  );
}
