-- Migration idempotente: seguro re-executar contra um banco que já
-- tenha essas tabelas/colunas (ex.: aplicação manual fora do
-- rastreamento normal do Prisma). CREATE TABLE não declara nenhuma
-- coluna - toda coluna, incluindo "id", é adicionada via ALTER TABLE
-- ADD COLUMN IF NOT EXISTS (ver regra em RULE.md).

-- CreateTable
CREATE TABLE IF NOT EXISTS "ca" ();

ALTER TABLE "ca" ADD COLUMN IF NOT EXISTS "id" UUID PRIMARY KEY DEFAULT gen_random_uuid();
ALTER TABLE "ca" ADD COLUMN IF NOT EXISTS "common_name" TEXT NOT NULL;
ALTER TABLE "ca" ADD COLUMN IF NOT EXISTS "organization" TEXT NOT NULL;
ALTER TABLE "ca" ADD COLUMN IF NOT EXISTS "serial_number" TEXT NOT NULL;
ALTER TABLE "ca" ADD COLUMN IF NOT EXISTS "certificate_pem" TEXT NOT NULL;
ALTER TABLE "ca" ADD COLUMN IF NOT EXISTS "private_key_pem" TEXT NOT NULL;
ALTER TABLE "ca" ADD COLUMN IF NOT EXISTS "issued_at" TIMESTAMPTZ NOT NULL;
ALTER TABLE "ca" ADD COLUMN IF NOT EXISTS "expires_at" TIMESTAMPTZ NOT NULL;
ALTER TABLE "ca" ADD COLUMN IF NOT EXISTS "source" TEXT NOT NULL;
ALTER TABLE "ca" ADD COLUMN IF NOT EXISTS "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;

-- CreateTable
CREATE TABLE IF NOT EXISTS "certificates" ();

ALTER TABLE "certificates" ADD COLUMN IF NOT EXISTS "id" UUID PRIMARY KEY DEFAULT gen_random_uuid();
ALTER TABLE "certificates" ADD COLUMN IF NOT EXISTS "domain" TEXT NOT NULL;
ALTER TABLE "certificates" ADD COLUMN IF NOT EXISTS "common_name" TEXT NOT NULL;
ALTER TABLE "certificates" ADD COLUMN IF NOT EXISTS "dns_names" TEXT[];
ALTER TABLE "certificates" ADD COLUMN IF NOT EXISTS "ip_addresses" TEXT[];
ALTER TABLE "certificates" ADD COLUMN IF NOT EXISTS "serial_number" TEXT NOT NULL UNIQUE;
ALTER TABLE "certificates" ADD COLUMN IF NOT EXISTS "certificate_pem" TEXT NOT NULL;
ALTER TABLE "certificates" ADD COLUMN IF NOT EXISTS "private_key_pem" TEXT NOT NULL;
ALTER TABLE "certificates" ADD COLUMN IF NOT EXISTS "issued_at" TIMESTAMPTZ NOT NULL;
ALTER TABLE "certificates" ADD COLUMN IF NOT EXISTS "expires_at" TIMESTAMPTZ NOT NULL;
ALTER TABLE "certificates" ADD COLUMN IF NOT EXISTS "revoked_at" TIMESTAMPTZ;
ALTER TABLE "certificates" ADD COLUMN IF NOT EXISTS "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;

-- CreateIndex
CREATE INDEX IF NOT EXISTS "certificates_domain_idx" ON "certificates"("domain");
