import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";

export interface DomainOption {
  fullDomain: string;
  domainId: string;
  domainNamespaceId: string;
}
export interface DomainAvailability {
  available: boolean;
  availabilityKnown?: boolean;
  message?: string;
  options: DomainOption[];
}
export function FreeDomainOptions({
  availability,
  domainId,
  onDomainChange,
  name,
}: {
  availability?: DomainAvailability;
  domainId: string;
  onDomainChange: (value: string) => void;
  name: string;
}) {
  const [showAll, setShowAll] = useState(false);
  useEffect(() => {
    setShowAll(false);
  }, [availability]);
  const visible =
    (showAll ? availability?.options : availability?.options.slice(0, 3)) ?? [];
  return (
    <>
      <fieldset className="grid gap-3 sm:grid-cols-2">
        <legend className="sr-only">Opções de domínio público</legend>
        {visible.map((value) => (
          <label
            key={`${value.domainId}-${value.fullDomain}`}
            className={`flex min-w-0 cursor-pointer items-start gap-3 rounded-lg border p-3 ${domainId === value.domainId ? "border-primary bg-primary/5" : "hover:bg-muted/40"}`}
          >
            <input
              type="radio"
              className="mt-1 accent-current"
              name={name}
              value={value.domainId}
              checked={domainId === value.domainId}
              onChange={() => onDomainChange(value.domainId)}
            />
            <span className="min-w-0">
              <span className="block break-all font-mono text-sm">
                {value.fullDomain}
              </span>
              <span className="text-xs text-muted-foreground">
                Namespace: {value.domainNamespaceId}
              </span>
            </span>
          </label>
        ))}
      </fieldset>
      {availability?.message && (
        <p
          role="note"
          className="rounded-lg border bg-muted/30 p-3 text-xs leading-relaxed text-muted-foreground sm:col-span-2"
        >
          {availability.message}
        </p>
      )}
      {availability?.options.length === 0 && !availability?.message && (
        <p
          role="status"
          className="rounded-lg border p-3 text-sm text-muted-foreground"
        >
          Nenhum domínio correspondente encontrado. Tente outro subdomínio ou
          verifique as permissões e o plano desta conta.
        </p>
      )}
      {(availability?.options.length ?? 0) > 3 && (
        <Button
          type="button"
          variant="outline"
          className="w-full"
          onClick={() => setShowAll(!showAll)}
        >
          {showAll ? "Mostrar menos" : "Mostrar mais"}
        </Button>
      )}
    </>
  );
}
