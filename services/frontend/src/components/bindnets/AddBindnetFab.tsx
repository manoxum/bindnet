import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Check, Plus, Radar, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { api, ApiError } from "@/lib/api";
import { cn } from "@/lib/utils";
import { type DiscoveredPeer, joinPeers, normalizePeerAddress, remoteRouteMode, splitPeers } from "@/lib/mesh";
import { EmptyState } from "@/components/bindnets/EmptyState";

export function AddBindnetFab({ config }: { config?: Record<string, string> }) {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);
  const [searchOpen, setSearchOpen] = useState(false);
  const [address, setAddress] = useState("");
  const [remoteMode, setRemoteMode] = useState<"auto" | "manual">("auto");
  const [remotePeers, setRemotePeers] = useState("");

  const refreshSearch = async () => {
    setMenuOpen(false);
    setSearchOpen(true);
    await scanPeers.mutateAsync();
  };

  const discovered = useQuery<DiscoveredPeer[]>({
    queryKey: ["dns", "peers"],
    queryFn: () => api.get<DiscoveredPeer[]>("/dns/peers"),
    enabled: searchOpen,
  });

  const scanPeers = useMutation({
    mutationFn: () => api.post<DiscoveredPeer[]>("/dns/peers/search"),
    onSuccess: (peers) => {
      queryClient.setQueryData(["dns", "peers"], peers);
      queryClient.invalidateQueries({ queryKey: ["bindnets", "mesh"] });
      queryClient.invalidateQueries({ queryKey: ["dns", "routes"] });
      toast.success("Busca de Bindnets concluída.");
    },
    onError: (error) => {
      const message = error instanceof ApiError || error instanceof Error ? error.message : "Falha ao buscar Bindnets";
      toast.error(message);
    },
  });

  const currentPeers = splitPeers(config?.DISCOVER_CONFIGURED_PEERS).map(normalizePeerAddress);
  const candidates =
    discovered.data?.filter((peer) => !currentPeers.includes(normalizePeerAddress(peer.address))) ?? [];

  const addAddresses = async (peer: string, extras: string[] = []) => {
    const normalized = normalizePeerAddress(peer);
    if (!normalized) return;
    const peers = splitPeers(config?.DISCOVER_CONFIGURED_PEERS);
    const next = [...peers];
    for (const candidate of [normalized, ...extras]) {
      const item = normalizePeerAddress(candidate);
      if (!item || next.includes(item)) continue;
      next.push(item);
    }
    await api.patch("/dns/config", { DISCOVER_CONFIGURED_PEERS: joinPeers(next), DISCOVER_REMOTE_ROUTES: remoteMode });
    await api.post("/dns/apply");
  };

  const addPeer = useMutation({
    mutationFn: async () => {
      const peer = normalizePeerAddress(address);
      if (!peer) throw new Error("Informe o endereço do Bindnet.");
      if (!peer.includes(":")) throw new Error("Use o formato host:porta.");

      if (currentPeers.includes(peer)) throw new Error("Este Bindnet já está na lista.");
      const extras = remoteMode === "manual" ? splitPeers(remotePeers).map(normalizePeerAddress) : [];
      const invalidExtra = extras.find((item) => !item.includes(":"));
      if (invalidExtra) throw new Error("Use host:porta em todos os vizinhos remotos.");
      await addAddresses(peer, extras);
    },
    onSuccess: () => {
      toast.success("Bindnet adicionado.");
      setAddress("");
      setRemoteMode("auto");
      setRemotePeers("");
      setOpen(false);
      setSearchOpen(false);
      queryClient.invalidateQueries({ queryKey: ["dns", "config"] });
      queryClient.invalidateQueries({ queryKey: ["bindnets", "mesh"] });
    },
    onError: (error) => {
      const message = error instanceof ApiError || error instanceof Error ? error.message : "Falha ao adicionar Bindnet";
      toast.error(message);
    },
  });

  const acceptCandidate = useMutation({
    mutationFn: async (peer: string) => {
      setAddress(peer);
      setRemoteMode(remoteRouteMode(config));
      setSearchOpen(false);
      setOpen(true);
    },
  });

  const openManualForm = () => {
    setMenuOpen(false);
    setAddress("");
    setRemoteMode(remoteRouteMode(config));
    setRemotePeers("");
    setOpen(true);
  };

  return (
    <>
      <div className="fixed bottom-6 right-6 z-30 flex flex-col items-end gap-3">
        {menuOpen && (
          <div className="w-56 rounded-lg border bg-card p-2 shadow-xl">
            <button
              type="button"
              onClick={openManualForm}
              className="flex w-full items-center gap-2 rounded-md px-3 py-2 text-sm transition hover:bg-accent"
            >
              <Plus className="h-4 w-4" />
              Adicionar manualmente
            </button>
            <button
              type="button"
              onClick={() => void refreshSearch()}
              className="flex w-full items-center gap-2 rounded-md px-3 py-2 text-sm transition hover:bg-accent"
              disabled={scanPeers.isPending}
            >
              <Radar className="h-4 w-4" />
              {scanPeers.isPending ? "Buscando..." : "Fazer busca"}
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
                <p className="mt-1 text-sm text-muted-foreground">Peer direto e vizinhos remotos.</p>
              </div>
              <Button variant="ghost" size="icon" onClick={() => setOpen(false)} aria-label="Fechar">
                <X className="h-4 w-4" />
              </Button>
            </div>
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="bindnet-peer">Peer direto</Label>
                <Input
                  id="bindnet-peer"
                  autoFocus
                  placeholder="10.0.0.2:8531"
                  value={address}
                  onChange={(event) => setAddress(event.target.value)}
                  onKeyDown={(event) => {
                    if (event.key === "Enter" && remoteMode === "auto") {
                      event.preventDefault();
                      addPeer.mutate();
                    }
                  }}
                />
              </div>

              <div className="grid gap-2">
                <Label>Vizinhos remotos</Label>
                <div className="grid gap-2 sm:grid-cols-2">
                  {[
                    ["auto", "Pegar automaticamente"],
                    ["manual", "Adicionar à mão"],
                  ].map(([value, label]) => {
                    const active = remoteMode === value;
                    return (
                      <button
                        key={value}
                        type="button"
                        onClick={() => setRemoteMode(value as "auto" | "manual")}
                        className={cn(
                          "flex min-h-10 items-center justify-between rounded-md border px-3 py-2 text-left text-sm transition",
                          active ? "border-primary bg-primary/10 text-primary" : "hover:bg-accent",
                        )}
                      >
                        <span>{label}</span>
                        {active && <Check className="h-4 w-4" />}
                      </button>
                    );
                  })}
                </div>
              </div>

              {remoteMode === "manual" && (
                <div className="space-y-2">
                  <Label htmlFor="bindnet-remote-peers">Vizinhos remotos</Label>
                  <textarea
                    id="bindnet-remote-peers"
                    className="min-h-24 w-full resize-y rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                    placeholder="10.0.1.2:8531, 10.0.2.2:8531"
                    value={remotePeers}
                    onChange={(event) => setRemotePeers(event.target.value)}
                  />
                </div>
              )}
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
                <p className="mt-1 text-sm text-muted-foreground">Selecione um peer para configurar.</p>
              </div>
              <Button variant="ghost" size="icon" onClick={() => setSearchOpen(false)} aria-label="Fechar">
                <X className="h-4 w-4" />
              </Button>
            </div>

            {scanPeers.isPending || discovered.isLoading ? (
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
                      {peer.domains && peer.domains.length > 0 && (
                        <p className="truncate text-xs text-muted-foreground">{peer.domains.join(", ")}</p>
                      )}
                      {peer.fingerprint && (
                        <p className="truncate text-xs text-muted-foreground">fingerprint: {peer.fingerprint}</p>
                      )}
                    </div>
                    <Button
                      size="sm"
                      onClick={() => acceptCandidate.mutate(peer.address)}
                      disabled={acceptCandidate.isPending}
                    >
                      Selecionar
                    </Button>
                  </div>
                ))}
              </div>
            )}
            <div className="mt-5 flex justify-end">
              <Button variant="outline" onClick={() => void scanPeers.mutateAsync()} disabled={scanPeers.isPending}>
                <Radar className="h-4 w-4" />
                {scanPeers.isPending ? "Buscando..." : "Buscar novamente"}
              </Button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
