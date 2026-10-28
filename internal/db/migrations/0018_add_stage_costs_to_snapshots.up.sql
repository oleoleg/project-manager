ALTER TABLE budget_plan_snapshots ADD COLUMN stage_costs_snapshot JSONB DEFAULT '[]'::jsonb;
ALTER TABLE budget_plan_snapshots ADD COLUMN stage_data_snapshot JSONB DEFAULT '[]'::jsonb;