import { useState } from "react";
import { Link } from "react-router-dom";
import {
  ArrowRight,
  CheckCircle2,
  Gauge,
  Network,
  QrCode,
  RefreshCw,
  ShieldCheck,
  Signal,
  Ticket,
  Wifi,
  WifiOff,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { HotspotQuotaPeriodBars } from "@/components/hotspot/HotspotQuotaPeriodBars";
import { bytesToGB } from "@/components/hotspot/hotspot-limits-types";
import { PortalMessages } from "@/components/portal/PortalMessages";
import { usePortalMe } from "@/components/portal/usePortalQueries";
import { useRedeemVoucher } from "@/components/portal/usePortalMutations";
import { PortalVoucherQrScanner } from "@/components/portal/PortalVoucherQrScanner";
import { ApiError } from "@/lib/api";

// Superfície pública exclusiva de bindnet.local.com. O dispositivo é
// identificado pelo IP de origem no backend, nunca por login ou por um
// MAC fornecido pelo browser.
export function PortalPage() {
  const me = usePortalMe();
  const redeem = useRedeemVoucher();
  const [code, setCode] = useState("");
  const [scannerOpen, setScannerOpen] = useState(false);

  function onRedeem(value = code) {
    if (!value.trim()) return;
    redeem.mutate(value.trim().toUpperCase(), { onSuccess: () => setCode("") });
  }

  const deviceReady = !!me.data;
  const connectionLabel = me.data?.blockedByCredit
    ? "Aguardando recarga"
    : me.data?.blocked
      ? "Conexão restrita"
      : "Conectado à rede";

  return (
    <div className="portal-surface relative min-h-screen overflow-hidden bg-[#06120f] text-[#e9fff6]">
      <div aria-hidden="true" className="portal-grid pointer-events-none absolute inset-0 opacity-35" />
      <div aria-hidden="true" className="absolute -left-28 top-20 h-72 w-72 rounded-full bg-[#43d69b]/15 blur-3xl" />
      <div aria-hidden="true" className="absolute -right-36 bottom-0 h-96 w-96 rounded-full bg-[#b8f24a]/10 blur-3xl" />
      <div aria-hidden="true" className="portal-orbit absolute right-[8%] top-16 hidden h-56 w-56 rounded-full border border-[#9bf0cf]/15 lg:block" />

      <main className="relative mx-auto flex min-h-screen w-full max-w-6xl flex-col px-4 py-5 sm:px-6 sm:py-8 lg:px-8">
        <header className="portal-reveal flex items-center justify-between border-b border-white/10 pb-5">
          <div className="flex items-center gap-3">
            <span className="flex h-10 w-10 items-center justify-center rounded-2xl border border-[#a8ee63]/30 bg-[#a8ee63]/10 shadow-[0_0_28px_rgba(168,238,99,0.14)]">
              <Network className="h-5 w-5 text-[#b8f24a]" />
            </span>
            <div>
              <p className="portal-wordmark text-lg font-semibold tracking-[-0.03em]">bindnet</p>
              <p className="text-[10px] font-semibold uppercase tracking-[0.24em] text-[#8eaaa0]">portal da rede</p>
            </div>
          </div>
          <div className="flex items-center gap-2 rounded-full border border-[#75dfb8]/20 bg-[#75dfb8]/[0.07] px-3 py-1.5 text-xs font-medium text-[#a6e9d0]">
            <span className="h-1.5 w-1.5 rounded-full bg-[#a8ee63] shadow-[0_0_10px_#a8ee63]" />
            bindnet.local.com
          </div>
        </header>

        <div className="portal-reveal mt-5" style={{ animationDelay: "70ms" }}>
          <PortalMessages />
        </div>

        <section className="my-auto grid gap-4 py-5 sm:py-8 lg:grid-cols-[1.45fr_0.85fr] lg:gap-6">
          <div
            className="portal-reveal relative overflow-hidden rounded-[1.75rem] border border-white/10 bg-[#0b211b]/90 p-5 shadow-[0_24px_80px_rgba(0,0,0,0.32)] backdrop-blur sm:p-8"
            style={{ animationDelay: "120ms" }}
          >
            <div aria-hidden="true" className="absolute right-0 top-0 h-40 w-40 translate-x-12 -translate-y-12 rounded-full border-[26px] border-[#9cf0cf]/[0.04]" />
            <div className="relative">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <p className="flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.2em] text-[#8eb2a5]">
                  <Signal className="h-4 w-4 text-[#a8ee63]" />
                  Estado da conexão
                </p>
                {me.data && (
                  <Badge
                    className={cn(
                      "rounded-full border px-3 py-1 font-medium",
                      me.data.blocked
                        ? "border-amber-300/30 bg-amber-300/10 text-amber-200"
                        : "border-[#8beaC7]/25 bg-[#8beac7]/10 text-[#b7f6df]",
                    )}
                  >
                    {connectionLabel}
                  </Badge>
                )}
              </div>

              {me.isLoading && (
                <div className="mt-9 space-y-4" aria-label="Carregando estado da conexão">
                  <div className="h-4 w-36 animate-pulse rounded-full bg-white/10" />
                  <div className="h-14 w-64 max-w-full animate-pulse rounded-2xl bg-white/10" />
                  <div className="h-24 animate-pulse rounded-2xl bg-white/[0.06]" />
                </div>
              )}

              {me.isError && (
                <div className="mt-8 rounded-3xl border border-amber-200/20 bg-amber-200/[0.06] p-5 sm:p-6">
                  <WifiOff className="h-8 w-8 text-amber-200" />
                  <h1 className="mt-4 text-2xl font-semibold tracking-tight">Não encontramos este dispositivo</h1>
                  <p className="mt-2 max-w-lg text-sm leading-6 text-[#a9beb7]">
                    {me.error instanceof ApiError && me.error.status === 409
                      ? "Reconecte-se ao Wi-Fi Bindnet e tente novamente. A identificação é feita automaticamente e não solicita dados do aparelho."
                      : "O estado da sua conexão não pôde ser carregado agora. Tente novamente em alguns instantes."}
                  </p>
                  <Button
                    variant="outline"
                    className="mt-5 border-white/15 bg-white/[0.04] text-[#effff9] hover:bg-white/10 hover:text-white"
                    onClick={() => me.refetch()}
                  >
                    <RefreshCw className="h-4 w-4" />
                    Tentar novamente
                  </Button>
                </div>
              )}

              {me.data && (
                <>
                  <div className="mt-9">
                    <p className="text-sm text-[#91aaa1]">{me.data.alias || me.data.mac}</p>
                    <h1 className="portal-display mt-2 max-w-xl text-4xl font-semibold leading-[1.02] tracking-[-0.045em] sm:text-6xl">
                      {me.data.limitType === "credit" ? (
                        <>
                          {bytesToGB(me.data.balanceBytes).toFixed(2)}
                          <span className="ml-2 text-xl font-medium text-[#9db7ae] sm:text-2xl">GB disponíveis</span>
                        </>
                      ) : (
                        "Navegação liberada"
                      )}
                    </h1>
                  </div>

                  <div className="mt-7 grid gap-3 sm:grid-cols-2">
                    <div className="rounded-2xl border border-white/[0.08] bg-white/[0.035] p-4">
                      <div className="flex items-center gap-2 text-xs font-medium uppercase tracking-[0.14em] text-[#8fa99f]">
                        <Wifi className="h-4 w-4 text-[#a8ee63]" />
                        acesso
                      </div>
                      <p className="mt-3 text-base font-semibold text-[#effff9]">{connectionLabel}</p>
                    </div>
                    <div className="rounded-2xl border border-white/[0.08] bg-white/[0.035] p-4">
                      <div className="flex items-center gap-2 text-xs font-medium uppercase tracking-[0.14em] text-[#8fa99f]">
                        <Gauge className="h-4 w-4 text-[#a8ee63]" />
                        modalidade
                      </div>
                      <p className="mt-3 text-base font-semibold text-[#effff9]">
                        {me.data.limitType === "credit"
                          ? "Crédito pré-pago"
                          : me.data.limitType === "quota"
                            ? "Cota de dados"
                            : "Sem limite de crédito"}
                      </p>
                    </div>
                  </div>

                  {me.data.limitType === "quota" && (
                    <div className="mt-4 rounded-2xl border border-white/[0.08] bg-black/10 p-4 sm:p-5">
                      <HotspotQuotaPeriodBars periods={me.data.quotaPeriods ?? []} />
                    </div>
                  )}
                </>
              )}
            </div>
          </div>

          <aside className="grid gap-4">
            <div
              className="portal-reveal rounded-[1.75rem] border border-[#b8f24a]/25 bg-[#b8f24a] p-5 text-[#102017] shadow-[0_20px_60px_rgba(111,164,42,0.18)] sm:p-7"
              style={{ animationDelay: "190ms" }}
            >
              <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-[#102017] text-[#c9ff78]">
                <Ticket className="h-5 w-5" />
              </div>
              <h2 className="portal-display mt-6 text-3xl font-semibold leading-none tracking-[-0.04em]">Adicionar crédito</h2>
              <p className="mt-3 text-sm leading-6 text-[#38503f]">Digite o código do cartão ou use a câmera para ler o QR code.</p>

              <div className="mt-6 space-y-2">
                <Label htmlFor="voucherCode" className="text-xs font-bold uppercase tracking-[0.16em] text-[#304638]">
                  Código do cartão
                </Label>
                <div className="flex gap-2">
                  <Input
                    id="voucherCode"
                    placeholder="XXXX-XXXX-XXXX"
                    value={code}
                    disabled={!deviceReady}
                    onChange={(event) => setCode(event.target.value)}
                    className="h-12 border-[#1c3526]/20 bg-[#f4ffd9]/75 px-4 font-mono text-base font-semibold tracking-wider text-[#102017] placeholder:text-[#607363] focus-visible:ring-[#102017]"
                  />
                  <Button
                    variant="outline"
                    size="icon"
                    className="h-12 w-12 shrink-0 border-[#102017]/20 bg-transparent text-[#102017] hover:bg-[#102017] hover:text-[#c9ff78]"
                    onClick={() => setScannerOpen(true)}
                    disabled={!deviceReady}
                    aria-label="Ler QR code"
                  >
                    <QrCode className="h-5 w-5" />
                  </Button>
                </div>
              </div>
              <Button
                onClick={() => onRedeem()}
                disabled={redeem.isPending || !code.trim() || !deviceReady}
                className="mt-3 h-12 w-full bg-[#102017] text-[#d8ff9d] shadow-none hover:bg-[#1a3323]"
              >
                {redeem.isPending ? "Validando código…" : "Resgatar crédito"}
                {!redeem.isPending && <ArrowRight className="h-4 w-4" />}
              </Button>
              {!deviceReady && (
                <p className="mt-3 text-xs leading-5 text-[#4e6554]">O resgate fica disponível assim que este dispositivo for identificado.</p>
              )}
            </div>

            <Link
              to="/ca"
              className="portal-reveal group flex items-center gap-4 rounded-[1.4rem] border border-white/10 bg-white/[0.045] p-4 text-[#ecfff7] transition-colors hover:border-[#8beac7]/30 hover:bg-white/[0.075] sm:p-5"
              style={{ animationDelay: "250ms" }}
            >
              <span className="flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl border border-[#8beac7]/20 bg-[#8beac7]/10 text-[#9cf0cf]">
                <ShieldCheck className="h-5 w-5" />
              </span>
              <span className="min-w-0 flex-1">
                <span className="block text-sm font-semibold">Certificado da rede</span>
                <span className="mt-0.5 block text-xs leading-5 text-[#91aaa1]">Evite avisos de segurança nos endereços internos.</span>
              </span>
              <ArrowRight className="h-4 w-4 text-[#718b82] transition-transform group-hover:translate-x-1 group-hover:text-[#b8f24a]" />
            </Link>
          </aside>
        </section>

        <footer className="portal-reveal flex flex-col gap-3 border-t border-white/10 pt-5 text-xs text-[#728d83] sm:flex-row sm:items-center sm:justify-between" style={{ animationDelay: "300ms" }}>
          <span className="flex items-center gap-2">
            <CheckCircle2 className="h-3.5 w-3.5 text-[#78d9b5]" />
            Identificação automática e segura pela rede local
          </span>
          <span>Bindnet · conectividade sob seu controle</span>
        </footer>
      </main>

      <PortalVoucherQrScanner
        open={scannerOpen}
        onOpenChange={setScannerOpen}
        onScan={(scanned) => {
          setCode(scanned);
          onRedeem(scanned);
        }}
      />
    </div>
  );
}
