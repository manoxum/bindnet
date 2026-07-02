import * as React from "react";
import { cn } from "@/lib/utils";

// Select nativo (em vez de @radix-ui/react-select) - suficiente para os
// dropdowns simples deste painel e evita mais uma dependência.
export type SelectNativeProps = React.SelectHTMLAttributes<HTMLSelectElement>;

const SelectNative = React.forwardRef<HTMLSelectElement, SelectNativeProps>(
  ({ className, children, ...props }, ref) => (
    <select
      ref={ref}
      className={cn(
        "flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50",
        className,
      )}
      {...props}
    >
      {children}
    </select>
  ),
);
SelectNative.displayName = "SelectNative";

export { SelectNative };
