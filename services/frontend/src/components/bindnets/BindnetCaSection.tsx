import { Download } from "lucide-react";
import { buttonVariants } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { InstallCaCard } from "@/components/InstallCaCard";
import { RemoteCaCard } from "@/components/bindnets/RemoteCaCard";
import type { BindnetNode } from "@/lib/mesh";

export function BindnetCaSection({ node }: { node: BindnetNode }) {
  if (node.kind !== "local") {
    return <RemoteCaCard node={node} />;
  }

  return (
    <div className="grid gap-4 lg:grid-cols-2">
      <Card>
        <CardHeader>
          <CardTitle>Autoridade certificadora</CardTitle>
          <CardDescription>Certificado raiz usado para assinar os certificados deste servidor.</CardDescription>
        </CardHeader>
        <CardContent className="flex items-center justify-between">
          <span className="text-sm text-muted-foreground">
            Importe este certificado nos dispositivos que devem confiar nos certificados emitidos por este servidor.
          </span>
          <a href="/api/certificates/ca" download className={buttonVariants({ variant: "outline" })}>
            <Download className="mr-2 h-4 w-4" />
            Baixar CA
          </a>
        </CardContent>
      </Card>

      <InstallCaCard />
    </div>
  );
}
