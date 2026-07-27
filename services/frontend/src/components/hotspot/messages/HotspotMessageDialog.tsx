import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { useHotspotMessageMutations } from "@/components/hotspot/messages/useHotspotMessages";
import {
  emptyHotspotMessageForm,
  formValuesToCreateRequest,
  hotspotMessageFormSchema,
  type HotspotMessageFormValues,
} from "@/components/hotspot/messages/hotspot-message-schema";

interface HotspotMessageDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  // Quando informado, o aviso vai so para esse dispositivo (campo de MAC
  // travado) - usado a partir da ficha do dispositivo. Sem ele, o aviso e
  // broadcast para todos os conectados.
  lockedMac?: string;
}

// Diálogo de criação de aviso, reutilizado pela aba "Avisos" (broadcast) e
// pela ficha do dispositivo (direcionado). Responsivo: DialogContent já é
// max-w-lg; só ajustamos com prefixo sm:.
export function HotspotMessageDialog({ open, onOpenChange, lockedMac }: HotspotMessageDialogProps) {
  const { create } = useHotspotMessageMutations();
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<HotspotMessageFormValues>({
    resolver: zodResolver(hotspotMessageFormSchema),
    defaultValues: { ...emptyHotspotMessageForm, targetMac: lockedMac ?? "" },
  });

  useEffect(() => {
    if (open) reset({ ...emptyHotspotMessageForm, targetMac: lockedMac ?? "" });
  }, [open, lockedMac, reset]);

  function onSubmit(values: HotspotMessageFormValues) {
    const request = formValuesToCreateRequest({ ...values, targetMac: lockedMac ?? values.targetMac });
    create.mutate(request, { onSuccess: () => onOpenChange(false) });
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Novo aviso</DialogTitle>
          <DialogDescription>
            {lockedMac
              ? `Enviar um aviso só para o dispositivo ${lockedMac}.`
              : "Enviar um aviso para todos os dispositivos conectados."}
          </DialogDescription>
        </DialogHeader>

        <form className="space-y-4" onSubmit={handleSubmit(onSubmit)}>
          <div className="space-y-2">
            <Label htmlFor="messageTitle">Título (opcional)</Label>
            <Input id="messageTitle" placeholder="ex.: Manutenção programada" {...register("title")} />
            {errors.title && <p className="text-sm text-destructive">{errors.title.message}</p>}
          </div>

          <div className="space-y-2">
            <Label htmlFor="messageBody">Mensagem</Label>
            <Textarea id="messageBody" rows={4} placeholder="Escreva o aviso…" {...register("body")} />
            {errors.body && <p className="text-sm text-destructive">{errors.body.message}</p>}
          </div>

          {!lockedMac && (
            <div className="space-y-2">
              <Label htmlFor="messageMac">MAC do dispositivo (opcional)</Label>
              <Input id="messageMac" placeholder="deixe vazio para enviar a todos" {...register("targetMac")} />
              <p className="text-xs text-muted-foreground">Vazio = broadcast para todos os conectados.</p>
            </div>
          )}

          <div className="space-y-2">
            <Label htmlFor="messageExpires">Expira em (opcional)</Label>
            <Input id="messageExpires" type="datetime-local" {...register("expiresAt")} />
          </div>

          <label className="flex items-start gap-2 text-sm">
            <input type="checkbox" className="mt-0.5 h-4 w-4 accent-primary" {...register("urgent")} />
            <span>
              <span className="font-medium">Urgente</span> — além do portal, força o balão "Entrar na rede" no
              dispositivo (só porta 80/HTTP, melhor esforço).
            </span>
          </label>

          <div className="flex flex-col gap-2 sm:flex-row sm:justify-end">
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)} disabled={create.isPending}>
              Cancelar
            </Button>
            <Button type="submit" disabled={create.isPending}>
              Enviar aviso
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
