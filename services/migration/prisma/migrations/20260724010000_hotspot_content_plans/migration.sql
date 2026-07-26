-- Planos de bloqueio/permissao de conteudo (parental control / firewall
-- de servicos). Um plano e um conjunto nomeado de regras (dominios, IPs,
-- categorias) vinculavel a um perfil ou dispositivo. Ver
-- services/backend/internal/hotspot/content e RULE.md. O bloqueio de
-- dominio e feito no dns-provider (sinkhole); o de IP/CIDR na zona WAN do
-- firewall; categorias sao materializadas de blocklists publicas por um
-- job de sync no backend.

CREATE TABLE IF NOT EXISTS "hotspot_content_plans" ();
ALTER TABLE "hotspot_content_plans" ADD COLUMN IF NOT EXISTS "id" UUID PRIMARY KEY DEFAULT gen_random_uuid();
ALTER TABLE "hotspot_content_plans" ADD COLUMN IF NOT EXISTS "name" TEXT NOT NULL UNIQUE;
ALTER TABLE "hotspot_content_plans" ADD COLUMN IF NOT EXISTS "description" TEXT NOT NULL DEFAULT '';
-- default_action: politica do plano para o que nenhuma regra cobre.
-- 'allow' = lista negra (bloqueia so o que casar block); 'block' =
-- lista branca (bloqueia tudo menos o que casar allow).
ALTER TABLE "hotspot_content_plans" ADD COLUMN IF NOT EXISTS "default_action" TEXT NOT NULL DEFAULT 'allow' CHECK ("default_action" IN ('allow','block'));
ALTER TABLE "hotspot_content_plans" ADD COLUMN IF NOT EXISTS "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE "hotspot_content_plans" ADD COLUMN IF NOT EXISTS "updated_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;

CREATE TABLE IF NOT EXISTS "hotspot_content_rules" ();
ALTER TABLE "hotspot_content_rules" ADD COLUMN IF NOT EXISTS "id" UUID PRIMARY KEY DEFAULT gen_random_uuid();
ALTER TABLE "hotspot_content_rules" ADD COLUMN IF NOT EXISTS "plan_id" UUID NOT NULL;
-- kind: 'domain'/'category' resolvidos pelo dns-provider; 'ip' (IP/CIDR)
-- resolvido pela zona WAN do firewall.
ALTER TABLE "hotspot_content_rules" ADD COLUMN IF NOT EXISTS "kind" TEXT NOT NULL CHECK ("kind" IN ('domain','ip','category'));
ALTER TABLE "hotspot_content_rules" ADD COLUMN IF NOT EXISTS "value" TEXT NOT NULL;
ALTER TABLE "hotspot_content_rules" ADD COLUMN IF NOT EXISTS "action" TEXT NOT NULL DEFAULT 'block' CHECK ("action" IN ('allow','block'));
ALTER TABLE "hotspot_content_rules" ADD COLUMN IF NOT EXISTS "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;
CREATE INDEX IF NOT EXISTS "hotspot_content_rules_plan_idx" ON "hotspot_content_rules" ("plan_id");

-- Catalogo de categorias. source_urls (uma URL por linha) sao blocklists
-- publicas em formato hosts/domain; categorias sem source_urls sao
-- "embutidas" (dominios semeados direto aqui, ex.: lojas de apps/DoH).
CREATE TABLE IF NOT EXISTS "hotspot_content_categories" ();
ALTER TABLE "hotspot_content_categories" ADD COLUMN IF NOT EXISTS "slug" TEXT PRIMARY KEY;
ALTER TABLE "hotspot_content_categories" ADD COLUMN IF NOT EXISTS "name" TEXT NOT NULL;
ALTER TABLE "hotspot_content_categories" ADD COLUMN IF NOT EXISTS "source_urls" TEXT NOT NULL DEFAULT '';
ALTER TABLE "hotspot_content_categories" ADD COLUMN IF NOT EXISTS "enabled" BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE "hotspot_content_categories" ADD COLUMN IF NOT EXISTS "last_synced_at" TIMESTAMPTZ;
ALTER TABLE "hotspot_content_categories" ADD COLUMN IF NOT EXISTS "domain_count" BIGINT NOT NULL DEFAULT 0;
ALTER TABLE "hotspot_content_categories" ADD COLUMN IF NOT EXISTS "updated_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;

