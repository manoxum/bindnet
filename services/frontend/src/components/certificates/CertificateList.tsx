import { useState } from "react";
import { ViewToggle, type ViewMode } from "@/components/ui/view-toggle";
import { CertificateTable, type CertificateListProps as CertificateData } from "@/components/certificates/CertificateTable";
import { CertificateGrid } from "@/components/certificates/CertificateCard";

interface CertificateListProps extends CertificateData {
  isLoading: boolean;
}

// Alterna entre visão em cards e em lista para uma listagem de certificados (emitidos ou revogados).
export function CertificateList({ isLoading, ...data }: CertificateListProps) {
  const [view, setView] = useState<ViewMode>("grid");

  return (
    <div className="space-y-3">
      <div className="flex justify-end">
        <ViewToggle value={view} onChange={setView} />
      </div>
      {isLoading ? (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 3 }).map((_, index) => (
            <div key={index} className="h-32 animate-pulse rounded-lg border bg-muted/30" />
          ))}
        </div>
      ) : view === "grid" ? (
        <CertificateGrid {...data} />
      ) : (
        <CertificateTable {...data} />
      )}
    </div>
  );
}
