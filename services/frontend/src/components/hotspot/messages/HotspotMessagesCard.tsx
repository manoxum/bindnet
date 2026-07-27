import { useState } from "react";
import { Plus, Trash2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { HotspotMessageDialog } from "@/components/hotspot/messages/HotspotMessageDialog";
import { useHotspotMessageMutations, useHotspotMessages } from "@/components/hotspot/messages/useHotspotMessages";

// Aba "Avisos": lista os avisos ativos e permite enviar um novo
// (broadcast ou direcionado) ou remover um existente. O envio direcionado
// a um único dispositivo também está disponível na ficha do dispositivo
// (HotspotDeviceDetail), reusando o mesmo HotspotMessageDialog.
export function HotspotMessagesCard() {
  const messages = useHotspotMessages();
  const { remove } = useHotspotMessageMutations();
  const [creating, setCreating] = useState(false);
  const [removingId, setRemovingId] = useState<string | null>(null);

  return (
    <Card>
      <CardHeader className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <CardTitle>Avisos aos dispositivos</CardTitle>
          <CardDescription>
            Mensagens que aparecem no portal do dispositivo; urgentes também forçam o balão "Entrar na rede".
          </CardDescription>
        </div>
        <Button size="sm" onClick={() => setCreating(true)}>
          <Plus className="h-4 w-4" />
          <span className="hidden sm:inline">Novo aviso</span>
        </Button>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Mensagem</TableHead>
              <TableHead className="hidden sm:table-cell">Alvo</TableHead>
              <TableHead>Tipo</TableHead>
              <TableHead className="hidden md:table-cell">Enviado em</TableHead>
              <TableHead className="text-right">Ações</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {(messages.data ?? []).map((message) => (
              <TableRow key={message.id}>
                <TableCell className="max-w-[16rem]">
                  {message.title && <p className="font-medium">{message.title}</p>}
                  <p className="truncate text-sm text-muted-foreground">{message.body}</p>
                </TableCell>
                <TableCell className="hidden text-sm sm:table-cell">
                  {message.targetMac ? <span className="font-mono text-xs">{message.targetMac}</span> : "Todos"}
                </TableCell>
                <TableCell>
                  {message.urgent ? <Badge variant="destructive">Urgente</Badge> : <Badge variant="secondary">Aviso</Badge>}
                </TableCell>
                <TableCell className="hidden text-sm text-muted-foreground md:table-cell">
                  {new Date(message.createdAt).toLocaleString()}
                </TableCell>
                <TableCell className="text-right">
                  <Button
                    variant="ghost"
                    size="icon"
                    aria-label="Remover aviso"
                    onClick={() => setRemovingId(message.id)}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </TableCell>
              </TableRow>
            ))}
            {messages.data?.length === 0 && (
              <TableRow>
                <TableCell colSpan={5} className="py-8 text-center text-sm text-muted-foreground">
                  Nenhum aviso ativo.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </CardContent>

      <HotspotMessageDialog open={creating} onOpenChange={setCreating} />

      <ConfirmDialog
        open={removingId !== null}
        onOpenChange={(open) => !open && setRemovingId(null)}
        title="Remover aviso?"
        description="O aviso deixa de aparecer para os dispositivos."
        variant="destructive"
        confirmLabel="Remover"
        pending={remove.isPending}
        onConfirm={() => {
          if (removingId) remove.mutate(removingId, { onSuccess: () => setRemovingId(null) });
        }}
      />
    </Card>
  );
}
