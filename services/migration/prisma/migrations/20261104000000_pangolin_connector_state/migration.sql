-- Conectores existentes e novos iniciam automaticamente; Parar preserva a escolha.
ALTER TABLE pangolin_account_sites ADD COLUMN IF NOT EXISTS desired_running BOOLEAN NOT NULL DEFAULT true;
