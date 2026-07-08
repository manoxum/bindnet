-- Migration idempotente que vincula cada dispositivo do hotspot a um
-- perfil (services/backend/hotspot_profiles_apply.go) e distingue
-- configuracao manual de credito de uma linha criada de forma
-- preguicosa pela reconciliacao.

-- Sem FK proposital (mesma convencao das demais tabelas
-- hotspot_device_*) - default aponta pro perfil "Padrao" semeado na
-- migration anterior, entao todo dispositivo (novo OU ja existente,
-- ja que o Postgres aplica o DEFAULT tambem as linhas ja existentes)
-- ja nasce vinculado a ele sem precisar de UPDATE de backfill.
ALTER TABLE "hotspot_device_info" ADD COLUMN IF NOT EXISTS "profile_id" UUID DEFAULT '00000000-0000-0000-0000-000000000001';

-- Distingue "admin configurou credito manualmente para este MAC"
-- (true) de "linha existe so porque ensureDeviceCreditRow criou de
-- forma preguicosa" (false, default) - diferente de
-- hotspot_device_limits, uma linha aqui e criada automaticamente a
-- cada ciclo de reconciliacao/leitura, entao "a linha existe" nao
-- pode ser usado como sinal de override explicito.
ALTER TABLE "hotspot_device_credit" ADD COLUMN IF NOT EXISTS "configured" BOOLEAN NOT NULL DEFAULT false;
