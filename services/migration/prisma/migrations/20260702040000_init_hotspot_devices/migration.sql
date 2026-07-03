-- Migration idempotente para dispositivos do hotspot:
-- bloqueios persistentes por MAC e cache local de identificacao.

-- CreateTable
CREATE TABLE IF NOT EXISTS "hotspot_blocked_devices" ();

ALTER TABLE "hotspot_blocked_devices" ADD COLUMN IF NOT EXISTS "mac_address" TEXT PRIMARY KEY;
ALTER TABLE "hotspot_blocked_devices" ADD COLUMN IF NOT EXISTS "note" TEXT;
ALTER TABLE "hotspot_blocked_devices" ADD COLUMN IF NOT EXISTS "blocked_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;

-- CreateTable
CREATE TABLE IF NOT EXISTS "hotspot_device_info" ();

ALTER TABLE "hotspot_device_info" ADD COLUMN IF NOT EXISTS "mac_address" TEXT PRIMARY KEY;
ALTER TABLE "hotspot_device_info" ADD COLUMN IF NOT EXISTS "vendor" TEXT;
ALTER TABLE "hotspot_device_info" ADD COLUMN IF NOT EXISTS "device_name" TEXT;
ALTER TABLE "hotspot_device_info" ADD COLUMN IF NOT EXISTS "os_name" TEXT;
ALTER TABLE "hotspot_device_info" ADD COLUMN IF NOT EXISTS "confidence" INTEGER;
ALTER TABLE "hotspot_device_info" ADD COLUMN IF NOT EXISTS "fetched_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;
