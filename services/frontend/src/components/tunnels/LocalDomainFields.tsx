import { useId } from "react";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { DiscoveredServer } from "@/lib/mesh";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { SelectNative } from "@/components/ui/select-native";

export function LocalDomainFields({
  localDomain,
  method,
  onDomainChange,
  onMethodChange,
}: {
  localDomain: string;
  method: string;
  onDomainChange: (value: string) => void;
  onMethodChange: (value: "http" | "https") => void;
}) {
  const id = useId();
  const localServers = useQuery({
    queryKey: ["dns", "discovered-servers"],
    queryFn: () => api.get<DiscoveredServer[]>("/dns/discovered-servers"),
  });
  return (
    <div className="grid gap-4 sm:grid-cols-2">
      <div className="space-y-2">
        <Label htmlFor={`${id}-local`}>Domínio DNS local no nginx</Label>
        <Input
          id={`${id}-local`}
          list={`${id}-servers`}
          value={localDomain}
          placeholder="app.empresa.local"
          onChange={(event) => onDomainChange(event.target.value)}
        />
        <datalist id={`${id}-servers`}>
          {localServers.data
            ?.filter((value) => value.kind === "host")
            .map((value) => (
              <option key={value.name} value={value.name} />
            ))}
        </datalist>
        <p className="text-xs text-muted-foreground">
          O server_name já configurado no nginx integrado, sem esquema, porta ou caminho.
        </p>
      </div>
      <div className="space-y-2">
        <Label htmlFor={`${id}-method`}>Conexão ao nginx local</Label>
        <SelectNative
          id={`${id}-method`}
          value={method}
          onChange={(event) => onMethodChange(event.target.value === "http" ? "http" : "https")}
        >
          <option value="https">HTTPS (porta 443)</option>
          <option value="http">HTTP (porta 80)</option>
        </SelectNative>
      </div>
      {localServers.isError && (
        <p className="text-xs text-muted-foreground sm:col-span-2">
          Sugestões indisponíveis. Pode informar manualmente o domínio local.
        </p>
      )}
    </div>
  );
}
