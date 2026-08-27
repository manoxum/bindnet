-- Distingue registros declarados pelo operador daqueles criados sob demanda
-- pelo resolvedor e registra o TLD que originou cada entrada dinamica.
ALTER TABLE "local_dns_records"
  ADD COLUMN IF NOT EXISTS "source" TEXT NOT NULL DEFAULT 'dynamic',
  ADD COLUMN IF NOT EXISTS "origin_tld" TEXT;

-- Os IPs ja configurados representam uma decisao explicita do operador.
-- Entradas historicas sem IP nao podem ser distinguidas com seguranca e
-- seguem como dinamicas para serem renovadas na proxima troca de TLD.
UPDATE "local_dns_records"
SET "source" = 'manual'
WHERE "address" IS NOT NULL;

ALTER TABLE "local_dns_records"
  DROP CONSTRAINT IF EXISTS "local_dns_records_source_check";

ALTER TABLE "local_dns_records"
  ADD CONSTRAINT "local_dns_records_source_check"
  CHECK ("source" IN ('manual', 'dynamic'));

CREATE INDEX IF NOT EXISTS "local_dns_records_source_origin_tld_idx"
  ON "local_dns_records" ("source", "origin_tld");
