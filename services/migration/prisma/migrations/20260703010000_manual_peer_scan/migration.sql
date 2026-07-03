-- Busca de peers agora e sempre manual/acionada pelo operador.

ALTER TABLE "discover_peers"
  ALTER COLUMN "source" SET DEFAULT 'manual-scan';

UPDATE "discover_peers"
SET "source" = 'manual-scan'
WHERE "source" = 'broadcast';
