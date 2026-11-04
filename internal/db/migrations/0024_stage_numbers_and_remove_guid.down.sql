DROP INDEX IF EXISTS idx_budget_fact_lines_stage_num;
DROP INDEX IF EXISTS idx_budget_plan_lines_stage_num;

ALTER TABLE budget_fact_lines ADD COLUMN stage_guid UUID;
ALTER TABLE budget_plan_lines ADD COLUMN stage_guid UUID;

UPDATE budget_fact_lines f SET stage_guid = s.guid FROM stages s WHERE f.contract_id = s.contract_id AND f.stage_number = s.stage_number;
UPDATE budget_plan_lines p SET stage_guid = s.guid FROM stages s WHERE p.contract_id = s.contract_id AND p.stage_number = s.stage_number;

ALTER TABLE budget_fact_lines DROP COLUMN IF EXISTS stage_name;
ALTER TABLE budget_fact_lines DROP COLUMN IF EXISTS stage_number;
ALTER TABLE budget_plan_lines DROP COLUMN IF EXISTS stage_name;
ALTER TABLE budget_plan_lines DROP COLUMN IF EXISTS stage_number;

ALTER TABLE stages DROP CONSTRAINT IF EXISTS unique_contract_stage_number;
ALTER TABLE stages ALTER COLUMN stage_number DROP NOT NULL;