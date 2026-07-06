ALTER TABLE "hotspot_blocked_devices" ADD COLUMN IF NOT EXISTS "mode" TEXT NOT NULL DEFAULT 'deauth' CHECK ("mode" IN ('deauth','traffic'));
