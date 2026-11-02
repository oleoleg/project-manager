ALTER TABLE budget_fact_lines ADD COLUMN plan_income NUMERIC(15,2) DEFAULT 0;
ALTER TABLE budget_fact_lines ADD COLUMN plan_expense NUMERIC(15,2) DEFAULT 0;