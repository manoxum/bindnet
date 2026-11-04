import { useState } from "react";
import { Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { TunnelPublicationForm } from "./TunnelPublicationForm";
import type { TunnelDomain, TunnelSite } from "./types";

export function TunnelPublicationDialog({
  domains,
  sites,
  createSite = false,
}: {
  domains: TunnelDomain[];
  sites: TunnelSite[];
  createSite?: boolean;
}) {
  const [open, setOpen] = useState(false);
  return (
    <>
      <Button onClick={() => setOpen(true)}>
        <Plus className="mr-2 h-4 w-4" />
        {createSite ? "Novo site" : "Novo recurso"}
      </Button>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-3xl">
          <DialogHeader>
            <DialogTitle>
              {createSite ? "Publicar um site local" : "Publicar outro serviço local"}
            </DialogTitle>
            <DialogDescription>
              {createSite
                ? "Cria site, recurso e alvo nginx, e inicia o conector Newt nesta conta."
                : "Reutiliza um conector existente para publicar outro domínio servido pelo nginx."}
            </DialogDescription>
          </DialogHeader>
          <TunnelPublicationForm
            sites={sites}
            domains={domains}
            createSite={createSite}
            onSuccess={() => setOpen(false)}
          />
        </DialogContent>
      </Dialog>
    </>
  );
}
