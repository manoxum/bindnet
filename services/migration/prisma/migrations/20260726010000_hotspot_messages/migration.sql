-- Avisos/mensagens que o operador envia aos dispositivos conectados ao
-- hotspot (ver services/backend/internal/hotspot/hotspot_messages.go e
-- RULE.md). A entrega base e "pull": o dispositivo ve a mensagem na
-- pagina de autoatendimento publica bindnet.local.com (identificado pelo MAC de
-- origem, igual aos vouchers). Mensagens marcadas "urgent" tambem sao
-- "empurradas" reusando o portal cativo por-MAC ja existente do bloqueio
-- de credito/cota (applyCaptivePortalRedirect) - nenhuma alteracao de DNS.
-- Migration idempotente (CREATE TABLE sem colunas + ADD COLUMN IF NOT
-- EXISTS para cada uma, incluindo id; ver RULE.md e a migration
-- 20260702000000_init_certificates).

CREATE TABLE IF NOT EXISTS "hotspot_messages" ();
ALTER TABLE "hotspot_messages" ADD COLUMN IF NOT EXISTS "id" UUID PRIMARY KEY DEFAULT gen_random_uuid();
ALTER TABLE "hotspot_messages" ADD COLUMN IF NOT EXISTS "title" TEXT NOT NULL DEFAULT '';
ALTER TABLE "hotspot_messages" ADD COLUMN IF NOT EXISTS "body" TEXT NOT NULL;
-- target_mac NULL = broadcast (todos os conectados); preenchido = aviso
-- direcionado a um unico dispositivo.
ALTER TABLE "hotspot_messages" ADD COLUMN IF NOT EXISTS "target_mac" TEXT;
-- urgent: alem de aparecer em bindnet.local.com, forca o balao "Entrar na rede" no
-- dispositivo via portal cativo (best-effort, so porta 80/HTTP).
ALTER TABLE "hotspot_messages" ADD COLUMN IF NOT EXISTS "urgent" BOOLEAN NOT NULL DEFAULT false;
-- active=false = removida pelo admin (soft delete); mantida para trilha.
ALTER TABLE "hotspot_messages" ADD COLUMN IF NOT EXISTS "active" BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE "hotspot_messages" ADD COLUMN IF NOT EXISTS "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;
-- expires_at NULL = sem expiracao.
ALTER TABLE "hotspot_messages" ADD COLUMN IF NOT EXISTS "expires_at" TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS "hotspot_messages_active_idx" ON "hotspot_messages" ("active");
CREATE INDEX IF NOT EXISTS "hotspot_messages_target_mac_idx" ON "hotspot_messages" ("target_mac");

-- Confirmacao de leitura por dispositivo: um MAC so ve cada mensagem ate
-- marca-la lida (botao "Ok" no portal). Unicidade (message_id, mac) por
-- indice unico (suporta IF NOT EXISTS, ao contrario de constraint composta).
CREATE TABLE IF NOT EXISTS "hotspot_message_reads" ();
ALTER TABLE "hotspot_message_reads" ADD COLUMN IF NOT EXISTS "id" UUID PRIMARY KEY DEFAULT gen_random_uuid();
ALTER TABLE "hotspot_message_reads" ADD COLUMN IF NOT EXISTS "message_id" UUID NOT NULL;
ALTER TABLE "hotspot_message_reads" ADD COLUMN IF NOT EXISTS "mac" TEXT NOT NULL;
ALTER TABLE "hotspot_message_reads" ADD COLUMN IF NOT EXISTS "read_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;
CREATE UNIQUE INDEX IF NOT EXISTS "hotspot_message_reads_uq" ON "hotspot_message_reads" ("message_id", "mac");
CREATE INDEX IF NOT EXISTS "hotspot_message_reads_mac_idx" ON "hotspot_message_reads" ("mac");
