ALTER TABLE budget_plan_snapshots ADD COLUMN plan_data_snapshot JSONB DEFAULT '[]'::jsonb;
ALTER TABLE budget_plan_lines DROP COLUMN snapshot_id;