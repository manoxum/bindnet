-- Amplia a categoria embutida app_stores com os dominios que a Play
-- Store/App Store usam para catalogo, autorizacao e entrega de APKs (nao
-- so as imagens). O casamento por sufixo do dns-provider ja cobre
-- subdominios. Remove tambem a entrada invalida com caminho
-- ("microsoft.com/store" nunca casa uma consulta DNS). Idempotente.
DELETE FROM "hotspot_content_category_domains" WHERE category_slug = 'app_stores' AND domain LIKE '%/%';

INSERT INTO "hotspot_content_category_domains" ("category_slug", "domain") VALUES
  ('app_stores', 'play.googleapis.com'),
  ('app_stores', 'dl.google.com'),
  ('app_stores', 'android.clients.google.com'),
  ('app_stores', 'iosapps.itunes.apple.com'),
  ('app_stores', 'osxapps.itunes.apple.com'),
  ('app_stores', 'updates.cdn-apple.com')
ON CONFLICT ("category_slug", "domain") DO NOTHING;

-- Atualiza domain_count/updated_at: e o que o dns-provider usa como
-- assinatura para detectar mudanca e recarregar as categorias em memoria
-- (ver LoadContentSignature). Sem isso, dominios de categoria embutida
-- adicionados fora do job de sync nao seriam recarregados.
UPDATE "hotspot_content_categories"
   SET domain_count = (SELECT count(*) FROM "hotspot_content_category_domains" WHERE category_slug = 'app_stores'),
       updated_at = CURRENT_TIMESTAMP
 WHERE slug = 'app_stores';
