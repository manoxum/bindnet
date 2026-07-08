import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type {
  HotspotVoucher,
  HotspotVoucherBatch,
  HotspotVoucherStatus,
} from "@/components/hotspot/hotspot-voucher-types";

export function useHotspotVouchers(status?: HotspotVoucherStatus, batchId?: string) {
  const params = new URLSearchParams();
  if (status) params.set("status", status);
  if (batchId) params.set("batchId", batchId);
  const query = params.toString();
  return useQuery<HotspotVoucher[]>({
    queryKey: ["hotspot", "vouchers", status ?? "all", batchId ?? "all"],
    queryFn: () => api.get<HotspotVoucher[]>(`/hotspot/vouchers${query ? `?${query}` : ""}`),
  });
}

export function useHotspotVoucherBatches() {
  return useQuery<HotspotVoucherBatch[]>({
    queryKey: ["hotspot", "voucher-batches"],
    queryFn: () => api.get<HotspotVoucherBatch[]>("/hotspot/voucher-batches"),
  });
}

export function useHotspotVoucherBatch(id: string) {
  return useQuery<HotspotVoucherBatch>({
    queryKey: ["hotspot", "voucher-batches", id],
    queryFn: () => api.get<HotspotVoucherBatch>(`/hotspot/voucher-batches/${encodeURIComponent(id)}`),
    enabled: !!id,
  });
}
