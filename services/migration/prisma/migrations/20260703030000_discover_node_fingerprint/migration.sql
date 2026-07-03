-- Identidade persistente do servidor Bindnet local e fingerprint nas rotas.

ALTER TABLE "discover_routes" ADD COLUMN IF NOT EXISTS "owner_fingerprint" TEXT;

CREATE TABLE IF NOT EXISTS "discover_node_identity" ();
ALTER TABLE "discover_node_identity" ADD COLUMN IF NOT EXISTS "id" INTEGER PRIMARY KEY DEFAULT 1;
ALTER TABLE "discover_node_identity" ADD COLUMN IF NOT EXISTS "fingerprint" TEXT NOT NULL UNIQUE;
ALTER TABLE "discover_node_identity" ADD COLUMN IF NOT EXISTS "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE "discover_node_identity" ADD COLUMN IF NOT EXISTS "updated_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;

DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conname = 'discover_node_identity_singleton'
  ) THEN
    ALTER TABLE "discover_node_identity"
      ADD CONSTRAINT "discover_node_identity_singleton" CHECK ("id" = 1);
  END IF;
END $$;

ALTER TABLE "discover_peers" ADD COLUMN IF NOT EXISTS "fingerprint" TEXT;
ALTER TABLE "discover_configured_peers" ADD COLUMN IF NOT EXISTS "node_name" TEXT;
ALTER TABLE "discover_configured_peers" ADD COLUMN IF NOT EXISTS "fingerprint" TEXT;
