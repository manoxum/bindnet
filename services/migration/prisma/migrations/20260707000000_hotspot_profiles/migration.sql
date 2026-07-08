-- Migration idempotente para perfis de dispositivo do hotspot (bundle
-- nomeado e reutilizavel de limites de trafego + politica de credito,
-- ver services/backend/hotspot_profiles.go). Mesmo shape de
-- hotspot_device_limits para os campos de taxa/cota/throttle, e um
-- subconjunto de hotspot_device_credit para a politica de credito (so
-- politica, nunca saldo/estado - isso continua exclusivo de
-- hotspot_device_credit).

CREATE TABLE IF NOT EXISTS "hotspot_profiles" ();

ALTER TABLE "hotspot_profiles" ADD COLUMN IF NOT EXISTS "id" UUID PRIMARY KEY DEFAULT gen_random_uuid();
ALTER TABLE "hotspot_profiles" ADD COLUMN IF NOT EXISTS "name" TEXT NOT NULL UNIQUE;
ALTER TABLE "hotspot_profiles" ADD COLUMN IF NOT EXISTS "is_default" BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE "hotspot_profiles" ADD COLUMN IF NOT EXISTS "download_rate_value" INTEGER;
ALTER TABLE "hotspot_profiles" ADD COLUMN IF NOT EXISTS "download_rate_unit" TEXT NOT NULL DEFAULT 'mbit' CHECK ("download_rate_unit" IN ('kbit','mbit','gbit','kbyte','mbyte','gbyte'));
ALTER TABLE "hotspot_profiles" ADD COLUMN IF NOT EXISTS "upload_rate_value" INTEGER;
ALTER TABLE "hotspot_profiles" ADD COLUMN IF NOT EXISTS "upload_rate_unit" TEXT NOT NULL DEFAULT 'mbit' CHECK ("upload_rate_unit" IN ('kbit','mbit','gbit','kbyte','mbyte','gbyte'));
ALTER TABLE "hotspot_profiles" ADD COLUMN IF NOT EXISTS "quota_bytes" BIGINT;
ALTER TABLE "hotspot_profiles" ADD COLUMN IF NOT EXISTS "quota_period" TEXT CHECK ("quota_period" IN ('daily','weekly','monthly'));
ALTER TABLE "hotspot_profiles" ADD COLUMN IF NOT EXISTS "quota_throttle_download_value" INTEGER;
ALTER TABLE "hotspot_profiles" ADD COLUMN IF NOT EXISTS "quota_throttle_download_unit" TEXT NOT NULL DEFAULT 'mbit' CHECK ("quota_throttle_download_unit" IN ('kbit','mbit','gbit','kbyte','mbyte','gbyte'));
ALTER TABLE "hotspot_profiles" ADD COLUMN IF NOT EXISTS "quota_throttle_upload_value" INTEGER;
ALTER TABLE "hotspot_profiles" ADD COLUMN IF NOT EXISTS "quota_throttle_upload_unit" TEXT NOT NULL DEFAULT 'mbit' CHECK ("quota_throttle_upload_unit" IN ('kbit','mbit','gbit','kbyte','mbyte','gbyte'));

ALTER TABLE "hotspot_profiles" ADD COLUMN IF NOT EXISTS "credit_enabled" BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE "hotspot_profiles" ADD COLUMN IF NOT EXISTS "credit_recharge_amount_bytes" BIGINT;
ALTER TABLE "hotspot_profiles" ADD COLUMN IF NOT EXISTS "credit_recharge_period" TEXT CHECK ("credit_recharge_period" IN ('daily','weekly','monthly'));
ALTER TABLE "hotspot_profiles" ADD COLUMN IF NOT EXISTS "credit_plafond_bytes" BIGINT;

ALTER TABLE "hotspot_profiles" ADD COLUMN IF NOT EXISTS "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE "hotspot_profiles" ADD COLUMN IF NOT EXISTS "updated_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;

-- So um perfil pode ser o default (reforco a nivel de banco - a
-- protecao "nao deletar/mudar o default" e a nivel de app, ver
-- services/backend/hotspot_profiles_store.go).
CREATE UNIQUE INDEX IF NOT EXISTS "hotspot_profiles_single_default_idx" ON "hotspot_profiles" ("is_default") WHERE "is_default" = true;

-- Seed do perfil "Padrao" com id fixo (mesmo idioma de
-- hotspot_global_limits.id='global') - e nele que
-- hotspot_device_info.profile_id aponta por default (ver migration
-- 20260707010000_hotspot_profile_assignment).
INSERT INTO "hotspot_profiles" ("id", "name", "is_default")
VALUES ('00000000-0000-0000-0000-000000000001', 'Padrão', true)
ON CONFLICT ("id") DO NOTHING;
