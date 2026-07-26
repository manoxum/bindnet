import { useState } from "react";
import { Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { SelectNative } from "@/components/ui/select-native";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { useContentCategories, useContentPlan, useHotspotContentMutations } from "@/components/hotspot/useHotspotContent";
import {
  CONTENT_KIND_LABELS,
  type ContentAction,
  type ContentPlan,
  type ContentRuleKind,
} from "@/components/hotspot/hotspot-content-types";

interface Props {
  plan: ContentPlan | null; // null = criar
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

// Dialog de plano de conteudo: no modo criar mostra so os metadados; no
// modo editar mostra metadados + a tabela de regras (dominio/ip/categoria,
// permitir/bloquear) com formulario de adicao.
export function HotspotContentPlanDialog({ plan, open, onOpenChange }: Props) {
  const isEdit = !!plan;
  const [name, setName] = useState(plan?.name ?? "");
  const [description, setDescription] = useState(plan?.description ?? "");
  const [defaultAction, setDefaultAction] = useState<ContentAction>(plan?.defaultAction ?? "allow");

  const detail = useContentPlan(isEdit ? plan!.id : null);
  const categories = useContentCategories();
  const { createPlan, updatePlan, addRule, removeRule } = useHotspotContentMutations();

  const [kind, setKind] = useState<ContentRuleKind>("domain");
  const [value, setValue] = useState("");
  const [action, setAction] = useState<ContentAction>("block");

  const saveMeta = () => {
    const req = { name: name.trim(), description, defaultAction };
    if (isEdit) {
      updatePlan.mutate({ id: plan!.id, req });
    } else {
      createPlan.mutate(req, { onSuccess: () => onOpenChange(false) });
    }
  };

  const submitRule = () => {
    if (!isEdit) return;
    const ruleValue = kind === "category" ? value : value.trim();
    if (!ruleValue) return;
    addRule.mutate({ planId: plan!.id, req: { kind, value: ruleValue, action } }, { onSuccess: () => setValue("") });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{isEdit ? "Editar plano" : "Novo plano"}</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="planName">Nome</Label>
              <Input id="planName" value={name} onChange={(e) => setName(e.target.value)} placeholder="ex.: Criança" />
            </div>
            <div className="space-y-2">
              <Label htmlFor="planDefault">Política padrão</Label>
              <SelectNative id="planDefault" value={defaultAction} onChange={(e) => setDefaultAction(e.target.value as ContentAction)}>
                <option value="allow">Liberar tudo, exceto o bloqueado</option>
                <option value="block">Bloquear tudo, exceto o permitido</option>
              </SelectNative>
            </div>
          </div>
          <div className="space-y-2">
            <Label htmlFor="planDesc">Descrição</Label>
            <Textarea id="planDesc" value={description} onChange={(e) => setDescription(e.target.value)} rows={2} />
          </div>
          <Button onClick={saveMeta} disabled={!name.trim() || createPlan.isPending || updatePlan.isPending}>
            {isEdit ? "Salvar" : "Criar"}
          </Button>
        </div>

        {isEdit && (
          <div className="space-y-3 border-t pt-4">
            <Label>Regras</Label>
            <div className="overflow-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Tipo</TableHead>
                    <TableHead>Valor</TableHead>
                    <TableHead>Ação</TableHead>
                    <TableHead className="text-right">—</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(detail.data?.rules ?? []).map((rule) => (
                    <TableRow key={rule.id}>
                      <TableCell>{CONTENT_KIND_LABELS[rule.kind]}</TableCell>
                      <TableCell className="break-all">{rule.value}</TableCell>
                      <TableCell>{rule.action === "block" ? "Bloquear" : "Permitir"}</TableCell>
                      <TableCell className="text-right">
                        <Button variant="ghost" size="sm" onClick={() => removeRule.mutate(rule.id)} aria-label="Remover regra">
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                  {(detail.data?.rules?.length ?? 0) === 0 && (
                    <TableRow>
                      <TableCell colSpan={4} className="text-center text-sm text-muted-foreground">
                        Nenhuma regra ainda.
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </div>

            <div className="flex flex-col gap-2 sm:flex-row">
              <SelectNative value={kind} onChange={(e) => setKind(e.target.value as ContentRuleKind)} className="sm:w-32">
                <option value="domain">Domínio</option>
                <option value="ip">IP/CIDR</option>
                <option value="category">Categoria</option>
              </SelectNative>
              {kind === "category" ? (
                <SelectNative value={value} onChange={(e) => setValue(e.target.value)} className="flex-1">
                  <option value="">Selecione…</option>
                  {(categories.data ?? []).map((c) => (
                    <option key={c.slug} value={c.slug}>
                      {c.name}
                    </option>
                  ))}
                </SelectNative>
              ) : (
                <Input
                  value={value}
                  onChange={(e) => setValue(e.target.value)}
                  placeholder={kind === "ip" ? "ex.: 203.0.113.0/24" : "ex.: exemplo.com"}
                  className="flex-1"
                />
              )}
              <SelectNative value={action} onChange={(e) => setAction(e.target.value as ContentAction)} className="sm:w-32">
                <option value="block">Bloquear</option>
                <option value="allow">Permitir</option>
              </SelectNative>
              <Button onClick={submitRule} disabled={!value || addRule.isPending}>
                Adicionar
              </Button>
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
