-- Um registro pode opcionalmente fixar um A record. Sem endereco, continua
-- usando o loopback reservado pelo split-horizon.
ALTER TABLE "local_dns_records"
  ADD COLUMN IF NOT EXISTS "address" INET;
