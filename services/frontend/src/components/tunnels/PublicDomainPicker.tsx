import { useEffect, useId, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  FreeDomainOptions,
  type DomainAvailability,
} from "./FreeDomainOptions";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { SelectNative } from "@/components/ui/select-native";
import { useTunnelAccount, useTunnelApi } from "./useTunnelAccount";
import type { TunnelDomain } from "./types";

interface Props {
  domains: TunnelDomain[];
  domainId: string;
  subdomain: string;
  onDomainChange: (value: string) => void;
  onSubdomainChange: (value: string) => void;
  onReadyChange?: (ready: boolean) => void;
}

export function PublicDomainPicker({
  domains,
  domainId,
  subdomain,
  onDomainChange,
  onSubdomainChange,
  onReadyChange,
}: Props) {
  const id = useId();
  const account = useTunnelAccount();
  const api = useTunnelApi();
  const [source, setSource] = useState(
    domains.some((value) => value.domainId === domainId) ? domainId : "free",
  );
  const [debounced, setDebounced] = useState(subdomain);
  useEffect(() => {
    if (!domainId && !subdomain) setSource("free");
  }, [domainId, subdomain]);
  useEffect(() => {
    const timer = setTimeout(() => setDebounced(subdomain), 400);
    return () => clearTimeout(timer);
  }, [subdomain]);
  const valid = /^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$/.test(debounced);
  const options = useQuery({
    queryKey: ["tunnels", account, "domain-options", debounced],
    queryFn: () =>
      api.get<DomainAvailability>(
        `/tunnels/domain-options?subdomain=${encodeURIComponent(debounced)}`,
      ),
    enabled: source === "free" && valid,
    retry: false,
    staleTime: 0,
  });
  const pending = debounced !== subdomain || options.isFetching;
  const domain = domains.find((value) => value.domainId === source);
  const chosen = options.data?.options.find(
    (value) => value.domainId === domainId,
  );
  const ready =
    source === "free"
      ? valid && !pending && !options.isError && !!chosen
      : !!domain && domainId === source;
  useEffect(() => {
    onReadyChange?.(ready);
  }, [ready, onReadyChange]);
  return (
    <div className="space-y-3">
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor={`${id}-subdomain`}>Subdomínio público</Label>
          <Input
            id={`${id}-subdomain`}
            value={subdomain}
            placeholder="minha-app"
            disabled={domain?.type === "cname"}
            onChange={(event) => {
              onSubdomainChange(event.target.value.toLowerCase());
              if (source === "free") onDomainChange("");
            }}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor={`${id}-base`}>Domínio base público</Label>
          <SelectNative
            id={`${id}-base`}
            value={source}
            onChange={(event) => {
              setSource(event.target.value);
              onDomainChange(
                event.target.value === "free" ? "" : event.target.value,
              );
              if (
                domains.find((value) => value.domainId === event.target.value)
                  ?.type === "cname"
              )
                onSubdomainChange("");
            }}
          >
            <option value="free">Domínio fornecido grátis</option>
            {domains.map((value) => (
              <option key={value.domainId} value={value.domainId}>
                {value.baseDomain}
                {value.verified ? "" : " · aguardando DNS"}
              </option>
            ))}
          </SelectNative>
        </div>
      </div>
      {source === "free" ? (
        <>
          {!subdomain ? (
            <p className="text-xs text-muted-foreground">
              Digite um subdomínio para ver as opções de domínio público.
            </p>
          ) : pending ? (
            <p role="status" className="text-sm text-muted-foreground">
              Consultando domínios…
            </p>
          ) : !valid ? (
            <p role="alert" className="text-sm text-destructive">
              Use até 63 letras minúsculas, números ou hífens, sem hífen no
              início ou no fim.
            </p>
          ) : options.isError ? (
            <p role="alert" className="text-sm text-destructive">
              Não foi possível consultar domínios gratuitos:{" "}
              {options.error.message}
            </p>
          ) : (
            <>
              <FreeDomainOptions
                availability={options.data}
                domainId={domainId}
                onDomainChange={onDomainChange}
                name={`${id}-option`}
              />
            </>
          )}
        </>
      ) : (
        <p className="break-all text-sm text-muted-foreground">
          Endereço público:{" "}
          {subdomain && domain?.type !== "cname" ? `${subdomain}.` : ""}
          {domain?.baseDomain}
        </p>
      )}
      {source === "free" && chosen && !pending && (
        <p className="break-all text-xs text-primary">
          Endereço selecionado: {chosen.fullDomain}.{" "}
          {options.data?.availabilityKnown === false
            ? "Disponibilidade confirmada apenas na criação."
            : "A disponibilidade será validada novamente ao criar."}
        </p>
      )}
    </div>
  );
}
