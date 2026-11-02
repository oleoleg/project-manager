-- 1. Переименовываем колонки
ALTER TABLE stages RENAME COLUMN work_kind_code TO work_kind;
ALTER TABLE stage_costs RENAME COLUMN work_kind_code TO work_kind;
ALTER TABLE budget_plan_lines RENAME COLUMN work_kind_code TO work_kind;
ALTER TABLE budget_fact_lines RENAME COLUMN work_kind_code TO work_kind;

-- 2. Обновляем данные (английские коды -> русские)
UPDATE stages SET work_kind = 'Сборка' WHERE work_kind = 'assembly';
UPDATE stages SET work_kind = 'Комплексный' WHERE work_kind = 'complex';
UPDATE stages SET work_kind = 'ПИР' WHERE work_kind = 'PIR';
UPDATE stages SET work_kind = 'ПНР' WHERE work_kind = 'PNR';
UPDATE stages SET work_kind = 'ППО' WHERE work_kind = 'PPO';
UPDATE stages SET work_kind = 'Ремонт' WHERE work_kind = 'repair';
UPDATE stages SET work_kind = 'Продажа' WHERE work_kind = 'sale';
UPDATE stages SET work_kind = 'СМР' WHERE work_kind = 'SMR';
UPDATE stages SET work_kind = 'ТО' WHERE work_kind = 'TO';

UPDATE stage_costs SET work_kind = 'Сборка' WHERE work_kind = 'assembly';
UPDATE stage_costs SET work_kind = 'Комплексный' WHERE work_kind = 'complex';
UPDATE stage_costs SET work_kind = 'ПИР' WHERE work_kind = 'PIR';
UPDATE stage_costs SET work_kind = 'ПНР' WHERE work_kind = 'PNR';
UPDATE stage_costs SET work_kind = 'ППО' WHERE work_kind = 'PPO';
UPDATE stage_costs SET work_kind = 'Ремонт' WHERE work_kind = 'repair';
UPDATE stage_costs SET work_kind = 'Продажа' WHERE work_kind = 'sale';
UPDATE stage_costs SET work_kind = 'СМР' WHERE work_kind = 'SMR';
UPDATE stage_costs SET work_kind = 'ТО' WHERE work_kind = 'TO';

UPDATE budget_plan_lines SET work_kind = 'Сборка' WHERE work_kind = 'assembly';
UPDATE budget_plan_lines SET work_kind = 'Комплексный' WHERE work_kind = 'complex';
UPDATE budget_plan_lines SET work_kind = 'ПИР' WHERE work_kind = 'PIR';
UPDATE budget_plan_lines SET work_kind = 'ПНР' WHERE work_kind = 'PNR';
UPDATE budget_plan_lines SET work_kind = 'ППО' WHERE work_kind = 'PPO';
UPDATE budget_plan_lines SET work_kind = 'Ремонт' WHERE work_kind = 'repair';
UPDATE budget_plan_lines SET work_kind = 'Продажа' WHERE work_kind = 'sale';
UPDATE budget_plan_lines SET work_kind = 'СМР' WHERE work_kind = 'SMR';
UPDATE budget_plan_lines SET work_kind = 'ТО' WHERE work_kind = 'TO';

UPDATE budget_fact_lines SET work_kind = 'Сборка' WHERE work_kind = 'assembly';
UPDATE budget_fact_lines SET work_kind = 'Комплексный' WHERE work_kind = 'complex';
UPDATE budget_fact_lines SET work_kind = 'ПИР' WHERE work_kind = 'PIR';
UPDATE budget_fact_lines SET work_kind = 'ПНР' WHERE work_kind = 'PNR';
UPDATE budget_fact_lines SET work_kind = 'ППО' WHERE work_kind = 'PPO';
UPDATE budget_fact_lines SET work_kind = 'Ремонт' WHERE work_kind = 'repair';
UPDATE budget_fact_lines SET work_kind = 'Продажа' WHERE work_kind = 'sale';
UPDATE budget_fact_lines SET work_kind = 'СМР' WHERE work_kind = 'SMR';
UPDATE budget_fact_lines SET work_kind = 'ТО' WHERE work_kind = 'TO';

-- 3. Удаляем старую таблицу
DROP TABLE work_kinds;