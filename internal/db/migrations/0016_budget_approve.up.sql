ALTER TABLE budget_plan_snapshots 
ADD COLUMN approved_by BIGINT REFERENCES users(id),
ADD COLUMN approved_at TIMESTAMPTZ;