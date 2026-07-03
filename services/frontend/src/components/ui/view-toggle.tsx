import { LayoutGrid, List } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

export type ViewMode = "grid" | "list";

interface ViewToggleProps {
  value: ViewMode;
  onChange: (value: ViewMode) => void;
  className?: string;
}

// Alternador genérico cards/lista, reutilizável em qualquer listagem de dados.
export function ViewToggle({ value, onChange, className }: ViewToggleProps) {
  return (
    <div className={cn("inline-flex items-center gap-0.5 rounded-md border p-0.5", className)}>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        className={cn("h-7 w-7", value === "grid" && "bg-accent text-accent-foreground")}
        aria-pressed={value === "grid"}
        aria-label="Ver em cards"
        onClick={() => onChange("grid")}
      >
        <LayoutGrid className="h-4 w-4" />
      </Button>
      <Button
        type="button"
        variant="ghost"
        size="icon"
        className={cn("h-7 w-7", value === "list" && "bg-accent text-accent-foreground")}
        aria-pressed={value === "list"}
        aria-label="Ver em lista"
        onClick={() => onChange("list")}
      >
        <List className="h-4 w-4" />
      </Button>
    </div>
  );
}
