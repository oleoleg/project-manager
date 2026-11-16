ALTER TABLE contracts DROP CONSTRAINT IF EXISTS contracts_status_code_fkey;
DROP TABLE IF EXISTS contract_statuses CASCADE;