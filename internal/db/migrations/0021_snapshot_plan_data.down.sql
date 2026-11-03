ALTER TABLE budget_plan_lines ADD COLUMN snapshot_id UUID;
ALTER TABLE budget_plan_snapshots DROP COLUMN plan_data_snapshot;