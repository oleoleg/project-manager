-- 1. В таблице stages делаем номер обязательным и уникальным в рамках договора
UPDATE stages SET stage_number = 'Без номера' WHERE stage_number IS NULL OR stage_number = '';
ALTER TABLE stages ALTER COLUMN stage_number SET NOT NULL;
ALTER TABLE stages ADD CONSTRAINT unique_contract_stage_number UNIQUE (contract_id, stage_number);

-- 2. В таблице plan_lines добавляем stage_number и stage_name
ALTER TABLE budget_plan_lines ADD COLUMN stage_number VARCHAR(50);
ALTER TABLE budget_plan_lines ADD COLUMN stage_name VARCHAR(255);

UPDATE budget_plan_lines p
SET stage_number = s.stage_number,
    stage_name = s.name
FROM stages s
WHERE p.stage_guid = s.guid;

-- 3. В таблице fact_lines добавляем stage_number и stage_name
ALTER TABLE budget_fact_lines ADD COLUMN stage_number VARCHAR(50);
ALTER TABLE budget_fact_lines ADD COLUMN stage_name VARCHAR(255);

UPDATE budget_fact_lines f
SET stage_number = s.stage_number,
    stage_name = s.name
FROM stages s
WHERE f.stage_guid = s.guid;

-- 4. Удаляем stage_guid из факта и из плана
ALTER TABLE budget_fact_lines DROP COLUMN IF EXISTS stage_guid;
ALTER TABLE budget_plan_lines DROP COLUMN IF EXISTS stage_guid;

-- 5. Создаем индексы для быстрого поиска
CREATE INDEX idx_budget_plan_lines_stage_num ON budget_plan_lines(contract_id, stage_number);
CREATE INDEX idx_budget_fact_lines_stage_num ON budget_fact_lines(contract_id, stage_number);