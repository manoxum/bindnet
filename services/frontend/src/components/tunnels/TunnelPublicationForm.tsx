import { useState } from "react";
import { PublicationProgress } from "./PublicationProgress";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { SelectNative } from "@/components/ui/select-native";

import { useTunnelMutation } from "./useTunnels";
import { useTunnelApi } from "./useTunnelAccount";
import { PublicDomainPicker } from "./PublicDomainPicker";
import { LocalDomainFields } from "./LocalDomainFields";
import {
  publicationSchema,
  type PublicationValues,
} from "./publication-schema";
import type { TunnelDomain, TunnelSite } from "./types";

export function TunnelPublicationForm({
  domains,
  sites,
  createSite = false,
  onSuccess,
}: {
  domains: TunnelDomain[];
  sites: TunnelSite[];
  createSite?: boolean;
  onSuccess?: () => void;
}) {
  const api = useTunnelApi();
  const [startedAt, setStartedAt] = useState(0);
  const [domainReady, setDomainReady] = useState(false);
  const managed = sites.filter((value) => value.managed);
  const {
    register,
    handleSubmit,
    watch,
    setValue,
    reset,
    clearErrors,
    formState: { errors },
  } = useForm<PublicationValues>({
    resolver: zodResolver(publicationSchema),
    defaultValues: {
      name: "",
      domainId: "",
      subdomain: "",
      localDomain: "",
      method: "https",
      siteId: managed[0]?.siteId ?? 0,
    },
  });
  const create = useTunnelMutation(
    (value: PublicationValues) => {
      const { siteId, ...body } = value;
      return api.post(
        createSite ? "/tunnels/sites" : "/tunnels/resources",
        createSite ? body : { ...body, siteId },
      );
    },
    createSite
      ? "Site e recurso criados; conector iniciado e publicação ativada com autenticação Pangolin."
      : "Recurso e alvo nginx criados. Ative o serviço quando estiver pronto.",
  );
  return (
    <form
      className="space-y-5"
      onSubmit={handleSubmit((value) => {
        if (!domainReady) return;
        setStartedAt(Date.now());
        create.mutate(value, {
          onSuccess: () => {
            reset();
            onSuccess?.();
          },
        });
      })}
    >
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="space-y-2">
          <Label
            htmlFor={
              createSite ? "publication-site-name" : "publication-resource-name"
            }
          >
            {createSite ? "Nome do site" : "Nome do serviço"}
          </Label>
          <Input
            id={
              createSite ? "publication-site-name" : "publication-resource-name"
            }
            placeholder="Minha aplicação"
            {...register("name")}
          />
        </div>
        {!createSite && (
          <div className="space-y-2">
            <Label htmlFor="publication-connector">Site / conector</Label>
            <SelectNative id="publication-connector" {...register("siteId")}>
              <option value="0">Selecione um site</option>
              {managed.map((value) => (
                <option key={value.siteId} value={value.siteId}>
                  {value.name}
                  {value.online ? " · online" : " · offline"}
                </option>
              ))}
            </SelectNative>
          </div>
        )}
      </div>
      <PublicDomainPicker
        domains={domains}
        domainId={watch("domainId")}
        subdomain={watch("subdomain")}
        onDomainChange={(value) => {
          setValue("domainId", value);
          clearErrors("domainId");
        }}
        onSubdomainChange={(value) => setValue("subdomain", value)}
        onReadyChange={setDomainReady}
      />
      <LocalDomainFields
        localDomain={watch("localDomain")}
        method={watch("method")}
        onDomainChange={(value) => setValue("localDomain", value)}
        onMethodChange={(value) => setValue("method", value)}
      />
      <div className="rounded-lg border bg-muted/30 p-3">
        <p className="break-all text-sm font-medium">
          Pangolin → Bindnet / Newt → nginx (
          {watch("localDomain") || "domínio local"}) → aplicação
        </p>
        <p className="mt-1 text-xs text-muted-foreground">
          O Bindnet configura o alvo em 127.0.0.1:
          {watch("method") === "https" ? 443 : 80}, Host local e SNI HTTPS. Não
          precisa informar o IP nem a porta da aplicação.
        </p>
      </div>
      <p className="text-xs text-muted-foreground">
        TLS público e autenticação Pangolin permanecem ativos. A aplicação deve
        aceitar o endereço público nas suas URLs e redirecionamentos.
        {createSite
          ? " Ao criar, o serviço será publicado; o primeiro arranque pode demorar enquanto baixa a imagem do conector."
          : " O recurso permanece desativado até ser ativado na lista."}
      </p>
      {Object.values(errors).map((error, index) => (
        <p key={index} role="alert" className="text-sm text-destructive">
          {error.message}
        </p>
      ))}
      {createSite && (
        <PublicationProgress active={create.isPending} startedAt={startedAt} />
      )}
      {create.isError && (
        <p
          role="alert"
          className="rounded border border-destructive/30 p-3 text-sm text-destructive"
        >
          {create.error.message}
        </p>
      )}
      <Button
        disabled={
          create.isPending || !domainReady || (!createSite && !watch("siteId"))
        }
      >
        <Plus className="mr-2 h-4 w-4" />
        {create.isPending
          ? "Preparando publicação…"
          : createSite
            ? "Criar e publicar site"
            : "Criar recurso e alvo local"}
      </Button>
    </form>
  );
}
