import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { LogsPanel } from "@/components/LogsPanel";
import { DiscoverMeshConfigCard } from "@/components/DiscoverMeshConfigCard";
import { api, ApiError } from "@/lib/api";
import { usePageHeader } from "@/hooks/usePageHeader";

interface TestResponse {
  addresses?: string[];
  error?: string;
}

export function DnsPage() {
  usePageHeader({ title: "DNS local (split-horizon)", description: "TLDs locais, malha de descoberta e testes de resolução." });

  const queryClient = useQueryClient();
  const [tlds, setTlds] = useState<string[]>([]);
  const [newTld, setNewTld] = useState("");
  const [testHostname, setTestHostname] = useState("");
  const [testResult, setTestResult] = useState<TestResponse | null>(null);

  const config = useQuery<Record<string, string>>({
    queryKey: ["dns", "config"],
    queryFn: () => api.get<Record<string, string>>("/dns/config"),
  });

  useEffect(() => {
    if (config.data?.DNS_LOCAL_TLDS) {
      setTlds(config.data.DNS_LOCAL_TLDS.split(",").map((t) => t.trim()).filter(Boolean));
    }
  }, [config.data]);

  const save = useMutation({
    mutationFn: (newTlds: string[]) => api.patch("/dns/config", { DNS_LOCAL_TLDS: newTlds.join(",") }),
    onSuccess: () => {
      toast.success("TLDs salvos. Clique em 'Aplicar' para reiniciar o DNS com os novos valores.");
      queryClient.invalidateQueries({ queryKey: ["dns", "config"] });
    },
    onError: (error) => toast.error(error instanceof ApiError ? error.message : "Falha ao salvar"),
  });

  const apply = useMutation({
    mutationFn: () => api.post("/dns/apply"),
    onSuccess: () => toast.success("DNS recriado com a configuração atual."),
    onError: (error) => toast.error(error instanceof ApiError ? error.message : "Falha ao aplicar"),
  });

  const test = useMutation({
    mutationFn: (hostname: string) => api.post<TestResponse>("/dns/test", { hostname }),
    onSuccess: (response) => setTestResult(response),
    onError: (error) => toast.error(error instanceof ApiError ? error.message : "Falha ao testar"),
  });

  function addTld() {
    const value = newTld.trim().toLowerCase();
    if (!value || tlds.includes(value)) return;
    setTlds((current) => [...current, value]);
    setNewTld("");
  }

  return (
    <div className="space-y-6">
      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>TLDs locais</CardTitle>
            <CardDescription>
              Domínios como *.local respondem com o IP do hotspot; qualquer outro domínio é encaminhado normalmente.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex flex-wrap gap-2">
              {tlds.map((tld) => (
                <Badge key={tld} variant="secondary" className="gap-1">
                  {tld}
                  <button onClick={() => setTlds((current) => current.filter((t) => t !== tld))}>
                    <X className="h-3 w-3" />
                  </button>
                </Badge>
              ))}
            </div>
            <div className="flex gap-2">
              <Input
                placeholder="ex.: local"
                value={newTld}
                onChange={(e) => setNewTld(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && (e.preventDefault(), addTld())}
              />
              <Button type="button" variant="outline" onClick={addTld}>
                Adicionar
              </Button>
            </div>
            <div className="flex gap-2">
              <Button onClick={() => save.mutate(tlds)} disabled={save.isPending || tlds.length === 0}>
                Salvar
              </Button>
              <Button variant="outline" onClick={() => apply.mutate()} disabled={apply.isPending}>
                Aplicar e reiniciar
              </Button>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Testar resolução</CardTitle>
            <CardDescription>Confirme se um hostname resolve como esperado após salvar os TLDs.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex gap-2">
              <Label htmlFor="testHostname" className="sr-only">
                Hostname
              </Label>
              <Input
                id="testHostname"
                placeholder="ex.: painel.local"
                value={testHostname}
                onChange={(e) => setTestHostname(e.target.value)}
              />
              <Button onClick={() => test.mutate(testHostname)} disabled={!testHostname || test.isPending}>
                Testar
              </Button>
            </div>
            {testResult && (
              <p className="text-sm">
                {testResult.error
                  ? `Erro: ${testResult.error}`
                  : `Resolvido para: ${testResult.addresses?.join(", ")}`}
              </p>
            )}
          </CardContent>
        </Card>
      </div>

      <DiscoverMeshConfigCard />

      <LogsPanel title="Logs do DNS" path="/dns/logs" />
    </div>
  );
}
