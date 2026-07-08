-- Migration idempotente que amplia o CHECK de entry_type em
-- hotspot_device_credit_history para aceitar 'voucher_redemption'
-- (resgate de voucher pelo proprio dispositivo, ver
-- services/backend/hotspot_vouchers.go). Nao existe "ADD CONSTRAINT
-- IF NOT EXISTS" nem forma de alterar um CHECK existente - localiza o
-- nome da constraint dinamicamente (nao confia em nome fixo) e
-- recria, protegido por IF para nao falhar em reaplicacoes.

DO $$
DECLARE
    existing_constraint TEXT;
BEGIN
    SELECT con.conname INTO existing_constraint
    FROM pg_constraint con
    JOIN pg_class rel ON rel.oid = con.conrelid
    WHERE rel.relname = 'hotspot_device_credit_history'
      AND con.contype = 'c'
      AND pg_get_constraintdef(con.oid) LIKE '%entry_type%';

    IF existing_constraint IS NOT NULL THEN
        EXECUTE format('ALTER TABLE "hotspot_device_credit_history" DROP CONSTRAINT %I', existing_constraint);
    END IF;

    EXECUTE '
        ALTER TABLE "hotspot_device_credit_history"
            ADD CONSTRAINT "hotspot_device_credit_history_entry_type_check"
            CHECK ("entry_type" IN (''manual_recharge'',''auto_recharge'',''debit'',''voucher_redemption''))
    ';
END $$;
