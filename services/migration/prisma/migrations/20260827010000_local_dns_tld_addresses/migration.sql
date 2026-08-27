-- Cada TLD local pode preservar a estrategia split-horizon (address NULL)
-- ou responder um IPv4 estatico para todos os dominios sob seu sufixo.
CREATE TABLE IF NOT EXISTS "local_dns_tlds" (
  "tld" TEXT PRIMARY KEY,
  "address" INET
);

-- Migra a lista existente para o novo formato sem alterar a estrategia
-- historica: todo TLD atual inicia no modo automatico (address NULL).
INSERT INTO "local_dns_tlds" ("tld", "address")
SELECT lower(trim(tld)), NULL
FROM "dns_config", regexp_split_to_table(value, ',') AS tld
WHERE key = 'DNS_LOCAL_TLDS' AND trim(tld) <> ''
ON CONFLICT ("tld") DO NOTHING;
