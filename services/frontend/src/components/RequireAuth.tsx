import { Navigate } from "react-router-dom";
import { useSession } from "@/hooks/useAuth";

export function RequireAuth({ children }: { children: React.ReactNode }) {
  const { data, isLoading, isError } = useSession();

  if (isLoading) {
    return <div className="flex min-h-screen items-center justify-center text-muted-foreground">Carregando...</div>;
  }
  if (isError || !data) {
    return <Navigate to="/login" replace />;
  }
  return <>{children}</>;
}
