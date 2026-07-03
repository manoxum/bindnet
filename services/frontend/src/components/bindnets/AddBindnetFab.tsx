import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Plus, Radar, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { api, ApiError } from "@/lib/api";
import { cn } from "@/lib/utils";
import { type DiscoveredPeer, joinPeers, normalizePeerAddress, splitPeers } from "@/lib/mesh";
import { EmptyState } from "@/components/bindnets/EmptyState";

export function AddBindnetFab({ config }: { config?: Record<string, string> }) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);
  const [searchOpen, setSearchOpen] = useState(false);
  const [address, setAddress] = useState("");

  const refreshSearch = async () => {
    setMenuOpen(false);
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ["bindnets", "mesh"] }),
      queryClient.invalidateQueries({ queryKey: ["dns", "peers"] }),
      queryClient.invalidateQueries({ queryKey: ["dns", "routes"] }),
    ]);
    setSearchOpen(true);
  };

  const discovered = useQuery<DiscoveredPeer[]>({
    queryKey: ["dns", "peers"],
    queryFn: () => api.get<DiscoveredPeer[]>("/dns/peers"),
    refetchInterval: searchOpen ? 5000 : false,
  });

  const currentPeers = splitPeers(config?.DISCOVER_PEERS);
  const candidates =
    discovered.data?.filter((peer) => !currentPeers.includes(peer.address)) ?? [];

  const addAddress = async (peer: string) => {
    const normalized = normalizePeerAddress(peer);
    const peers = splitPeers(config?.DISCOVER_PEERS);
    if (!normalized || peers.includes(normalized)) return;
    await api.patch("/dns/config", { DISCOVER_PEERS: joinPeers([...peers, normalized]) });
    await api.post("/dns/apply");
  };

  const addPeer = useMutation({
    mutationFn: async () => {
      const peer = normalizePeerAddress(address);
      if (!peer) throw new Error("Informe o endereço do Bindnet.");
      if (!peer.includes(":")) throw new Error("Use o formato host:porta.");

      if (currentPeers.includes(peer)) throw new Error("Este Bindnet já está na lista.");
      await addAddress(peer);
    },
    onSuccess: () => {
      toast.success("Bindnet adicionado.");
      setAddress("");
      setOpen(false);
      queryClient.invalidateQueries({ queryKey: ["dns", "config"] });
      queryClient.invalidateQueries({ queryKey: ["bindnets", "mesh"] });
    },
    onError: (error) => {
      const message = error instanceof ApiError || error instanceof Error ? error.message : "Falha ao adicionar Bindnet";
      toast.error(message);
    },
  });

  const acceptCandidate = useMutation({
    mutationFn: (peer: string) => addAddress(peer),
    onSuccess: () => {
      toast.success("Bindnet incorporado.");
      queryClient.invalidateQueries({ queryKey: ["dns", "config"] });
      queryClient.invalidateQueries({ queryKey: ["dns", "peers"] });
      queryClient.invalidateQueries({ queryKey: ["bindnets", "mesh"] });
    },
    onError: (error) => {
      const message = error instanceof ApiError || error instanceof Error ? error.message : "Falha ao incorporar Bindnet";
      toast.error(message);
    },
  });

  return (
    <>
      <div className="fixed bottom-6 right-6 z-30 flex flex-col items-end gap-3">
        {menuOpen && (
          <div className="w-56 rounded-lg border bg-card p-2 shadow-xl">
            <button
              type="button"
              onClick={() => {
                setMenuOpen(false);
                setOpen(true);
              }}
              className="flex w-full items-center gap-2 rounded-md px-3 py-2 text-sm transition hover:bg-accent"
            >
              <Plus className="h-4 w-4" />
              Criar um novo
            </button>
            <button
              type="button"
              onClick={() => void refreshSearch()}
              className="flex w-full items-center gap-2 rounded-md px-3 py-2 text-sm transition hover:bg-accent"
            >
              <Radar className="h-4 w-4" />
              Fazer nova busca
            </button>
          </div>
        )}
        <button
          type="button"
          onClick={() => setMenuOpen((current) => !current)}
          className="flex h-14 w-14 items-center justify-center rounded-full bg-primary text-primary-foreground shadow-lg shadow-primary/25 transition hover:-translate-y-0.5 hover:bg-primary/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          aria-label="Ações de Bindnet"
        >
          <Plus className={cn("h-6 w-6 transition", menuOpen && "rotate-45")} />
        </button>
      </div>

      {open && (
        <div className="fixed inset-0 z-40 flex items-end bg-background/70 p-4 backdrop-blur-sm sm:items-center sm:justify-center">
          <div className="w-full rounded-lg border bg-card p-5 shadow-xl sm:max-w-md">
            <div className="mb-5 flex items-start justify-between gap-4">
              <div>
                <h2 className="font-semibold">Adicionar Bindnet</h2>
                <p className="mt-1 text-sm text-muted-foreground">Endereço do endpoint de descoberta.</p>
              </div>
              <Button variant="ghost" size="icon" onClick={() => setOpen(false)} aria-label="Fechar">
                <X className="h-4 w-4" />
              </Button>
            </div>
            <div className="space-y-2">
              <Label htmlFor="bindnet-peer">Host e porta</Label>
              <Input
                id="bindnet-peer"
                autoFocus
                placeholder="10.0.0.2:8531"
                value={address}
                onChange={(event) => setAddress(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === "Enter") {
                    event.preventDefault();
                    addPeer.mutate();
                  }
                }}
              />
            </div>
            <div className="mt-5 flex justify-end gap-2">
              <Button variant="outline" onClick={() => setOpen(false)}>
                Cancelar
              </Button>
              <Button onClick={() => addPeer.mutate()} disabled={addPeer.isPending}>
                Adicionar
              </Button>
            </div>
          </div>
        </div>
      )}

      {searchOpen && (
        <div className="fixed inset-0 z-40 flex items-end bg-background/70 p-4 backdrop-blur-sm sm:items-center sm:justify-center">
          <div className="w-full rounded-lg border bg-card p-5 shadow-xl sm:max-w-xl">
            <div className="mb-5 flex items-start justify-between gap-4">
              <div>
                <h2 className="font-semibold">Bindnets encontrados</h2>
                <p className="mt-1 text-sm text-muted-foreground">Escolha quais servidores entram na malha.</p>
              </div>
              <Button variant="ghost" size="icon" onClick={() => setSearchOpen(false)} aria-label="Fechar">
                <X className="h-4 w-4" />
              </Button>
            </div>

            {discovered.isLoading ? (
              <div className="h-24 animate-pulse rounded-md border bg-muted/30" />
            ) : candidates.length === 0 ? (
              <EmptyState label="Nenhum novo Bindnet encontrado." />
            ) : (
              <div className="space-y-2">
                {candidates.map((peer) => (
                  <div key={peer.address} className="flex items-center justify-between gap-3 rounded-md border px-3 py-3">
                    <div className="min-w-0">
                      <p className="truncate text-sm font-medium">{peer.nodeName || peer.address}</p>
                      <p className="truncate text-xs text-muted-foreground">{peer.address}</p>
                    </div>
                    <Button
                      size="sm"
                      onClick={() => acceptCandidate.mutate(peer.address)}
                      disabled={acceptCandidate.isPending}
                    >
                      Adicionar
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}
    </>
  );
}
