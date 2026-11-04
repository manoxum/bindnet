import { TunnelSiteCard } from "./TunnelSiteCard";
import { TunnelPublicationDialog } from "./TunnelPublicationDialog";
import type { TunnelDomain, TunnelSite } from "./types";

export function TunnelSites({ sites, domains }: { sites: TunnelSite[]; domains: TunnelDomain[] }) {
  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 className="text-lg font-semibold">Sites e conectores</h2>
          <p className="text-sm text-muted-foreground">Domínios locais publicados por esta conta Pangolin.</p>
        </div>
        <TunnelPublicationDialog domains={domains} sites={sites} createSite />
      </div>
      {sites.length === 0 ? (
        <p className="py-8 text-center text-sm text-muted-foreground">
          Nenhum site. Publique o primeiro domínio local para conectar a sua aplicação.
        </p>
      ) : (
        <div className="grid gap-4 lg:grid-cols-2">
          {sites.map((site) => (
            <TunnelSiteCard key={site.siteId} site={site} />
          ))}
        </div>
      )}
    </div>
  );
}
