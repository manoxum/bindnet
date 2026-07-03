import { useState } from "react";
import { NavLink } from "react-router-dom";
import { ChevronLeft, ChevronRight, Network } from "lucide-react";
import { cn } from "@/lib/utils";
import { navItems } from "@/components/layout/nav-items";

const STORAGE_KEY = "bindnet-sidebar-collapsed";

function getInitialCollapsed() {
  return localStorage.getItem(STORAGE_KEY) === "1";
}

// Navegação fixa em telas >= sm; em telas menores o acesso é só pelo MobileNav (menu hambúrguer).
export function Sidebar() {
  const [collapsed, setCollapsed] = useState(getInitialCollapsed);

  function toggleCollapsed() {
    setCollapsed((current) => {
      const next = !current;
      localStorage.setItem(STORAGE_KEY, next ? "1" : "0");
      return next;
    });
  }

  return (
    <aside
      className={cn(
        "hidden flex-col border-r bg-background p-3 transition-[width] duration-200 sm:flex",
        collapsed ? "w-[4.5rem]" : "w-60",
      )}
    >
      <div className={cn("mb-6 flex items-center gap-2 px-1", collapsed && "justify-center")}>
        <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-primary text-primary-foreground">
          <Network className="h-4 w-4" />
        </div>
        {!collapsed && <span className="truncate text-lg font-semibold tracking-tight">bindnet</span>}
      </div>

      <nav className="flex flex-1 flex-col gap-1.5">
        {navItems.map(({ to, label, icon: Icon, end }) => (
          <NavLink
            key={to}
            to={to}
            end={end}
            title={collapsed ? label : undefined}
            className={({ isActive }) =>
              cn(
                "flex h-11 items-center gap-3 rounded-md border-l-2 border-transparent px-2.5 text-sm font-medium text-muted-foreground transition-all hover:translate-x-0.5 hover:bg-accent hover:text-accent-foreground",
                collapsed && "justify-center px-0 hover:translate-x-0",
                isActive && "border-primary bg-accent font-semibold text-accent-foreground",
              )
            }
          >
            <Icon className="h-5 w-5 shrink-0" />
            {!collapsed && <span className="truncate">{label}</span>}
          </NavLink>
        ))}
      </nav>

      <button
        type="button"
        onClick={toggleCollapsed}
        className="mt-2 flex h-9 items-center justify-center gap-2 rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
        aria-label={collapsed ? "Expandir menu" : "Encolher menu"}
      >
        {collapsed ? <ChevronRight className="h-4 w-4" /> : <ChevronLeft className="h-4 w-4" />}
        {!collapsed && <span className="text-xs">Encolher</span>}
      </button>
    </aside>
  );
}
