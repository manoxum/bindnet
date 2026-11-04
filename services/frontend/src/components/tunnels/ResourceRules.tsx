import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { SelectNative } from "@/components/ui/select-native";
import { ResourceSettingsForm } from "./ResourceSettingsForm";
import { ConfirmTunnelAction, type TunnelConfirmation } from "./ConfirmTunnelAction";
import { useTunnelMutation } from "./useTunnels";
import { useTunnelAccount, useTunnelApi } from "./useTunnelAccount";
import type { ResourceSettings, SettingField } from "./resource-settings-types";
const fields: SettingField[] = [{ key: "applyRules", label: "Ativar regras de acesso", type: "toggle" }];
const schema = z.object({
  action: z.enum(["ACCEPT", "DROP", "PASS"]),
  match: z.string().min(1),
  value: z.string().min(1),
  priority: z.coerce.number().int().nonnegative(),
  enabled: z.boolean(),
});
type Rule = z.infer<typeof schema> & { ruleId?: number };
export function ResourceRules({ resourceId, settings }: { resourceId: number; settings: ResourceSettings }) {
  const api = useTunnelApi();
  const account = useTunnelAccount();
  const [editing, setEditing] = useState<number | null>(null);
  const [confirmation, setConfirmation] = useState<TunnelConfirmation | null>(null);
  const { register, handleSubmit, reset } = useForm<Rule>({
    resolver: zodResolver(schema),
    defaultValues: { action: "PASS", match: "PATH", value: "", priority: 1, enabled: true },
  });
  const rules = useQuery({
    queryKey: ["tunnels", account, "rules", resourceId],
    queryFn: () => api.get<Rule[]>(`/tunnels/resources/${resourceId}/rules`),
    retry: false,
  });
  const save = useTunnelMutation(
    (body: Rule) =>
      editing
        ? api.patch(`/tunnels/resources/${resourceId}/rules/${editing}`, body)
        : api.post(`/tunnels/resources/${resourceId}/rules`, body),
    "Regra salva.",
  );
  const remove = useTunnelMutation(
    (id: number) => api.del(`/tunnels/resources/${resourceId}/rules/${id}`),
    "Regra removida.",
  );
  return (
    <div className="space-y-5">
      <ResourceSettingsForm resourceId={resourceId} settings={settings} fields={fields} />
      <p className="text-xs text-muted-foreground">
        Aceitar permite acesso sem autenticação. Bloquear impede o acesso. Autenticar mantém os métodos
        configurados. As regras são avaliadas pelo Pangolin na ordem de prioridade.
      </p>
      {rules.isError && (
        <p role="alert" className="text-sm text-destructive">
          {rules.error.message}
        </p>
      )}
      <div className="space-y-2">
        {rules.data?.map((rule) => (
          <div
            key={rule.ruleId}
            className="flex flex-col gap-2 rounded border p-3 sm:flex-row sm:items-center sm:justify-between"
          >
            <p className="break-all text-sm">
              {rule.priority} · {rule.action} · {rule.match}: {rule.value} ·{" "}
              {rule.enabled ? "Ativa" : "Desativada"}
            </p>
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => {
                  setEditing(rule.ruleId ?? null);
                  const { ruleId: _, ...body } = rule;
                  reset(body);
                }}
              >
                Editar
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={remove.isPending}
                onClick={() =>
                  setConfirmation({
                    title: "Remover regra?",
                    description: "A política de acesso será alterada imediatamente.",
                    action: () => remove.mutate(rule.ruleId!),
                  })
                }
              >
                Remover
              </Button>
            </div>
          </div>
        ))}
      </div>
      <form
        className="space-y-4 rounded-lg border p-4"
        onSubmit={handleSubmit((body) =>
          save.mutate(body, {
            onSuccess: () => {
              setEditing(null);
              reset();
            },
          }),
        )}
      >
        <h3 className="font-medium">{editing ? "Editar regra" : "Adicionar regra"}</h3>
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor="rule-action">Ação</Label>
            <SelectNative id="rule-action" {...register("action")}>
              <option value="PASS">Exigir autenticação</option>
              <option value="ACCEPT">Aceitar sem autenticação</option>
              <option value="DROP">Bloquear</option>
            </SelectNative>
          </div>
          <div className="space-y-2">
            <Label htmlFor="rule-match">Correspondência</Label>
            <SelectNative id="rule-match" {...register("match")}>
              {["PATH", "IP", "CIDR", "COUNTRY", "COUNTRY_IS_NOT", "ASN", "REGION", "METHOD"].map((value) => (
                <option key={value} value={value}>
                  {value}
                </option>
              ))}
            </SelectNative>
          </div>
          <div className="space-y-2">
            <Label htmlFor="rule-value">Valor</Label>
            <Input id="rule-value" {...register("value")} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="rule-priority">Prioridade</Label>
            <Input id="rule-priority" type="number" min={0} {...register("priority")} />
          </div>
        </div>
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" {...register("enabled")} />
          Regra ativa
        </label>
        <Button disabled={save.isPending}>Salvar regra</Button>
      </form>
      <ConfirmTunnelAction value={confirmation} close={() => setConfirmation(null)} />
    </div>
  );
}
