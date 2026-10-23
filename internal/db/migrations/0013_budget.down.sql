-- Удаляем индексы
DROP INDEX IF EXISTS idx_budget_fact_lines_contract;
DROP INDEX IF EXISTS idx_budget_plan_lines_snapshot;
DROP INDEX IF EXISTS idx_budget_plan_snapshots_contract;

-- Удаляем таблицы в порядке, обратном созданию
DROP TABLE IF EXISTS budget_fact_lines;
DROP TABLE IF EXISTS budget_plan_lines;
DROP TABLE IF EXISTS budget_plan_snapshots;
DROP TABLE IF EXISTS budget_settings;