import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { api, ApiError } from "@/lib/api";
import type { PortalMeResponse } from "@/components/portal/portal-types";

export function useRedeemVoucher() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (code: string) => api.post<PortalMeResponse>("/hotspot/portal/vouchers/redeem", { code }),
    onSuccess: () => {
      toast.success("Voucher resgatado com sucesso.");
      queryClient.invalidateQueries({ queryKey: ["portal"] });
    },
    onError: (error: unknown) => {
      toast.error(error instanceof ApiError ? error.message : "Falha ao resgatar voucher");
    },
  });
}
