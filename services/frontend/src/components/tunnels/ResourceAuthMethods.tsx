import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { SelectNative } from "@/components/ui/select-native";
import { useTunnelMutation } from "./useTunnels";
import { useTunnelApi } from "./useTunnelAccount";

const schema = z.object({ value: z.string(), user: z.string() });
export function ResourceAuthMethods({ resourceId }: { resourceId: number }) {
  const api = useTunnelApi();
  const [method, setMethod] = useState("pincode");
  const { register, handleSubmit, reset } = useForm<z.infer<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: { value: "", user: "" },
  });
  const save = useTunnelMutation(
    (body: unknown) => api.post(`/tunnels/resources/${resourceId}/auth-methods/${method}`, body),
    "Método de autenticação atualizado.",
  );
  function body(value: string | null, user: string | null = null) {
    return method === "whitelist"
      ? {
          emails:
            value
              ?.split(/[\n,]/)
              .map((item) => item.trim())
              .filter(Boolean) ?? [],
        }
      : method === "header-auth"
        ? { user, password: value, extendedCompatibility: false }
        : { [method]: value };
  }
  return (
    <section className="space-y-4 rounded-lg border p-4">
      <h3 className="font-medium">Outros métodos de autenticação</h3>
      <p className="text-xs text-muted-foreground">
        PINs e senhas nunca são devolvidos pelo Bindnet. Preencha somente quando quiser substituir o valor.
      </p>
      <form
        className="space-y-4"
        onSubmit={handleSubmit((values) =>
          save.mutate(body(values.value, values.user), { onSuccess: () => reset() }),
        )}
      >
        <div className="space-y-2">
          <Label htmlFor="auth-method">Método</Label>
          <SelectNative
            id="auth-method"
            value={method}
            onChange={(event) => {
              setMethod(event.target.value);
              reset();
            }}
          >
            <option value="pincode">Código PIN (seis dígitos)</option>
            <option value="password">Código de acesso / senha</option>
            <option value="whitelist">Lista de e-mails permitidos</option>
            <option value="header-auth">Autenticação básica por cabeçalho</option>
          </SelectNative>
        </div>
        {method === "header-auth" && (
          <div className="space-y-2">
            <Label htmlFor="auth-method-user">Utilizador</Label>
            <Input id="auth-method-user" autoComplete="off" {...register("user")} />
          </div>
        )}
        <div className="space-y-2">
          <Label htmlFor="auth-method-value">
            {method === "whitelist" ? "E-mails (um por linha)" : "Novo valor"}
          </Label>
          {method === "whitelist" ? (
            <Textarea id="auth-method-value" rows={4} {...register("value")} />
          ) : (
            <Input
              id="auth-method-value"
              type="password"
              autoComplete="new-password"
              {...register("value")}
            />
          )}
        </div>
        <div className="flex flex-col gap-2 sm:flex-row">
          <Button disabled={save.isPending}>Salvar método</Button>
          <Button
            type="button"
            variant="outline"
            disabled={save.isPending}
            onClick={() => save.mutate(body(null), { onSuccess: () => reset() })}
          >
            Remover configuração do método
          </Button>
        </div>
      </form>
    </section>
  );
}
