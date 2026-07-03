import { NavLink } from "react-router-dom";
import { Network } from "lucide-react";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { cn } from "@/lib/utils";
import { navItems } from "@/components/layout/nav-items";

interface MobileNavProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

// Drawer de navegação acionado pelo botão hambúrguer, para telas < sm (onde a Sidebar fica oculta).
export function MobileNav({ open, onOpenChange }: MobileNavProps) {
  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="left" className="flex w-64 flex-col p-4">
        <SheetHeader>
          <SheetTitle className="flex items-center gap-2">
            <div className="flex h-7 w-7 items-center justify-center rounded-md bg-primary text-primary-foreground">
              <Network className="h-4 w-4" />
            </div>
            bindnet
          </SheetTitle>
        </SheetHeader>
        <nav className="mt-4 flex flex-col gap-1.5">
          {navItems.map(({ to, label, icon: Icon, end }) => (
            <NavLink
              key={to}
              to={to}
              end={end}
              onClick={() => onOpenChange(false)}
              className={({ isActive }) =>
                cn(
                  "flex h-11 items-center gap-3 rounded-md border-l-2 border-transparent px-2.5 text-sm font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground",
                  isActive && "border-primary bg-accent font-semibold text-accent-foreground",
                )
              }
            >
              <Icon className="h-5 w-5" />
              {label}
            </NavLink>
          ))}
        </nav>
      </SheetContent>
    </Sheet>
  );
}
