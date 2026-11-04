-- Nome DNS usado no Host/SNI para selecionar o site do nginx integrado.
CREATE TABLE IF NOT EXISTS pangolin_resources ();
ALTER TABLE pangolin_resources ADD COLUMN IF NOT EXISTS local_domain TEXT NOT NULL DEFAULT '';
