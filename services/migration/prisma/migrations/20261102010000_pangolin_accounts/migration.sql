-- Cada conta/organização mantém os seus segredos e identificadores remotos.
CREATE TABLE IF NOT EXISTS pangolin_accounts ();
ALTER TABLE pangolin_accounts ADD COLUMN IF NOT EXISTS id TEXT PRIMARY KEY;
ALTER TABLE pangolin_accounts ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT '';
ALTER TABLE pangolin_accounts ADD COLUMN IF NOT EXISTS api_url TEXT NOT NULL DEFAULT '';
ALTER TABLE pangolin_accounts ADD COLUMN IF NOT EXISTS endpoint TEXT NOT NULL DEFAULT '';
ALTER TABLE pangolin_accounts ADD COLUMN IF NOT EXISTS org_id TEXT NOT NULL DEFAULT '';
ALTER TABLE pangolin_accounts ADD COLUMN IF NOT EXISTS api_key TEXT NOT NULL DEFAULT '';
ALTER TABLE pangolin_accounts ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;
CREATE TABLE IF NOT EXISTS pangolin_account_sites ();
ALTER TABLE pangolin_account_sites ADD COLUMN IF NOT EXISTS id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text;
ALTER TABLE pangolin_account_sites ADD COLUMN IF NOT EXISTS account_id TEXT NOT NULL REFERENCES pangolin_accounts(id);
ALTER TABLE pangolin_account_sites ADD COLUMN IF NOT EXISTS site_id INTEGER NOT NULL;
ALTER TABLE pangolin_account_sites ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT '';
ALTER TABLE pangolin_account_sites ADD COLUMN IF NOT EXISTS newt_id TEXT NOT NULL DEFAULT '';
ALTER TABLE pangolin_account_sites ADD COLUMN IF NOT EXISTS secret TEXT NOT NULL DEFAULT '';
CREATE UNIQUE INDEX IF NOT EXISTS pangolin_account_sites_remote_idx ON pangolin_account_sites(account_id,site_id);
CREATE TABLE IF NOT EXISTS pangolin_account_resources ();
ALTER TABLE pangolin_account_resources ADD COLUMN IF NOT EXISTS id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text;
ALTER TABLE pangolin_account_resources ADD COLUMN IF NOT EXISTS account_id TEXT NOT NULL REFERENCES pangolin_accounts(id);
ALTER TABLE pangolin_account_resources ADD COLUMN IF NOT EXISTS resource_id INTEGER NOT NULL;
ALTER TABLE pangolin_account_resources ADD COLUMN IF NOT EXISTS site_id INTEGER NOT NULL;
ALTER TABLE pangolin_account_resources ADD COLUMN IF NOT EXISTS local_domain TEXT NOT NULL DEFAULT '';
CREATE UNIQUE INDEX IF NOT EXISTS pangolin_account_resources_remote_idx ON pangolin_account_resources(account_id,resource_id);
CREATE INDEX IF NOT EXISTS pangolin_account_resources_site_idx ON pangolin_account_resources(account_id,site_id);

-- Preserva a integração antiga e o nome dos seus containers Newt existentes.
INSERT INTO pangolin_accounts(id,name,api_url,endpoint,org_id,api_key)
SELECT 'legacy','Conta principal',api_url,endpoint,org_id,api_key FROM pangolin_config WHERE id=1
ON CONFLICT(id) DO NOTHING;
INSERT INTO pangolin_account_sites(account_id,site_id,name,newt_id,secret)
SELECT 'legacy',site_id,name,newt_id,secret FROM pangolin_sites
WHERE EXISTS(SELECT 1 FROM pangolin_accounts WHERE id='legacy')
ON CONFLICT(account_id,site_id) DO NOTHING;
INSERT INTO pangolin_account_resources(account_id,resource_id,site_id,local_domain)
SELECT 'legacy',resource_id,site_id,local_domain FROM pangolin_resources
WHERE EXISTS(SELECT 1 FROM pangolin_accounts WHERE id='legacy')
ON CONFLICT(account_id,resource_id) DO NOTHING;
