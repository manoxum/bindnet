import { useForm } from "react-hook-form";
import { useEffect } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { SelectNative } from "@/components/ui/select-native";
import { useTunnelMutation } from "./useTunnels";
import { useTunnelApi } from "./useTunnelAccount";
import type { HeaderValue, ResourceSettings, SettingField } from "./resource-settings-types";

function defaults(settings: ResourceSettings, fields: SettingField[]) {
  const result: Record<string, string | boolean> = {};
  for (const field of fields) {
    const value = settings[field.key];
    result[field.key] =
      field.type === "headers"
        ? Array.isArray(value)
          ? value.map((header) => `${header.name}: ${header.value}`).join("\n")
          : ""
        : field.type === "toggle"
          ? value === true
          : String(value ?? (field.type === "select" ? (field.options?.[0]?.value ?? "") : ""));
  }
  return result;
}
export function ResourceSettingsForm({
  resourceId,
  settings,
  fields,
}: {
  resourceId: number;
  settings: ResourceSettings;
  fields: SettingField[];
}) {
  const api = useTunnelApi();
  const {
    register,
    handleSubmit,
    reset,
    formState: { dirtyFields },
  } = useForm<Record<string, string | boolean>>({ defaultValues: defaults(settings, fields) });
  useEffect(() => reset(defaults(settings, fields)), [settings, fields, reset]);
  const save = useTunnelMutation(
    (value: Record<string, unknown>) => api.patch(`/tunnels/resources/${resourceId}/settings`, value),
    "Configurações do recurso salvas no Pangolin.",
  );
  function submit(values: Record<string, string | boolean>) {
    const body: Record<string, unknown> = {};
    for (const field of fields) {
      if (!dirtyFields[field.key]) continue;
      const value = values[field.key];
      if (field.type === "headers") {
        const headers: HeaderValue[] = [];
        for (const line of String(value)
          .split("\n")
          .filter((line) => line.trim())) {
          const colon = line.indexOf(":");
          if (colon <= 0) {
            toast.error("Use Nome: valor em cada cabeçalho.");
            return;
          }
          headers.push({ name: line.slice(0, colon).trim(), value: line.slice(colon + 1).trim() });
        }
        body[field.key] = headers;
      } else if (field.type === "number") {
        body[field.key] = value === "" ? null : Number(value);
      } else body[field.key] = value;
    }
    if (Object.keys(body).length === 0) {
      toast.info("Nenhuma configuração alterada.");
      return;
    }
    save.mutate(body);
  }
  return (
    <form className="space-y-4" onSubmit={handleSubmit(submit)}>
      <div className="grid gap-4 sm:grid-cols-2">
        {fields.map((field) => (
          <div
            key={field.key}
            className={`space-y-2 ${field.type === "headers" || field.type === "textarea" ? "sm:col-span-2" : ""}`}
          >
            {field.type === "toggle" ? (
              <label className="flex items-center gap-3 rounded-lg border p-3">
                <input type="checkbox" className="h-4 w-4 accent-current" {...register(field.key)} />
                <span className="text-sm font-medium">{field.label}</span>
              </label>
            ) : (
              <>
                <Label htmlFor={`setting-${field.key}`}>{field.label}</Label>
                {field.type === "select" ? (
                  <SelectNative id={`setting-${field.key}`} {...register(field.key)}>
                    {field.options?.map((option) => (
                      <option key={option.value} value={option.value}>
                        {option.label}
                      </option>
                    ))}
                  </SelectNative>
                ) : field.type === "textarea" || field.type === "headers" ? (
                  <Textarea id={`setting-${field.key}`} rows={3} {...register(field.key)} />
                ) : (
                  <Input
                    id={`setting-${field.key}`}
                    type={field.type === "number" ? "number" : "text"}
                    {...register(field.key)}
                  />
                )}
              </>
            )}
            {field.description && <p className="text-xs text-muted-foreground">{field.description}</p>}
          </div>
        ))}
      </div>
      <Button disabled={save.isPending}>{save.isPending ? "Salvando…" : "Salvar configurações"}</Button>
    </form>
  );
}
