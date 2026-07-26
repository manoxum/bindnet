import { useState } from "react";
import { Pencil, Plus, Trash2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { HotspotContentPlanDialog } from "@/components/hotspot/HotspotContentPlanDialog";
import { HotspotContentCategoriesCard } from "@/components/hotspot/HotspotContentCategoriesCard";
import { useContentPlans, useHotspotContentMutations } from "@/components/hotspot/useHotspotContent";
import type { ContentPlan } from "@/components/hotspot/hotspot-content-types";

// Aba "Conteúdo": planos de bloqueio/permissão (vinculáveis a perfil ou
// dispositivo) + catálogo de categorias. O bloqueio de domínio é feito no
// dns-provider e o de IP no firewall; aqui é só a gestão.
export function HotspotContentCard() {
  const plans = useContentPlans();
  const { deletePlan } = useHotspotContentMutations();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<ContentPlan | null>(null);
  const [deleting, setDeleting] = useState<ContentPlan | null>(null);

  const openCreate = () => {
    setEditing(null);
    setDialogOpen(true);
  };
  const openEdit = (plan: ContentPlan) => {
    setEditing(plan);
    setDialogOpen(true);
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <CardTitle>Planos de conteúdo</CardTitle>
            <Button onClick={openCreate} className="gap-2">
              <Plus className="h-4 w-4" />
              Novo plano
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          <div className="overflow-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Nome</TableHead>
                  <TableHead className="hidden sm:table-cell">Política padrão</TableHead>
                  <TableHead className="text-right">Ações</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {(plans.data ?? []).map((plan) => (
                  <TableRow key={plan.id}>
                    <TableCell>
                      <div className="font-medium">{plan.name}</div>
                      {plan.description && <div className="text-xs text-muted-foreground">{plan.description}</div>}
                    </TableCell>
                    <TableCell className="hidden sm:table-cell">
                      <Badge variant="outline">{plan.defaultAction === "block" ? "Lista branca" : "Lista negra"}</Badge>
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex justify-end gap-2">
                        <Button variant="outline" size="sm" onClick={() => openEdit(plan)} aria-label="Editar plano">
                          <Pencil className="h-4 w-4" />
                          <span className="hidden sm:inline">Editar</span>
                        </Button>
                        <Button variant="ghost" size="sm" onClick={() => setDeleting(plan)} aria-label="Remover plano">
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
                {(plans.data?.length ?? 0) === 0 && (
                  <TableRow>
                    <TableCell colSpan={3} className="text-center text-sm text-muted-foreground">
                      Nenhum plano ainda. Crie um e vincule a um perfil ou dispositivo.
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>

      <HotspotContentCategoriesCard />

      <HotspotContentPlanDialog key={editing?.id ?? "new"} plan={editing} open={dialogOpen} onOpenChange={setDialogOpen} />

      <ConfirmDialog
        open={!!deleting}
        onOpenChange={(open) => !open && setDeleting(null)}
        title="Remover plano"
        description={`Remover o plano "${deleting?.name}"? Perfis/dispositivos vinculados ficam sem plano.`}
        confirmLabel="Remover"
        onConfirm={() => {
          if (deleting) deletePlan.mutate(deleting.id);
          setDeleting(null);
        }}
      />
    </div>
  );
}
