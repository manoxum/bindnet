-- Separa o nome amigavel do certificado de seu dominio principal e
-- preserva o periodo escolhido para preencher o formulario de reemissao.
ALTER TABLE "certificates" ADD COLUMN IF NOT EXISTS "name" TEXT;
UPDATE "certificates" SET "name" = "domain" WHERE "name" IS NULL;
ALTER TABLE "certificates" ALTER COLUMN "name" SET NOT NULL;

ALTER TABLE "certificates" ADD COLUMN IF NOT EXISTS "validity_quantity" INTEGER;
ALTER TABLE "certificates" ADD COLUMN IF NOT EXISTS "validity_unit" TEXT;
UPDATE "certificates"
SET "validity_quantity" = GREATEST(1, CEIL(EXTRACT(EPOCH FROM ("expires_at" - "issued_at")) / 86400)::INTEGER),
    "validity_unit" = 'days'
WHERE "validity_quantity" IS NULL OR "validity_unit" IS NULL;
ALTER TABLE "certificates" ALTER COLUMN "validity_quantity" SET DEFAULT 2;
ALTER TABLE "certificates" ALTER COLUMN "validity_quantity" SET NOT NULL;
ALTER TABLE "certificates" ALTER COLUMN "validity_unit" SET DEFAULT 'years';
ALTER TABLE "certificates" ALTER COLUMN "validity_unit" SET NOT NULL;

CREATE INDEX IF NOT EXISTS "certificates_name_idx" ON "certificates"("name");
