import type { ApexOptions } from "apexcharts";
import type { ApexChartColors } from "@/components/hotspot/apex-chart-theme";
import { formatScaledValue } from "@/components/hotspot/hotspot-limits-types";

type GridPadding = NonNullable<ApexOptions["grid"]>["padding"];

// speedAreaChartOptions monta as opcoes do grafico de area de
// velocidade no tempo (download/upload) - a mesma configuracao pro
// grafico de um dispositivo (device-detail/DeviceSpeedChart.tsx) e pro
// painel geral do hotspot (HotspotGlobalSpeedPanel.tsx), que antes
// duplicavam esse bloco inteiro e so diferem no padding do grid.
//
// A serie ja chega dividida pelo divisor de pickByteScale, entao
// "label" e a unidade do eixo (KB, Mb, ...) e todo ponto/tick esta
// nessa mesma escala. Os rotulos do eixo Y saem por formatScaledValue
// (casas decimais conforme a grandeza do tick, ver
// hotspot-limits-types.ts) e a escala usa forceNiceScale a partir do
// zero, pra os numeros da vertical baterem com a curva desenhada em vez
// de virarem varios "0.0" seguidos.
export function speedAreaChartOptions(
  colors: ApexChartColors,
  label: string,
  gridPadding: GridPadding,
): ApexOptions {
  return {
    chart: {
      type: "area",
      toolbar: { show: false },
      zoom: { enabled: false },
      animations: { enabled: false },
      fontFamily: "inherit",
      foreColor: colors.mutedForeground,
    },
    colors: [colors.primary, colors.secondary],
    stroke: { curve: "smooth", width: [2.5, 1.75] },
    fill: {
      type: "gradient",
      gradient: { shadeIntensity: 1, opacityFrom: 0.35, opacityTo: 0, stops: [0, 90, 100] },
    },
    dataLabels: { enabled: false },
    grid: { borderColor: colors.border, strokeDashArray: 3, padding: gridPadding },
    xaxis: {
      type: "datetime",
      labels: { datetimeUTC: false, style: { colors: colors.mutedForeground, fontSize: "10px" } },
      axisBorder: { show: false },
      axisTicks: { show: false },
    },
    yaxis: {
      min: 0,
      forceNiceScale: true,
      tickAmount: 4,
      labels: {
        formatter: (value) => formatScaledValue(value, label),
        style: { colors: colors.mutedForeground, fontSize: "10px" },
      },
    },
    tooltip: {
      theme: "dark",
      x: { format: "HH:mm:ss" },
      y: { formatter: (value) => `${formatScaledValue(value, label)}/s` },
    },
    legend: { show: false },
  };
}
