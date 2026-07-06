-- Configuracao operacional do hotspot administrada pelo painel.
-- Estes valores nao ficam em .env.main/.env.example; o backend le daqui
-- e envia ao worker para montar o ambiente do container hotspot/dns-provider.

CREATE TABLE IF NOT EXISTS "hotspot_config" ();

ALTER TABLE "hotspot_config" ADD COLUMN IF NOT EXISTS "key" TEXT PRIMARY KEY;
ALTER TABLE "hotspot_config" ADD COLUMN IF NOT EXISTS "value" TEXT NOT NULL;
ALTER TABLE "hotspot_config" ADD COLUMN IF NOT EXISTS "updated_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;
