import { lazy, Suspense } from "react";
import { Network, ShieldAlert } from "lucide-react";
import { appSurfaceForHostname } from "@/lib/app-hostname";

// Cada domínio carrega apenas a sua superfície. Além de impedir que o
// pathname escolha entre portal e administração, os imports dinâmicos
// mantêm as telas administrativas fora do bundle inicial do portal.
const AdminApp = lazy(() => import("@/apps/AdminApp"));
const PortalApp = lazy(() => import("@/apps/PortalApp"));

function AppLoading() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-[#071613] text-[#dff8ee]">
      <div className="flex items-center gap-3 text-sm font-medium">
        <Network className="h-5 w-5 animate-pulse text-[#a8ee63]" />
        Carregando Bindnet…
      </div>
    </div>
  );
}

function UnsupportedHost() {
  return (
    <main className="flex min-h-screen items-center justify-center bg-[#0a1220] p-6 text-[#e8eef8]">
      <div className="max-w-md rounded-2xl border border-white/10 bg-white/[0.04] p-8 text-center shadow-2xl">
        <ShieldAlert className="mx-auto h-9 w-9 text-amber-300" />
        <h1 className="mt-4 text-xl font-semibold">Endereço Bindnet não reconhecido</h1>
        <p className="mt-2 text-sm leading-6 text-slate-400">
          Use bindnet.local.com para o portal cativo ou admin.bindnet.local.com para a área administrativa.
        </p>
      </div>
    </main>
  );
}

export default function App() {
  const surface = appSurfaceForHostname(window.location.hostname);

  if (surface === "unknown") return <UnsupportedHost />;

  const SurfaceApp = surface === "portal" ? PortalApp : AdminApp;
  return (
    <Suspense fallback={<AppLoading />}>
      <SurfaceApp />
    </Suspense>
  );
}
