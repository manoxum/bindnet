-- Taxa (download/upload) de perfil e de override de dispositivo passa a
-- aceitar valor fracionario. As colunas eram INTEGER, entao uma taxa
-- como 1.5mbit ou 17.5kbyte/s nao tinha como ser gravada: o admin so
-- conseguia expressar 1MB/s ou 2MB/s na unidade escolhida (trocar pra
-- 1500kbyte funcionava, mas e outra leitura - a UI mostra a unidade que
-- ele escolheu, ver HotspotRateFields.tsx).
--
-- double precision (nao numeric) por dois motivos: o valor vai direto
-- pro argumento de taxa do tc, que ja parseia decimal (get_rate/strtod
-- do iproute2, ver rate() em services/worker/controller/shaping_tc.go),
-- e nao ha exatidao decimal contabil em jogo aqui; e double precision
-- mapeia pra float64 nativo no pgx, sem pgtype.Numeric (ver
-- hotspotLimits em services/backend/hotspot_device_limits.go).
--
-- ALTER ... TYPE nao aceita "IF NOT EXISTS", mas continua re-executavel
-- como as demais migrations deste repo: pedir double precision numa
-- coluna que ja e double precision e aceito pelo Postgres sem erro.
-- INTEGER -> double precision e cast implicito (sem USING) e preserva
-- todos os valores ja gravados.
--
-- hotspot_global_limits e as colunas quota_throttle_* ficam INTEGER de
-- proposito: sao o shape legado, sem leitor/escritor Go nenhum (ver
-- HotspotGlobalLimits em schema.prisma).
--
-- Cota nao entra aqui: ja e gravada em BYTES (daily/weekly/
-- monthly_quota_bytes, BIGINT), entao 1.5GB sempre coube - o valor
-- fracionario so era recusado pela validacao do formulario.

ALTER TABLE "hotspot_profiles" ALTER COLUMN "download_rate_value" TYPE double precision;
ALTER TABLE "hotspot_profiles" ALTER COLUMN "upload_rate_value" TYPE double precision;

ALTER TABLE "hotspot_device_limits" ALTER COLUMN "download_rate_value" TYPE double precision;
ALTER TABLE "hotspot_device_limits" ALTER COLUMN "upload_rate_value" TYPE double precision;
