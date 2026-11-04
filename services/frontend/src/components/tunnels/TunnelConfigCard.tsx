import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useEffect } from "react";
import { KeyRound, PlugZap } from "lucide-react";
import { api } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { useTunnelMutation } from "./useTunnels";
import type { TunnelConfig } from "./types";

const schema = z.object({
  name: z.string().trim().min(1, "Informe o nome da conta").max(100),
  apiUrl: z.string().url("URL da API inválida"),
  endpoint: z.string().url("Endpoint inválido"),
  orgId: z.string().trim().min(1, "Informe a organização"),
  apiKey: z.string(),
});
type Values = z.infer<typeof schema>;

export function TunnelConfigCard({
  config,
  onCreated,
}: {
  config?: TunnelConfig;
  onCreated?: (account: TunnelConfig) => void;
}) {
  const {
    register,
    handleSubmit,
    reset,
    setValue,
    formState: { errors },
  } = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: "",
      apiUrl: "https://api.pangolin.net",
      endpoint: "https://app.pangolin.net",
      orgId: "",
      apiKey: "",
    },
  });
  useEffect(() => {
    if (config) reset({ ...config, apiKey: "" });
  }, [config, reset]);
  const save = useTunnelMutation(async (value: Values) => {
    const body = { ...value, ...(!value.apiKey ? { apiKey: undefined } : {}) };
    if (config) return api.put(`/tunnels/accounts/${config.id}/config`, body);
    const created = await api.post<TunnelConfig>("/tunnels/accounts", body);
    onCreated?.(created);
    return created;
  }, "Conta Pangolin salva.");
  const test = useTunnelMutation(
    () => api.post(`/tunnels/accounts/${config?.id}/test`),
    "Autenticação e listagens confirmadas. Domínios gratuitos dependem das rotas disponíveis no Cloud.",
  );

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <KeyRound className="h-5 w-5 text-primary" />{" "}
          {config ? "Configuração da conta" : "Adicionar conta Pangolin"}
        </CardTitle>
        <CardDescription>
          Conecte um servidor existente ou o Pangolin Cloud usando uma chave da Integration API.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <form
          className="space-y-4"
          onSubmit={handleSubmit((value) => save.mutate(value, { onSuccess: () => setValue("apiKey", "") }))}
        >
          <div className="space-y-2">
            <Label htmlFor="tunnel-account-name">Nome da conta no Bindnet</Label>
            <Input id="tunnel-account-name" placeholder="Conta principal / projeto" {...register("name")} />
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="tunnel-api">Origem da API</Label>
              <Input id="tunnel-api" placeholder="https://api.exemplo.com" {...register("apiUrl")} />
              <p className="text-xs text-muted-foreground">
                Sem /v1. Pode usar host e porta próprios da API.
              </p>
            </div>
            <div className="space-y-2">
              <Label htmlFor="tunnel-endpoint">Endpoint público Pangolin</Label>
              <Input
                id="tunnel-endpoint"
                placeholder="https://pangolin.exemplo.com"
                {...register("endpoint")}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="tunnel-org">Identificador da organização</Label>
              <Input id="tunnel-org" autoComplete="off" {...register("orgId")} />
            </div>
            <div className="space-y-2">
              <Label htmlFor="tunnel-key">Chave da Integration API</Label>
              <Input
                id="tunnel-key"
                type="password"
                autoComplete="new-password"
                placeholder={
                  config?.configured ? "Salva — preencha para substituir" : "Cole a chave da organização"
                }
                {...register("apiKey")}
              />
            </div>
          </div>
          {Object.values(errors).map((error, index) => (
            <p key={index} role="alert" className="text-sm text-destructive">
              {error.message}
            </p>
          ))}
          <div className="flex flex-col gap-2 sm:flex-row">
            <Button disabled={save.isPending}>{save.isPending ? "Salvando…" : "Salvar conta"}</Button>
            <Button
              type="button"
              variant="outline"
              disabled={!config?.configured || test.isPending || save.isPending}
              onClick={() => test.mutate()}
            >
              <PlugZap className="mr-2 h-4 w-4" />
              {test.isPending ? "Testando…" : "Testar configuração salva"}
            </Button>
          </div>
        </form>
        <p className="text-xs leading-relaxed text-muted-foreground">
          No Pangolin Cloud, crie uma chave de organização em API Keys. A chave precisa de permissões de
          sites, recursos, destinos e listagem de domínios. Credenciais ficam no PostgreSQL.{" "}
          <a
            href="https://docs.pangolin.net/manage/integration-api"
            target="_blank"
            rel="noreferrer"
            className="underline underline-offset-4"
          >
            Ver documentação oficial
          </a>
          .
        </p>
      </CardContent>
    </Card>
  );
}
