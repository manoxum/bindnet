-- Migration idempotente para limites de trafego (global + por
-- dispositivo), contadores de uso acumulado e credito por dispositivo
-- do hotspot Wi-Fi (ver services/backend/hotspot_limits.go,
-- hotspot_traffic.go, hotspot_credit.go).

-- Sequence usada para atribuir um fwmark/classid HTB unico por
-- dispositivo (nunca hash de MAC, para evitar colisao).
CREATE SEQUENCE IF NOT EXISTS hotspot_device_fwmark_seq START WITH 100;

-- CreateTable
CREATE TABLE IF NOT EXISTS "hotspot_global_limits" ();

ALTER TABLE "hotspot_global_limits" ADD COLUMN IF NOT EXISTS "id" TEXT PRIMARY KEY DEFAULT 'global' CHECK ("id" = 'global');
ALTER TABLE "hotspot_global_limits" ADD COLUMN IF NOT EXISTS "download_rate_mbps" INTEGER;
ALTER TABLE "hotspot_global_limits" ADD COLUMN IF NOT EXISTS "upload_rate_mbps" INTEGER;
ALTER TABLE "hotspot_global_limits" ADD COLUMN IF NOT EXISTS "quota_bytes" BIGINT;
ALTER TABLE "hotspot_global_limits" ADD COLUMN IF NOT EXISTS "quota_period" TEXT CHECK ("quota_period" IN ('daily','weekly','monthly'));
ALTER TABLE "hotspot_global_limits" ADD COLUMN IF NOT EXISTS "quota_throttle_download_mbps" INTEGER;
ALTER TABLE "hotspot_global_limits" ADD COLUMN IF NOT EXISTS "quota_throttle_upload_mbps" INTEGER;
ALTER TABLE "hotspot_global_limits" ADD COLUMN IF NOT EXISTS "updated_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;

-- CreateTable
CREATE TABLE IF NOT EXISTS "hotspot_device_limits" ();

ALTER TABLE "hotspot_device_limits" ADD COLUMN IF NOT EXISTS "mac_address" TEXT PRIMARY KEY;
ALTER TABLE "hotspot_device_limits" ADD COLUMN IF NOT EXISTS "download_rate_mbps" INTEGER;
ALTER TABLE "hotspot_device_limits" ADD COLUMN IF NOT EXISTS "upload_rate_mbps" INTEGER;
ALTER TABLE "hotspot_device_limits" ADD COLUMN IF NOT EXISTS "quota_bytes" BIGINT;
ALTER TABLE "hotspot_device_limits" ADD COLUMN IF NOT EXISTS "quota_period" TEXT CHECK ("quota_period" IN ('daily','weekly','monthly'));
ALTER TABLE "hotspot_device_limits" ADD COLUMN IF NOT EXISTS "quota_throttle_download_mbps" INTEGER;
ALTER TABLE "hotspot_device_limits" ADD COLUMN IF NOT EXISTS "quota_throttle_upload_mbps" INTEGER;
ALTER TABLE "hotspot_device_limits" ADD COLUMN IF NOT EXISTS "updated_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;

-- CreateTable
CREATE TABLE IF NOT EXISTS "hotspot_global_traffic" ();

ALTER TABLE "hotspot_global_traffic" ADD COLUMN IF NOT EXISTS "id" TEXT PRIMARY KEY DEFAULT 'global' CHECK ("id" = 'global');
ALTER TABLE "hotspot_global_traffic" ADD COLUMN IF NOT EXISTS "period_start" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE "hotspot_global_traffic" ADD COLUMN IF NOT EXISTS "period_end" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE "hotspot_global_traffic" ADD COLUMN IF NOT EXISTS "download_bytes" BIGINT NOT NULL DEFAULT 0;
ALTER TABLE "hotspot_global_traffic" ADD COLUMN IF NOT EXISTS "upload_bytes" BIGINT NOT NULL DEFAULT 0;
ALTER TABLE "hotspot_global_traffic" ADD COLUMN IF NOT EXISTS "last_download_counter" BIGINT NOT NULL DEFAULT 0;
ALTER TABLE "hotspot_global_traffic" ADD COLUMN IF NOT EXISTS "last_upload_counter" BIGINT NOT NULL DEFAULT 0;
ALTER TABLE "hotspot_global_traffic" ADD COLUMN IF NOT EXISTS "throttled" BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE "hotspot_global_traffic" ADD COLUMN IF NOT EXISTS "updated_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;