-- Dominios materializados de cada categoria (pelo sync ou semeados).
CREATE TABLE IF NOT EXISTS "hotspot_content_category_domains" ();
ALTER TABLE "hotspot_content_category_domains" ADD COLUMN IF NOT EXISTS "category_slug" TEXT NOT NULL;
ALTER TABLE "hotspot_content_category_domains" ADD COLUMN IF NOT EXISTS "domain" TEXT NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS "hotspot_content_category_domains_uq" ON "hotspot_content_category_domains" ("category_slug", "domain");
CREATE INDEX IF NOT EXISTS "hotspot_content_category_domains_slug_idx" ON "hotspot_content_category_domains" ("category_slug");

-- Vinculo do plano (precedencia device->perfil, igual aos limites).
ALTER TABLE "hotspot_profiles" ADD COLUMN IF NOT EXISTS "content_plan_id" UUID;
ALTER TABLE "hotspot_device_limits" ADD COLUMN IF NOT EXISTS "content_plan_id" UUID;

-- Mapa IP -> plano dos clientes conectados, publicado pelo backend a
-- cada ciclo de reconciliacao (o backend sabe MAC+IP ao vivo via worker
-- e device/perfil -> plano) e lido pelo dns-provider para resolver o
-- plano de cada consulta pela origem. Fica no Postgres (nao no Redis)
-- porque o backend nao fala Redis - o dns-provider ja o carrega em
-- memoria por poll, igual a tabela de rotas.
CREATE TABLE IF NOT EXISTS "hotspot_content_client_bindings" ();
ALTER TABLE "hotspot_content_client_bindings" ADD COLUMN IF NOT EXISTS "ip" TEXT PRIMARY KEY;
ALTER TABLE "hotspot_content_client_bindings" ADD COLUMN IF NOT EXISTS "plan_id" UUID NOT NULL;
ALTER TABLE "hotspot_content_client_bindings" ADD COLUMN IF NOT EXISTS "mac_address" TEXT NOT NULL DEFAULT '';
ALTER TABLE "hotspot_content_client_bindings" ADD COLUMN IF NOT EXISTS "updated_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;

-- Categorias padrao. As de source_urls apontam para blocklists publicas
-- (StevenBlack, formato hosts); app_stores e doh_bypass sao embutidas.
INSERT INTO "hotspot_content_categories" ("slug", "name", "source_urls") VALUES
  ('ads', 'Anúncios e rastreadores', 'https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts'),
  ('adult', 'Conteúdo adulto (+18)', 'https://raw.githubusercontent.com/StevenBlack/hosts/master/alternates/porn-only/hosts'),
  ('gambling', 'Apostas e jogos de azar', 'https://raw.githubusercontent.com/StevenBlack/hosts/master/alternates/gambling-only/hosts'),
  ('social', 'Redes sociais', 'https://raw.githubusercontent.com/StevenBlack/hosts/master/alternates/social-only/hosts'),
  ('app_stores', 'Lojas de aplicativos', ''),
  ('doh_bypass', 'DNS alternativo (DoH/DoT)', '')
ON CONFLICT ("slug") DO NOTHING;

-- Dominios embutidos das categorias sem source_urls.
INSERT INTO "hotspot_content_category_domains" ("category_slug", "domain") VALUES
  ('app_stores', 'play.google.com'),
  ('app_stores', 'android.clients.google.com'),
  ('app_stores', 'play-lh.googleusercontent.com'),
  ('app_stores', 'apps.apple.com'),
  ('app_stores', 'itunes.apple.com'),
  ('app_stores', 'apps.mzstatic.com'),
  ('app_stores', 'appstore.com'),
  ('app_stores', 'microsoft.com/store'),
  ('doh_bypass', 'dns.google'),
  ('doh_bypass', 'dns.google.com'),
  ('doh_bypass', 'cloudflare-dns.com'),
  ('doh_bypass', 'mozilla.cloudflare-dns.com'),
  ('doh_bypass', 'chrome.cloudflare-dns.com'),
  ('doh_bypass', 'dns.quad9.net'),
  ('doh_bypass', 'doh.opendns.com'),
  ('doh_bypass', 'dns.adguard.com'),
  ('doh_bypass', 'dns.nextdns.io'),
  ('doh_bypass', 'doh.cleanbrowsing.org'),
  ('doh_bypass', 'mask.icloud.com'),
  ('doh_bypass', 'mask-h2.icloud.com')
ON CONFLICT ("category_slug", "domain") DO NOTHING;
