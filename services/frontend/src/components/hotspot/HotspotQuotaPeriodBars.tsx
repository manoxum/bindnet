import { useState } from "react";
import { Progress } from "@/components/ui/progress";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { formatAutoScaleBytes, type ByteNature } from "@/components/hotspot/hotspot-limits-types";
import type { HotspotDeviceQuotaPeriodUsage, QuotaPeriod } from "@/components/hotspot/hotspot-limits-types";

const periodLabels: Record<QuotaPeriod, string> = {
  daily: "diária",
  weekly: "semanal",
  monthly: "mensal",
};

interface HotspotQuotaPeriodBarsProps {
  periods: HotspotDeviceQuotaPeriodUsage[];
  // onReset ausente = view de auto-serviço (portal do dispositivo, sem
  // acao de admin) - so desenha as barras, sem botao.
  onReset?: (period: QuotaPeriod) => void;
  resetPending?: QuotaPeriod | null;
}

// Uma barra de progresso + botao "Resetar" por periodo configurado
// (nao um botao unico que zera tudo) - reusada pelo detalhe de
// dispositivo (com onReset) e pelo portal de autoatendimento (sem
// onReset).
//
// Todo valor (consumido, teto, restante) escolhe a propria escala pelo
// proprio tamanho, via autoScaleBytes - antes o consumido era formatado
// na unidade do TETO, entao 1KB gastos de uma cota de 3GB apareciam
// como "0GB". A natureza (bit ou byte) e a unica escolha do operador e
// vale pro card inteiro: clicar em qualquer valor consumido alterna as
// duas, mantendo todas as barras na mesma grandeza (misturar bits numa
// linha e bytes na outra so confundiria a leitura).
export function HotspotQuotaPeriodBars({ periods, onReset, resetPending }: HotspotQuotaPeriodBarsProps) {
  const [nature, setNature] = useState<ByteNature>("byte");

  if (periods.length === 0) {
    return <p className="text-sm text-muted-foreground">Nenhuma cota configurada.</p>;
  }

  const toggleNature = () => setNature((current) => (current === "byte" ? "bit" : "byte"));

  return (
    <div className="space-y-4">
      {periods.map((period) => {
        const usedBytes = period.downloadBytes + period.uploadBytes;
        const remainingBytes = period.blocked ? 0 : Math.max(period.quotaBytes - usedBytes, 0);
        const used = formatAutoScaleBytes(usedBytes, nature);
        const quota = formatAutoScaleBytes(period.quotaBytes, nature);
        const remaining = formatAutoScaleBytes(remainingBytes, nature);
        const percent = period.quotaBytes > 0 ? (usedBytes / period.quotaBytes) * 100 : 0;
        return (
          <div key={period.periodType} className="space-y-2">
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">Cota {periodLabels[period.periodType]}</span>
              <span className={cn("font-medium", period.blocked && "text-destructive")}>
                <button
                  type="button"
                  onClick={toggleNature}
                  title="Alternar entre bits e bytes"
                  aria-label={`Consumo em ${nature === "byte" ? "bytes" : "bits"}: alternar grandeza`}
                  className="rounded underline decoration-dotted underline-offset-4 hover:text-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                >
                  {used}
                </button>{" "}
                / {quota}
              </span>
            </div>
            <Progress value={Math.min(percent, 100)} />
            <p className="text-xs text-muted-foreground">Restante: {remaining}</p>
            <div className="flex items-center justify-between gap-2">
              {period.blocked ? (
                <p className="text-xs text-destructive">Cota estourada - tráfego bloqueado.</p>
              ) : (
                <span />
              )}
              {onReset && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => onReset(period.periodType)}
                  disabled={resetPending === period.periodType}
                >
                  Resetar
                </Button>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}
