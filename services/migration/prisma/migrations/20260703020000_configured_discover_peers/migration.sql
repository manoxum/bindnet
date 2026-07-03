-- Peers efetivos da malha agora sao estado operacional do banco.
-- Assim `docker compose down -v` remove os peers adicionados pelo painel.

CREATE TABLE IF NOT EXISTS "discover_configured_peers" ();

ALTER TABLE "discover_configured_peers" ADD COLUMN IF NOT EXISTS "id" UUID PRIMARY KEY DEFAULT gen_random_uuid();
ALTER TABLE "discover_configured_peers" ADD COLUMN IF NOT EXISTS "address" TEXT NOT NULL UNIQUE;
ALTER TABLE "discover_configured_peers" ADD COLUMN IF NOT EXISTS "source" TEXT NOT NULL DEFAULT 'manual';
ALTER TABLE "discover_configured_peers" ADD COLUMN IF NOT EXISTS "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE "discover_configured_peers" ADD COLUMN IF NOT EXISTS "updated_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;
