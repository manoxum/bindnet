import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { api, ApiError } from "@/lib/api";
import type { HotspotVoucherIssueRequest, HotspotVoucherIssueResponse } from "@/components/hotspot/hotspot-voucher-types";

function onVoucherError(error: unknown) {
  toast.error(error instanceof ApiError ? error.message : "Falha ao processar voucher");
}

export function useHotspotVoucherMutations() {
  const queryClient = useQueryClient();
  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: ["hotspot", "vouchers"] });
    queryClient.invalidateQueries({ queryKey: ["hotspot", "voucher-batches"] });
  };

  const issue = useMutation({
    mutationFn: (request: HotspotVoucherIssueRequest) =>
      api.post<HotspotVoucherIssueResponse>("/hotspot/vouchers", request),
    onSuccess: (response) => {
      toast.success(`${response.vouchers.length} voucher(s) emitido(s).`);
      invalidate();
    },
    onError: onVoucherError,
  });

  const revoke = useMutation({
    mutationFn: (code: string) => api.del(`/hotspot/vouchers/${encodeURIComponent(code)}`),
    onSuccess: () => {
      toast.success("Voucher anulado.");
      invalidate();
    },
    onError: onVoucherError,
  });

  return { issue, revoke };
}
