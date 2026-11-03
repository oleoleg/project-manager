DROP INDEX IF EXISTS idx_budget_plan_lines_contract_id;

ALTER TABLE budget_plan_lines 
DROP CONSTRAINT IF EXISTS fk_budget_plan_lines_contract;

ALTER TABLE budget_plan_lines 
DROP COLUMN IF EXISTS contract_id;