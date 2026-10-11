DROP TABLE IF EXISTS contract_assignees;
DROP INDEX IF EXISTS idx_contracts_responsible;
ALTER TABLE contracts DROP COLUMN IF EXISTS responsible_user_id;