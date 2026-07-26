-- Categorias adicionais de conteudo, populadas de blocklists publicas
-- (The Block List Project + StevenBlack), formato hosts - buscadas pelo
-- job de sync do backend (StartContentBlocklistSyncLoop). Idempotente:
-- ON CONFLICT DO NOTHING nao sobrescreve categorias ja existentes/ajustadas.
INSERT INTO "hotspot_content_categories" ("slug", "name", "source_urls") VALUES
  ('torrent',      'Torrent / P2P',            'https://raw.githubusercontent.com/blocklistproject/Lists/master/torrent.txt'),
  ('piracy',       'Pirataria / streaming ilegal', 'https://raw.githubusercontent.com/blocklistproject/Lists/master/piracy.txt'),
  ('cryptomining', 'Mineração de cripto',      'https://raw.githubusercontent.com/blocklistproject/Lists/master/crypto.txt'),
  ('drugs',        'Drogas',                   'https://raw.githubusercontent.com/blocklistproject/Lists/master/drugs.txt'),
  ('scam',         'Golpes / phishing',        'https://raw.githubusercontent.com/blocklistproject/Lists/master/scam.txt'),
  ('malware',      'Malware',                  'https://raw.githubusercontent.com/blocklistproject/Lists/master/malware.txt'),
  ('fakenews',     'Notícias falsas',          'https://raw.githubusercontent.com/StevenBlack/hosts/master/alternates/fakenews-only/hosts')
ON CONFLICT ("slug") DO NOTHING;
