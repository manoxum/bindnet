import type { UseFormRegister } from "react-hook-form";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { SelectNative } from "@/components/ui/select-native";
import { cn } from "@/lib/utils";
import type { TimeMode } from "@/components/hotspot/hotspot-limits-types";

// Fieldset da politica de tempo (LimitType "time"), reusado por perfil e
// (futuramente) pela aba de tempo do dispositivo. O modo e um seletor
// controlado (budget/deadline); os campos por modo usam register (mesmo
// idioma de HotspotCreditConfigFields). register fica frouxamente tipado
// (UseFormRegister<any>) de proposito - o mesmo subconjunto de campos e
// compartilhado por schemas zod diferentes.
const MODES: { value: TimeMode; label: string; hint: string }[] = [
  { value: "budget", label: "Saldo de tempo", hint: "Gasta minutos de conexão aos poucos; recarrega por período." },
  { value: "deadline", label: "Prazo (deadline)", hint: "Acesso liberado até uma data/hora; depois bloqueia." },
];

export function HotspotTimeConfigFields({
  register,
  mode,
  onModeChange,
}: {
  register: UseFormRegister<any>;
  mode: TimeMode;
  onModeChange: (mode: TimeMode) => void;
}) {
  return (
    <fieldset className="space-y-4">
      <legend className="text-sm font-medium text-muted-foreground">Limitação por tempo</legend>

      <div className="grid gap-3 sm:grid-cols-2" role="radiogroup" aria-label="Modo de tempo">
        {MODES.map((option) => {
          const active = mode === option.value;
          return (
            <button
              key={option.value}
              type="button"
              role="radio"
              aria-checked={active}
              onClick={() => onModeChange(option.value)}
              className={cn(
                "flex flex-col gap-1 rounded-lg border p-3 text-left transition-colors",
                active ? "border-primary bg-primary/5 ring-1 ring-primary" : "border-border hover:border-primary/50 hover:bg-muted/40",
              )}
            >
              <span className={cn("text-sm font-medium", active && "text-primary")}>{option.label}</span>
              <span className="text-xs text-muted-foreground">{option.hint}</span>
            </button>
          );
        })}
      </div>

      {mode === "budget" ? (
        <div className="grid gap-4 sm:grid-cols-3">
          <div className="space-y-2">
            <Label htmlFor="timeBudgetMinutes">Recarga por período (min)</Label>
            <Input id="timeBudgetMinutes" placeholder="só manual" {...register("timeBudgetMinutes")} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="timeRechargePeriod">Período</Label>
            <SelectNative id="timeRechargePeriod" {...register("timeRechargePeriod")}>
              <option value="daily">Diário</option>
              <option value="weekly">Semanal</option>
              <option value="monthly">Mensal</option>
            </SelectNative>
          </div>
          <div className="space-y-2">
            <Label htmlFor="timePlafondMinutes">Plafond - teto do saldo (min)</Label>
            <Input id="timePlafondMinutes" placeholder="sem teto" {...register("timePlafondMinutes")} />
          </div>
        </div>
      ) : (
        <div className="space-y-2">
          <Label htmlFor="timeDeadlineAt">Acesso até</Label>
          <Input id="timeDeadlineAt" type="datetime-local" {...register("timeDeadlineAt")} />
          <p className="text-xs text-muted-foreground">Depois desse instante o tráfego é bloqueado (portal cativo).</p>
        </div>
      )}
    </fieldset>
  );
}
