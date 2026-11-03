ALTER TABLE budget_fact_lines 
DROP CONSTRAINT IF EXISTS budget_fact_lines_plan_line_id_fkey;

ALTER TABLE budget_fact_lines 
ADD CONSTRAINT budget_fact_lines_plan_line_id_fkey 
FOREIGN KEY (plan_line_id) REFERENCES budget_plan_lines(id) ON DELETE CASCADE;

ALTER TABLE budget_fact_lines 
DROP CONSTRAINT IF EXISTS fk_budget_fact_lines_contract;