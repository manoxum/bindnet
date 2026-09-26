import { cn } from "@/lib/utils";

interface BrandMarkProps {
  className?: string;
}

export function BrandMark({ className }: BrandMarkProps) {
  return (
    <img
      src="/09-nucleo.svg"
      alt=""
      aria-hidden="true"
      draggable={false}
      className={cn("shrink-0 select-none rounded-xl", className)}
    />
  );
}
