import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { SelectNative } from "@/components/ui/select-native";
import { useTunnelApi, useTunnelAccount } from "./useTunnelAccount";
import { useTunnelMutation } from "./useTunnels";
interface Subject {
  id?: string;
  userId?: string;
  roleId?: number;
  name?: string;
  email?: string;
  isAdmin?: boolean;
}
export function ResourceMembership({ resourceId, kind }: { resourceId: number; kind: "roles" | "users" }) {
  const api = useTunnelApi();
  const account = useTunnelAccount();
  const [selected, setSelected] = useState<string[]>([]);
  const options = useQuery({
    queryKey: ["tunnels", account, "access-options", kind],
    queryFn: () => api.get<Subject[]>(`/tunnels/access-options/${kind}`),
    retry: false,
  });
  const current = useQuery({
    queryKey: ["tunnels", account, "access", resourceId, kind],
    queryFn: () => api.get<Record<string, Subject[]>>(`/tunnels/resources/${resourceId}/access/${kind}`),
    retry: false,
  });
  const key = (value: Subject) => String(kind === "roles" ? value.roleId : (value.userId ?? value.id ?? ""));
  const eligible = (value: Subject) =>
    !value.isAdmin && !(kind === "roles" && value.name?.toLowerCase() === "admin");
  useEffect(() => {
    if (current.data) setSelected((current.data[kind] ?? []).filter(eligible).map(key));
  }, [current.data, kind]);
  const save = useTunnelMutation(
    () =>
      api.post(
        `/tunnels/resources/${resourceId}/auth-methods/${kind}`,
        kind === "roles" ? { roleIds: selected.map(Number) } : { userIds: selected },
      ),
    "Permissões de acesso atualizadas.",
  );
  const label = kind === "roles" ? "Funções permitidas" : "Utilizadores permitidos";
  return (
    <section className="space-y-3 rounded-lg border p-4">
      <Label htmlFor={`resource-${kind}`}>{label}</Label>
      {options.isError || current.isError ? (
        <p role="alert" className="text-sm text-destructive">
          {options.error?.message ?? current.error?.message}
        </p>
      ) : (
        <>
          <SelectNative
            id={`resource-${kind}`}
            multiple
            className="h-32"
            value={selected}
            onChange={(event) =>
              setSelected(Array.from(event.target.selectedOptions, (option) => option.value))
            }
          >
            {options.data?.filter(eligible).map((value) => (
              <option key={key(value)} value={key(value)}>
                {value.name ?? value.email ?? key(value)}
              </option>
            ))}
          </SelectNative>
          <p className="text-xs text-muted-foreground">
            A função Admin mantém acesso automático. A seleção substitui as permissões adicionais deste
            recurso.
          </p>
          <Button
            variant="outline"
            disabled={save.isPending || options.isLoading || current.isLoading}
            onClick={() => save.mutate()}
          >
            Salvar permissões
          </Button>
        </>
      )}
    </section>
  );
}
