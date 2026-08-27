import { useEffect, useState } from "react";
import { Copy, CopyPlus, Pencil, Trash2, X } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { SelectNative } from "@/components/ui/select-native";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import type { LocalTldConfig } from "@/components/dns/useDnsQueries";
import type { useDnsMutations } from "@/components/dns/useDnsMutations";

interface DnsLocalTldsCardProps {
  config: Record<string, string> | undefined;
  tldConfigs?: LocalTldConfig[];
  mutations: ReturnType<typeof useDnsMutations>;
  showActions?: boolean;
  onChange?: (tlds: string[]) => void;
}

const automaticTlds = (config: Record<string, string> | undefined): LocalTldConfig[] =>
  (config?.DNS_LOCAL_TLDS ?? "").split(",").map((tld) => tld.trim()).filter(Boolean).map((tld) => ({ tld, mode: "automatic" }));

const sameTlds = (left: LocalTldConfig[], right: LocalTldConfig[]) =>
  left.length === right.length && left.every((entry, index) => entry.tld === right[index].tld && entry.mode === right[index].mode && entry.address === right[index].address);

export function DnsLocalTldsCard({ config, tldConfigs, mutations, showActions = true, onChange }: DnsLocalTldsCardProps) {
  const [tlds, setTlds] = useState<LocalTldConfig[]>([]);
  const [newTld, setNewTld] = useState("");
  const [newMode, setNewMode] = useState<LocalTldConfig["mode"]>("automatic");
  const [newAddress, setNewAddress] = useState("");
  const [editingIndex, setEditingIndex] = useState<number | null>(null);
  const [editingSnapshot, setEditingSnapshot] = useState<LocalTldConfig | null>(null);
  const [editingDraft, setEditingDraft] = useState(false);
  const [removingIndex, setRemovingIndex] = useState<number | null>(null);

  useEffect(() => {
    const next = tldConfigs ?? automaticTlds(config);
    setTlds((current) => sameTlds(current, next) ? current : next);
  }, [config, tldConfigs]);
  useEffect(() => onChange?.(tlds.map(({ tld }) => tld)), [onChange, tlds]);

  function updateTld(index: number, update: Partial<LocalTldConfig>) {
    setTlds((current) => current.map((entry, entryIndex) => entryIndex === index ? { ...entry, ...update } : entry));
  }

  function startEditing(index: number, isDraft = false) {
    setEditingIndex(index);
    setEditingSnapshot({ ...tlds[index] });
    setEditingDraft(isDraft);
  }

  function stopEditing() {
    setEditingIndex(null);
    setEditingSnapshot(null);
    setEditingDraft(false);
  }

  function cancelEditing() {
    if (editingIndex === null) return;
    if (editingDraft) {
      setTlds((current) => current.filter((_, index) => index !== editingIndex));
    } else if (editingSnapshot) {
      updateTld(editingIndex, editingSnapshot);
    }
    stopEditing();
  }

  async function copyValue(value: string, label: string) {
    try {
      await navigator.clipboard.writeText(value);
      toast.success(`${label} copiado.`);
    } catch {
      toast.error(`Não foi possível copiar ${label.toLowerCase()}.`);
    }
  }

  function duplicateTld(index: number) {
    const duplicate = { ...tlds[index], tld: "" };
    const next = [...tlds, duplicate];
    setTlds(next);
    setEditingIndex(next.length - 1);
    setEditingSnapshot({ ...duplicate });
    setEditingDraft(true);
  }

  function confirmRemoveTld() {
    if (removingIndex === null) return;
    const next = tlds.filter((_, index) => index !== removingIndex);
    setTlds(next);
    if (showActions) mutations.saveLocalTlds.mutate(next, { onSuccess: () => setRemovingIndex(null) });
    else setRemovingIndex(null);
  }

  function addTld() {
    const tld = newTld.trim().toLowerCase().replace(/^[*.]+/, "").replace(/\.+$/, "");
    const label = "[a-z0-9]([a-z0-9-]*[a-z0-9])?";
    if (!tld || tlds.some((entry) => entry.tld === tld)) return;
    if (!new RegExp(`^${label}(\\.${label})*$`).test(tld)) {
      toast.error(`TLD inválido: "${tld}". Use labels válidos, como local ou local.com.`);
      return;
    }
    if (newMode === "static" && !newAddress.trim()) {
      toast.error("Informe o IPv4 estático para este TLD.");
      return;
    }
    const entry = { tld, mode: newMode, address: newMode === "static" ? newAddress.trim() : undefined } satisfies LocalTldConfig;
    const next = [...tlds, entry];
    setTlds(next);
    if (showActions) mutations.saveLocalTlds.mutate(next);
    setNewTld("");
    setNewAddress("");
    setNewMode("automatic");
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>TLDs locais</CardTitle>
        <CardDescription>Escolha por TLD entre IP automático do split-horizon ou um IPv4 fixo para todos os seus domínios.</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          {tlds.map((entry, index) => {
            const isEditing = editingIndex === index;
            return (
              <div key={index} className="grid gap-2 rounded-md border p-3 sm:grid-cols-[minmax(9rem,1fr)_13rem_12rem_auto] sm:items-center">
                <div className="relative">
                  <Input aria-label="TLD local" className="pr-9 font-mono" value={entry.tld} disabled={!isEditing} onChange={(event) => updateTld(index, { tld: event.target.value })} onBlur={(event) => updateTld(index, { tld: event.target.value.trim().toLowerCase().replace(/^[*.]+/, "").replace(/\.+$/, "") })} />
                  <Button type="button" variant="ghost" size="icon" className="absolute right-0 top-0 h-9 w-9" aria-label={`Copiar .${entry.tld}`} disabled={!entry.tld} onClick={() => copyValue(entry.tld, "TLD")}><Copy className="h-4 w-4" /></Button>
                </div>
                <SelectNative disabled={!isEditing} value={entry.mode} onChange={(event) => updateTld(index, { mode: event.target.value as LocalTldConfig["mode"], address: event.target.value === "automatic" ? undefined : entry.address })}>
                  <option value="automatic">IP local automático</option>
                  <option value="static">IP estático</option>
                </SelectNative>
                <div className="relative">
                  <Input disabled={!isEditing || entry.mode !== "static"} className="pr-9" placeholder="IPv4 estático" value={entry.address ?? ""} onChange={(event) => updateTld(index, { address: event.target.value })} />
                  <Button type="button" variant="ghost" size="icon" className="absolute right-0 top-0 h-9 w-9" aria-label={`Copiar IP de .${entry.tld}`} disabled={!entry.address} onClick={() => copyValue(entry.address ?? "", "IP")}><Copy className="h-4 w-4" /></Button>
                </div>
                <div className="flex justify-end gap-1">
                  {isEditing && showActions ? <>
                    <Button size="sm" disabled={mutations.saveLocalTlds.isPending} onClick={() => mutations.saveLocalTlds.mutate(tlds, { onSuccess: stopEditing })}>{mutations.saveLocalTlds.isPending ? "Aplicando..." : "Salvar e aplicar"}</Button>
                    <Button type="button" variant="ghost" size="icon" aria-label="Cancelar edição" disabled={mutations.saveLocalTlds.isPending} onClick={cancelEditing}><X className="h-4 w-4" /></Button>
                  </> : <Button type="button" variant="ghost" size="icon" aria-label={`Editar .${entry.tld}`} disabled={editingIndex !== null || mutations.saveLocalTlds.isPending} onClick={() => startEditing(index)}><Pencil className="h-4 w-4" /></Button>}
                  <Button type="button" variant="ghost" size="icon" aria-label={`Duplicar .${entry.tld}`} disabled={editingIndex !== null || mutations.saveLocalTlds.isPending} onClick={() => duplicateTld(index)}><CopyPlus className="h-4 w-4" /></Button>
                  <Button type="button" variant="ghost" size="icon" aria-label={`Remover .${entry.tld}`} disabled={mutations.saveLocalTlds.isPending} onClick={() => setRemovingIndex(index)}><Trash2 className="h-4 w-4" /></Button>
                </div>
              </div>
            );
          })}
        </div>
        <div className="grid gap-2 rounded-md border border-dashed p-3 sm:grid-cols-[minmax(9rem,1fr)_13rem_12rem_auto] sm:items-center">
          <Input placeholder="Novo TLD, ex.: local" value={newTld} onChange={(event) => setNewTld(event.target.value)} onKeyDown={(event) => event.key === "Enter" && (event.preventDefault(), addTld())} />
          <SelectNative value={newMode} onChange={(event) => setNewMode(event.target.value as LocalTldConfig["mode"])}><option value="automatic">IP local automático</option><option value="static">IP estático</option></SelectNative>
          <Input disabled={newMode !== "static"} placeholder="IPv4 estático" value={newAddress} onChange={(event) => setNewAddress(event.target.value)} onKeyDown={(event) => event.key === "Enter" && (event.preventDefault(), addTld())} />
          <Button type="button" variant="outline" onClick={addTld} disabled={mutations.saveLocalTlds.isPending}>{showActions ? "Adicionar e aplicar" : "Adicionar"}</Button>
        </div>
        <ConfirmDialog open={removingIndex !== null} onOpenChange={(open) => !open && setRemovingIndex(null)} title="Remover TLD local?" description={`Remover .${removingIndex === null ? "" : tlds[removingIndex]?.tld ?? ""} também apagará todos os DNS resolvidos dinamicamente por esse TLD. Registros manuais serão preservados.`} confirmLabel="Remover TLD" variant="destructive" pending={mutations.saveLocalTlds.isPending} onConfirm={confirmRemoveTld} />
      </CardContent>
    </Card>
  );
}
