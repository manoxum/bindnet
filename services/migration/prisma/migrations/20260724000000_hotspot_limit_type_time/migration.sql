-- Novo tipo de limitacao 'time' (limitacao por tempo), alem de
-- unlimited/credit/quota/custom - ver services/backend/internal/hotspot/
-- store/types.go e hotspot_reconcile_time.go. Dois modos:
--   * budget:   saldo de tempo de conexao (segundos), gasto enquanto o
--               device fica associado ao Wi-Fi (mesmo ocioso); ao zerar,
--               bloqueia (traffic block + portal cativo, igual credito).
--   * deadline: acesso ate um instante (deadline_at); passado ele,
--               bloqueia.
-- Espelha a estrutura do credito (hotspot_device_credit + colunas de
-- politica no perfil): o SALDO/estado vive em hotspot_device_time; a
-- POLITICA (modo/recarga/deadline) vem do perfil quando o device herda,
-- ou do proprio override do device quando configura manualmente.

-- Postgres nao tem "ALTER CONSTRAINT" para CHECK: idempotente via
-- DROP CONSTRAINT IF EXISTS + ADD CONSTRAINT (mesmo precedente de
-- 20260713020000_hotspot_limit_type_custom). Acrescenta 'time' ao
-- conjunto ja existente (unlimited/credit/quota/custom).
ALTER TABLE "hotspot_device_limits" DROP CONSTRAINT IF EXISTS "hotspot_device_limits_limit_type_check";
ALTER TABLE "hotspot_device_limits" ADD CONSTRAINT "hotspot_device_limits_limit_type_check" CHECK ("limit_type" IN ('unlimited','credit','quota','custom','time'));

ALTER TABLE "hotspot_profiles" DROP CONSTRAINT IF EXISTS "hotspot_profiles_limit_type_check";
ALTER TABLE "hotspot_profiles" ADD CONSTRAINT "hotspot_profiles_limit_type_check" CHECK ("limit_type" IN ('unlimited','credit','quota','custom','time'));

-- Saldo/estado de tempo por dispositivo (espelha hotspot_device_credit).
CREATE TABLE IF NOT EXISTS "hotspot_device_time" ();
ALTER TABLE "hotspot_device_time" ADD COLUMN IF NOT EXISTS "mac_address" TEXT PRIMARY KEY;
ALTER TABLE "hotspot_device_time" ADD COLUMN IF NOT EXISTS "enabled" BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE "hotspot_device_time" ADD COLUMN IF NOT EXISTS "mode" TEXT NOT NULL DEFAULT 'budget' CHECK ("mode" IN ('budget','deadline'));
ALTER TABLE "hotspot_device_time" ADD COLUMN IF NOT EXISTS "balance_seconds" BIGINT NOT NULL DEFAULT 0;
ALTER TABLE "hotspot_device_time" ADD COLUMN IF NOT EXISTS "recharge_seconds" BIGINT;
ALTER TABLE "hotspot_device_time" ADD COLUMN IF NOT EXISTS "recharge_period" TEXT CHECK ("recharge_period" IN ('daily','weekly','monthly'));
ALTER TABLE "hotspot_device_time" ADD COLUMN IF NOT EXISTS "plafond_seconds" BIGINT;
ALTER TABLE "hotspot_device_time" ADD COLUMN IF NOT EXISTS "next_recharge_at" TIMESTAMPTZ;
ALTER TABLE "hotspot_device_time" ADD COLUMN IF NOT EXISTS "deadline_at" TIMESTAMPTZ;
ALTER TABLE "hotspot_device_time" ADD COLUMN IF NOT EXISTS "last_charged_at" TIMESTAMPTZ;
ALTER TABLE "hotspot_device_time" ADD COLUMN IF NOT EXISTS "blocked_by_time" BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE "hotspot_device_time" ADD COLUMN IF NOT EXISTS "configured" BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE "hotspot_device_time" ADD COLUMN IF NOT EXISTS "updated_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;

-- Politica de tempo no perfil (espelha as colunas credit_* do perfil).
ALTER TABLE "hotspot_profiles" ADD COLUMN IF NOT EXISTS "time_mode" TEXT CHECK ("time_mode" IN ('budget','deadline'));
ALTER TABLE "hotspot_profiles" ADD COLUMN IF NOT EXISTS "time_recharge_seconds" BIGINT;
ALTER TABLE "hotspot_profiles" ADD COLUMN IF NOT EXISTS "time_recharge_period" TEXT CHECK ("time_recharge_period" IN ('daily','weekly','monthly'));
ALTER TABLE "hotspot_profiles" ADD COLUMN IF NOT EXISTS "time_plafond_seconds" BIGINT;
ALTER TABLE "hotspot_profiles" ADD COLUMN IF NOT EXISTS "time_deadline_at" TIMESTAMPTZ;
