export function EmptyState({ label }: { label: string }) {
  return (
    <div className="rounded-md border border-dashed bg-muted/20 px-4 py-8 text-center text-sm text-muted-foreground">
      {label}
    </div>
  );
}