-- CreateTable
-- Criada de forma preguicosa (lazy) na 1a vez que qualquer rota
-- precisa rastrear o dispositivo (reconciliacao ou abertura da pagina
-- de detalhe) - nao depende de o admin ter configurado limite, pois
-- tambem alimenta a velocidade ao vivo (accounting sem shaping).
CREATE TABLE IF NOT EXISTS "hotspot_device_traffic" ();

ALTER TABLE "hotspot_device_traffic" ADD COLUMN IF NOT EXISTS "mac_address" TEXT PRIMARY KEY;
ALTER TABLE "hotspot_device_traffic" ADD COLUMN IF NOT EXISTS "fwmark" INTEGER NOT NULL UNIQUE DEFAULT nextval('hotspot_device_fwmark_seq');
ALTER TABLE "hotspot_device_traffic" ADD COLUMN IF NOT EXISTS "period_start" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE "hotspot_device_traffic" ADD COLUMN IF NOT EXISTS "period_end" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE "hotspot_device_traffic" ADD COLUMN IF NOT EXISTS "download_bytes" BIGINT NOT NULL DEFAULT 0;
ALTER TABLE "hotspot_device_traffic" ADD COLUMN IF NOT EXISTS "upload_bytes" BIGINT NOT NULL DEFAULT 0;
ALTER TABLE "hotspot_device_traffic" ADD COLUMN IF NOT EXISTS "last_download_counter" BIGINT NOT NULL DEFAULT 0;
ALTER TABLE "hotspot_device_traffic" ADD COLUMN IF NOT EXISTS "last_upload_counter" BIGINT NOT NULL DEFAULT 0;
ALTER TABLE "hotspot_device_traffic" ADD COLUMN IF NOT EXISTS "throttled" BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE "hotspot_device_traffic" ADD COLUMN IF NOT EXISTS "updated_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;

-- CreateTable
-- blocked_by_credit fica separado de hotspot_blocked_devices (bloqueio
-- manual do admin) de proposito - os dois mecanismos de bloqueio nao
-- devem se confundir nem se sobrescrever um ao outro.
CREATE TABLE IF NOT EXISTS "hotspot_device_credit" ();

ALTER TABLE "hotspot_device_credit" ADD COLUMN IF NOT EXISTS "mac_address" TEXT PRIMARY KEY;
ALTER TABLE "hotspot_device_credit" ADD COLUMN IF NOT EXISTS "enabled" BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE "hotspot_device_credit" ADD COLUMN IF NOT EXISTS "balance_bytes" BIGINT NOT NULL DEFAULT 0;
ALTER TABLE "hotspot_device_credit" ADD COLUMN IF NOT EXISTS "recharge_amount_bytes" BIGINT;
ALTER TABLE "hotspot_device_credit" ADD COLUMN IF NOT EXISTS "recharge_period" TEXT CHECK ("recharge_period" IN ('daily','weekly','monthly'));
ALTER TABLE "hotspot_device_credit" ADD COLUMN IF NOT EXISTS "plafond_bytes" BIGINT;
ALTER TABLE "hotspot_device_credit" ADD COLUMN IF NOT EXISTS "next_recharge_at" TIMESTAMPTZ;
ALTER TABLE "hotspot_device_credit" ADD COLUMN IF NOT EXISTS "blocked_by_credit" BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE "hotspot_device_credit" ADD COLUMN IF NOT EXISTS "updated_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;
