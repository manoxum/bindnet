-- Migration idempotente (mesmo padrao das anteriores).

-- CreateTable
CREATE TABLE IF NOT EXISTS "discover_peers" ();

ALTER TABLE "discover_peers" ADD COLUMN IF NOT EXISTS "id" UUID PRIMARY KEY DEFAULT gen_random_uuid();
ALTER TABLE "discover_peers" ADD COLUMN IF NOT EXISTS "address" TEXT NOT NULL UNIQUE;
ALTER TABLE "discover_peers" ADD COLUMN IF NOT EXISTS "node_name" TEXT NOT NULL;
ALTER TABLE "discover_peers" ADD COLUMN IF NOT EXISTS "source" TEXT NOT NULL DEFAULT 'manual-scan';
ALTER TABLE "discover_peers" ADD COLUMN IF NOT EXISTS "last_seen_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE "discover_peers" ADD COLUMN IF NOT EXISTS "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;
