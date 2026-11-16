ALTER TABLE stages DROP CONSTRAINT IF EXISTS stages_status_code_fkey;
DROP TABLE IF EXISTS stage_statuses CASCADE;