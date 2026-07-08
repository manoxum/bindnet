-- Migration idempotente para lotes de vouchers - agrupa os vouchers
-- emitidos juntos numa mesma chamada para permitir listagem e
-- impressao em lote (ver services/backend/hotspot_vouchers.go).

CREATE TABLE IF NOT EXISTS "hotspot_voucher_batches" ();

ALTER TABLE "hotspot_voucher_batches" ADD COLUMN IF NOT EXISTS "id" TEXT PRIMARY KEY;
ALTER TABLE "hotspot_voucher_batches" ADD COLUMN IF NOT EXISTS "amount_bytes" BIGINT NOT NULL;
ALTER TABLE "hotspot_voucher_batches" ADD COLUMN IF NOT EXISTS "quantity" INT NOT NULL;
ALTER TABLE "hotspot_voucher_batches" ADD COLUMN IF NOT EXISTS "note" TEXT;
ALTER TABLE "hotspot_voucher_batches" ADD COLUMN IF NOT EXISTS "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;

CREATE INDEX IF NOT EXISTS "hotspot_voucher_batches_created_at_idx" ON "hotspot_voucher_batches" ("created_at" DESC);

ALTER TABLE "hotspot_vouchers" ADD COLUMN IF NOT EXISTS "batch_id" TEXT REFERENCES "hotspot_voucher_batches"("id");

CREATE INDEX IF NOT EXISTS "hotspot_vouchers_batch_id_idx" ON "hotspot_vouchers" ("batch_id");
