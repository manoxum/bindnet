-- Migration idempotente para vouchers (cartoes de recarga) do
-- hotspot - emitidos pelo admin, resgatados pelo proprio dispositivo
-- sem login (ver services/backend/hotspot_vouchers.go e
-- hotspot_portal.go).

CREATE TABLE IF NOT EXISTS "hotspot_vouchers" ();

ALTER TABLE "hotspot_vouchers" ADD COLUMN IF NOT EXISTS "code" TEXT PRIMARY KEY;
ALTER TABLE "hotspot_vouchers" ADD COLUMN IF NOT EXISTS "amount_bytes" BIGINT NOT NULL;
ALTER TABLE "hotspot_vouchers" ADD COLUMN IF NOT EXISTS "status" TEXT NOT NULL DEFAULT 'active' CHECK ("status" IN ('active','redeemed','revoked'));
ALTER TABLE "hotspot_vouchers" ADD COLUMN IF NOT EXISTS "note" TEXT;
ALTER TABLE "hotspot_vouchers" ADD COLUMN IF NOT EXISTS "redeemed_by_mac" TEXT;
ALTER TABLE "hotspot_vouchers" ADD COLUMN IF NOT EXISTS "redeemed_at" TIMESTAMPTZ;
ALTER TABLE "hotspot_vouchers" ADD COLUMN IF NOT EXISTS "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;

CREATE INDEX IF NOT EXISTS "hotspot_vouchers_status_idx" ON "hotspot_vouchers" ("status", "created_at" DESC);
