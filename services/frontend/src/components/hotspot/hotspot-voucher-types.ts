import type { RateUnit } from "@/components/hotspot/hotspot-limits-types";

// Voucher (cartao de recarga) - emitido pelo admin com valor fixo em
// bytes, resgatado uma unica vez pelo proprio dispositivo no portal
// (ver services/backend/hotspot_vouchers.go). Status segue o mesmo
// ciclo de vida definido la: "active" (ainda nao usado), "redeemed"
// (ja resgatado) ou "revoked" (anulado pelo admin antes do uso).
export type HotspotVoucherStatus = "active" | "redeemed" | "revoked";

export interface HotspotVoucher {
  code: string;
  batchId?: string;
  amountBytes: number;
  status: HotspotVoucherStatus;
  note?: string;
  redeemedByMac?: string;
  redeemedAt?: string;
  createdAt: string;
}

// Lote de vouchers - agrupa todos os vouchers emitidos numa mesma
// chamada (ver services/backend/hotspot_voucher_batches.go) para
// listagem/impressao em conjunto.
export interface HotspotVoucherBatch {
  id: string;
  amountBytes: number;
  amountUnit: RateUnit;
  quantity: number;
  note?: string;
  createdAt: string;
  activeCount: number;
  redeemedCount: number;
  revokedCount: number;
}

export interface HotspotVoucherIssueRequest {
  amountBytes: number;
  amountUnit: RateUnit;
  quantity: number;
  note?: string;
}

export interface HotspotVoucherIssueResponse {
  batch: HotspotVoucherBatch;
  vouchers: HotspotVoucher[];
}
