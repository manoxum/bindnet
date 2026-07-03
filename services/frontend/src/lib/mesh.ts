import { useQuery } from "@tanstack/react-query";
import { Globe2, Network, Route } from "lucide-react";
import { api } from "@/lib/api";

export interface DiscoveredPeer {
  address: string;
  nodeName: string;
  source: string;
  lastSeenAt: string;
}

export interface DiscoverRoute {
  domain: string;
  owner: string;
  nextHop: string;
  distance: number;
  source: string;
  state: string;
  lastSeenAt: string;
}

export interface DiscoveredServer {
  name: string;
  zone: string;
  source: string;
  kind: string;
  file?: string;
}

export type BindnetNodeKind = "local" | "direct" | "inferred";

export interface BindnetNode {
  id: string;
  name: string;
  address: string;
  kind: BindnetNodeKind;
  source: string;
  lastSeenAt?: string;
}

export interface MeshData {
  config: Record<string, string>;
  peers: DiscoveredPeer[];
  routes: DiscoverRoute[];
  localServices: DiscoveredServer[];
  nodes: BindnetNode[];
}

export function splitPeers(raw?: string) {
  return (raw ?? "")
    .split(/[,\s;]+/)
    .map((peer) => peer.trim())
    .filter(Boolean);
}

export function joinPeers(peers: string[]) {
  return peers.join(",");
}

export function normalizePeerAddress(value: string) {
  return value.trim().replace(/^https?:\/\//, "").replace(/\/+$/, "");
}

export async function unlinkPeerAddress(config: Record<string, string> | undefined, address: string) {
  const target = normalizePeerAddress(address);
  const peers = splitPeers(config?.DISCOVER_PEERS);
  const remaining = peers.filter((peer) => normalizePeerAddress(peer) !== target);

  if (!target || remaining.length === peers.length) {
    throw new Error("Este Bindnet não está vinculado manualmente.");
  }

  await api.patch("/dns/config", { DISCOVER_PEERS: joinPeers(remaining) });
  await api.post("/dns/apply");
}

export function peerHost(address: string) {
  return address.includes(":") ? address.split(":")[0] : address;
}

export function nodePath(id: string) {
  return `/bindnets/${encodeURIComponent(id)}`;
}

export function buildNodes(config: Record<string, string>, routes: DiscoverRoute[]): BindnetNode[] {
  const localName = config.DISCOVER_NODE_NAME?.trim() || "este-servidor";
  const nodes = new Map<string, BindnetNode>();

  nodes.set("local", {
    id: "local",
    name: localName,
    address: "local",
    kind: "local",
    source: "self",
  });

  for (const address of splitPeers(config.DISCOVER_PEERS)) {
    nodes.set(`peer:${address}`, {
      id: `peer:${address}`,
      name: address,
      address,
      kind: "direct",
      source: "manual",
    });
  }

  for (const route of routes) {
    if (!route.owner || route.owner === localName) continue;
    const sourceMatch = [...nodes.values()].find(
      (node) => node.kind === "direct" && peerHost(node.address) === peerHost(route.source),
    );
    if (sourceMatch && sourceMatch.name === route.owner) continue;
    const id = `owner:${route.owner}`;
    if (!nodes.has(id)) {
      nodes.set(id, {
        id,
        name: route.owner,
        address: route.nextHop ? `via ${route.nextHop}` : "rota aprendida",
        kind: "inferred",
        source: route.distance <= 1 ? "rota direta" : "rota indireta",
        lastSeenAt: route.lastSeenAt,
      });
    }
  }

  return [...nodes.values()].sort((a, b) => {
    const order = { local: 0, direct: 1, inferred: 2 };
    return order[a.kind] - order[b.kind] || a.name.localeCompare(b.name);
  });
}

export function useMeshData() {
  return useQuery<MeshData>({
    queryKey: ["bindnets", "mesh"],
    queryFn: async () => {
      const [config, peers, routes, localServices] = await Promise.all([
        api.get<Record<string, string>>("/dns/config"),
        api.get<DiscoveredPeer[]>("/dns/peers"),
        api.get<DiscoverRoute[]>("/dns/routes"),
        api.get<DiscoveredServer[]>("/dns/discovered-servers"),
      ]);
      return { config, peers, routes, localServices, nodes: buildNodes(config, routes) };
    },
    refetchInterval: 10000,
  });
}

export function nodeTone(kind: BindnetNodeKind) {
  if (kind === "local") return "border-emerald-500/40 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300";
  if (kind === "direct") return "border-sky-500/40 bg-sky-500/10 text-sky-700 dark:text-sky-300";
  return "border-amber-500/40 bg-amber-500/10 text-amber-700 dark:text-amber-300";
}

export function nodeLabel(kind: BindnetNodeKind) {
  if (kind === "local") return "local";
  if (kind === "direct") return "direto";
  return "indireto";
}

export function formatSeen(value?: string) {
  if (!value) return "sem leitura";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
}

export function serviceRowsForNode(node: BindnetNode, data: MeshData) {
  if (node.kind === "local") {
    return data.localServices.map((service) => ({
      name: service.name,
      detail: service.kind,
      state: "local",
      via: "nginx-ui",
    }));
  }

  const routes = data.routes.filter((route) => {
    if (route.owner === node.name) return true;
    return node.kind === "direct" && peerHost(route.source) === peerHost(node.address) && route.distance === 1;
  });

  return routes.map((route) => ({
    name: route.domain,
    detail: `${route.distance} salto${route.distance === 1 ? "" : "s"}`,
    state: route.state,
    via: route.nextHop || route.source,
  }));
}

export function routesViaNode(node: BindnetNode, routes: DiscoverRoute[]) {
  if (node.kind !== "direct") return [];
  return routes.filter((route) => peerHost(route.source) === peerHost(node.address) && route.owner !== node.name);
}

export function neighborRows(node: BindnetNode, data: MeshData) {
  if (node.kind === "local") {
    return data.nodes.filter((candidate) => candidate.kind === "direct");
  }

  const indirectNames = new Set(routesViaNode(node, data.routes).map((route) => route.owner).filter(Boolean));
  return data.nodes.filter((candidate) => candidate.id === "local" || indirectNames.has(candidate.name));
}

export function metricCards(node: BindnetNode, data: MeshData) {
  const services = serviceRowsForNode(node, data).length;
  const neighbors = neighborRows(node, data).length;
  const routed = node.kind === "direct" ? routesViaNode(node, data.routes).length : data.routes.length;
  return [
    { label: "Serviços", value: services, icon: Globe2 },
    { label: "Vizinhos", value: neighbors, icon: Network },
    { label: "Rotas", value: routed, icon: Route },
  ];
}
