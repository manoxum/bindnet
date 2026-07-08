-- Migration idempotente: guarda a unidade escolhida na emissao do
-- lote (kbit/mbit/gbit/kbyte/mbyte/gbyte, mesmo conjunto de RateUnit
-- do frontend) para exibir o valor por voucher na mesma unidade
-- emitida, em vez de sempre converter para GB (ver
-- services/backend/hotspot_voucher_batches.go).

ALTER TABLE "hotspot_voucher_batches" ADD COLUMN IF NOT EXISTS "amount_unit" TEXT NOT NULL DEFAULT 'gbyte'
	CHECK ("amount_unit" IN ('kbit','mbit','gbit','kbyte','mbyte','gbyte'));
