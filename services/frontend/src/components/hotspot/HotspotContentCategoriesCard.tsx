import { RefreshCw } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { useContentCategories, useHotspotContentMutations } from "@/components/hotspot/useHotspotContent";

// Card do catalogo de categorias: liga/desliga cada uma e dispara o sync
// manual das que tem fonte publica. As categorias sao referenciadas
// pelas regras dos planos (kind=category).
export function HotspotContentCategoriesCard() {
  const categories = useContentCategories();
  const { setCategoryEnabled, syncCategory } = useHotspotContentMutations();

  return (
    <Card>
      <CardHeader>
        <CardTitle>Categorias</CardTitle>
      </CardHeader>
      <CardContent>
        <p className="mb-4 text-sm text-muted-foreground">
          Categorias populadas de blocklists públicas (ou embutidas). Use-as nas regras de um plano em vez de cadastrar
          site por site.
        </p>
        <div className="overflow-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Categoria</TableHead>
                <TableHead className="hidden sm:table-cell">Domínios</TableHead>
                <TableHead className="hidden md:table-cell">Última sync</TableHead>
                <TableHead className="text-right">Ações</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(categories.data ?? []).map((cat) => (
                <TableRow key={cat.slug}>
                  <TableCell>
                    <div className="font-medium">{cat.name}</div>
                    <div className="text-xs text-muted-foreground">{cat.slug}</div>
                  </TableCell>
                  <TableCell className="hidden sm:table-cell">{cat.domainCount.toLocaleString()}</TableCell>
                  <TableCell className="hidden md:table-cell text-sm text-muted-foreground">
                    {cat.lastSyncedAt ? new Date(cat.lastSyncedAt).toLocaleString() : "—"}
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex justify-end gap-2">
                      <Button
                        variant={cat.enabled ? "secondary" : "outline"}
                        size="sm"
                        onClick={() => setCategoryEnabled.mutate({ slug: cat.slug, enabled: !cat.enabled })}
                      >
                        {cat.enabled ? "Ativa" : "Inativa"}
                      </Button>
                      {cat.sourceUrls.trim() !== "" && (
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => syncCategory.mutate(cat.slug)}
                          aria-label={`Sincronizar ${cat.name}`}
                        >
                          <RefreshCw className="h-4 w-4" />
                          <span className="hidden sm:inline">Sincronizar</span>
                        </Button>
                      )}
                    </div>
                  </TableCell>
                </TableRow>
              ))}
              {(categories.data?.length ?? 0) === 0 && (
                <TableRow>
                  <TableCell colSpan={4} className="text-center text-sm text-muted-foreground">
                    Nenhuma categoria.
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>
        <Badge variant="outline" className="mt-4">
          A sincronização das listas públicas roda 1x/dia automaticamente.
        </Badge>
      </CardContent>
    </Card>
  );
}
