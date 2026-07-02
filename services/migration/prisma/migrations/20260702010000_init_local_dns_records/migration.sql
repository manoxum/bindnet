-- Migration idempotente (mesmo padrao de 20260702000000_init_certificates):
-- CREATE TABLE sem colunas, toda coluna via ALTER TABLE ADD COLUMN IF NOT
-- EXISTS, indices via CREATE INDEX IF NOT EXISTS.

-- CreateTable
CREATE TABLE IF NOT EXISTS "local_dns_records" ();

ALTER TABLE "local_dns_records" ADD COLUMN IF NOT EXISTS "id" UUID PRIMARY KEY DEFAULT gen_random_uuid();
ALTER TABLE "local_dns_records" ADD COLUMN IF NOT EXISTS "hostname" TEXT NOT NULL UNIQUE;
ALTER TABLE "local_dns_records" ADD COLUMN IF NOT EXISTS "loopback_offset" INTEGER NOT NULL UNIQUE;
ALTER TABLE "local_dns_records" ADD COLUMN IF NOT EXISTS "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;

-- CreateSequence
-- Comeca em 2: o offset 1 (127.0.0.1) fica reservado como IP de loopback
-- generico/self, nunca alocado a um hostname especifico.
CREATE SEQUENCE IF NOT EXISTS "local_dns_records_offset_seq" START 2;
