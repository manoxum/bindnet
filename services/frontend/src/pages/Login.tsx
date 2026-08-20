import { useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { Navigate } from "react-router-dom";
import { toast } from "sonner";
import {
  ArrowRight,
  Check,
  Eye,
  EyeOff,
  KeyRound,
  LockKeyhole,
  Network,
  RadioTower,
  ServerCog,
  ShieldCheck,
  UserRound,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useLogin, useSession } from "@/hooks/useAuth";
import { ApiError } from "@/lib/api";

const loginSchema = z.object({
  username: z.string().min(1, "Informe o usuário"),
  password: z.string().min(1, "Informe a senha"),
});
type LoginForm = z.infer<typeof loginSchema>;

const operations = [
  { icon: RadioTower, label: "Hotspot e dispositivos" },
  { icon: ServerCog, label: "DNS e serviços internos" },
  { icon: KeyRound, label: "Certificados locais" },
];

export function LoginPage() {
  const { data: session } = useSession();
  const login = useLogin();
  const [showPassword, setShowPassword] = useState(false);
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginForm>({ resolver: zodResolver(loginSchema) });

  if (session?.username) {
    return <Navigate to="/" replace />;
  }

  async function onSubmit(data: LoginForm) {
    try {
      await login.mutateAsync(data);
    } catch (error) {
      const message = error instanceof ApiError ? error.message : "Falha ao entrar";
      toast.error(message || "Usuário ou senha inválidos");
    }
  }

  return (
    <main className="admin-login-surface relative min-h-screen overflow-hidden bg-[#08111f] text-slate-100">
      <div aria-hidden="true" className="admin-login-grid absolute inset-0 opacity-25" />
      <div aria-hidden="true" className="absolute -left-36 bottom-[-13rem] h-[32rem] w-[32rem] rounded-full border-[5rem] border-emerald-300/[0.035]" />

      <div className="relative mx-auto grid min-h-screen w-full max-w-[1440px] lg:grid-cols-[1.05fr_0.95fr]">
        <section className="login-reveal flex flex-col justify-between px-6 py-7 sm:px-10 lg:px-16 lg:py-12 xl:px-24">
          <header className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <span className="flex h-10 w-10 items-center justify-center rounded-xl border border-emerald-200/20 bg-emerald-200/10 text-emerald-200">
                <Network className="h-5 w-5" />
              </span>
              <div>
                <p className="text-lg font-semibold tracking-[-0.03em]">bindnet</p>
                <p className="text-[9px] font-semibold uppercase tracking-[0.24em] text-slate-500">network control</p>
              </div>
            </div>
            <span className="hidden rounded-full border border-white/10 bg-white/[0.035] px-3 py-1.5 text-[10px] font-semibold uppercase tracking-[0.16em] text-slate-400 sm:block">
              acesso administrativo
            </span>
          </header>

          <div className="my-14 max-w-2xl lg:my-20">
            <div className="inline-flex items-center gap-2 rounded-full border border-emerald-200/15 bg-emerald-200/[0.06] px-3 py-1.5 text-xs font-medium text-emerald-100">
              <span className="h-1.5 w-1.5 rounded-full bg-emerald-300 shadow-[0_0_10px_#6ee7b7]" />
              Operação local protegida
            </div>
            <h1 className="admin-display mt-7 text-[clamp(2.8rem,6vw,6rem)] font-medium leading-[0.92] tracking-[-0.055em] text-[#eef5ff]">
              Sua rede,
              <br />
              <span className="text-emerald-200">sem pontos cegos.</span>
            </h1>
            <p className="mt-7 max-w-lg text-sm leading-7 text-slate-400 sm:text-base">
              Um único posto de comando para acompanhar acesso, políticas de tráfego, nomes internos e confiança digital.
            </p>

            <div className="mt-9 grid gap-3 sm:grid-cols-3 lg:max-w-2xl">
              {operations.map(({ icon: Icon, label }) => (
                <div key={label} className="rounded-2xl border border-white/[0.08] bg-white/[0.035] p-4 backdrop-blur">
                  <Icon className="h-5 w-5 text-emerald-200" />
                  <p className="mt-5 text-xs font-medium leading-5 text-slate-300">{label}</p>
                </div>
              ))}
            </div>
          </div>

          <footer className="flex items-center gap-2 text-xs text-slate-500">
            <ShieldCheck className="h-4 w-4 text-emerald-300/70" />
            Sessão protegida na infraestrutura local
          </footer>
        </section>

        <section className="login-reveal flex items-center justify-center border-t border-white/10 bg-[#edf2ec] px-4 py-10 text-[#102019] sm:px-8 lg:border-l lg:border-t-0 lg:py-16" style={{ animationDelay: "100ms" }}>
          <div className="w-full max-w-md">
            <div className="mb-9">
              <span className="inline-flex h-12 w-12 items-center justify-center rounded-2xl bg-[#102019] text-[#c5f86f] shadow-[0_12px_32px_rgba(16,32,25,0.18)]">
                <LockKeyhole className="h-5 w-5" />
              </span>
              <p className="mt-8 text-xs font-bold uppercase tracking-[0.2em] text-[#617168]">Área administrativa</p>
              <h2 className="admin-display mt-3 text-4xl font-semibold tracking-[-0.045em] text-[#102019] sm:text-5xl">Bem-vindo de volta.</h2>
              <p className="mt-3 text-sm leading-6 text-[#607067]">Entre com as credenciais definidas durante a instalação do Bindnet.</p>
            </div>

            <form className="space-y-5" onSubmit={handleSubmit(onSubmit)} noValidate>
              <div className="space-y-2">
                <Label htmlFor="username" className="text-xs font-bold uppercase tracking-[0.14em] text-[#45564c]">
                  Usuário
                </Label>
                <div className="relative">
                  <UserRound className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-[#738179]" />
                  <Input
                    id="username"
                    autoComplete="username"
                    autoFocus
                    aria-invalid={!!errors.username}
                    className="h-12 border-[#bdc8c0] bg-white/70 pl-11 text-base text-[#102019] shadow-none placeholder:text-[#8d9991] focus-visible:border-[#426e58] focus-visible:ring-[#426e58]"
                    {...register("username")}
                  />
                </div>
                {errors.username && <p className="text-sm font-medium text-red-700">{errors.username.message}</p>}
              </div>

              <div className="space-y-2">
                <Label htmlFor="password" className="text-xs font-bold uppercase tracking-[0.14em] text-[#45564c]">
                  Senha
                </Label>
                <div className="relative">
                  <KeyRound className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-[#738179]" />
                  <Input
                    id="password"
                    type={showPassword ? "text" : "password"}
                    autoComplete="current-password"
                    aria-invalid={!!errors.password}
                    className="h-12 border-[#bdc8c0] bg-white/70 pl-11 pr-12 text-base text-[#102019] shadow-none placeholder:text-[#8d9991] focus-visible:border-[#426e58] focus-visible:ring-[#426e58]"
                    {...register("password")}
                  />
                  <button
                    type="button"
                    className="absolute right-3 top-1/2 flex h-8 w-8 -translate-y-1/2 items-center justify-center rounded-lg text-[#66766d] transition-colors hover:bg-[#dce5dd] hover:text-[#102019] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#426e58]"
                    onClick={() => setShowPassword((visible) => !visible)}
                    aria-label={showPassword ? "Ocultar senha" : "Mostrar senha"}
                  >
                    {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                  </button>
                </div>
                {errors.password && <p className="text-sm font-medium text-red-700">{errors.password.message}</p>}
              </div>

              <Button
                type="submit"
                className="group h-12 w-full bg-[#102019] text-[#dcff9e] shadow-[0_12px_28px_rgba(16,32,25,0.18)] hover:bg-[#1b3527]"
                disabled={login.isPending}
              >
                {login.isPending ? "Verificando acesso…" : "Entrar no painel"}
                {!login.isPending && <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-1" />}
              </Button>
            </form>

            <div className="mt-7 flex items-start gap-3 border-t border-[#cbd4cd] pt-6 text-xs leading-5 text-[#68776e]">
              <span className="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-[#d9e8dc] text-[#315a45]">
                <Check className="h-3 w-3" />
              </span>
              Este painel está disponível exclusivamente em admin.bindnet.local.com.
            </div>
          </div>
        </section>
      </div>
    </main>
  );
}
