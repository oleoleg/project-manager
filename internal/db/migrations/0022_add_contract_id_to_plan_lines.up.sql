-- 1. Добавляем колонку contract_id (пока разрешаем NULL)
ALTER TABLE budget_plan_lines 
ADD COLUMN contract_id VARCHAR(255);

-- 2. Заполняем contract_id для существующих строк через таблицу stages
UPDATE budget_plan_lines p
SET contract_id = s.contract_id
FROM stages s
WHERE p.stage_guid = s.guid;

-- 3. Если остались строки, у которых stage_guid = NULL (и мы не смогли узнать договор),
-- привяжем их к первому попавшемуся договору, чтобы миграция прошла.
-- (В будущем у нас таких строк быть не должно, так как мы всегда будем передавать contract_id).
UPDATE budget_plan_lines 
SET contract_id = (SELECT id FROM contracts LIMIT 1) 
WHERE contract_id IS NULL;

-- 4. Теперь, когда все строки имеют contract_id, делаем колонку обязательной
ALTER TABLE budget_plan_lines 
ALTER COLUMN contract_id SET NOT NULL;

-- 5. Добавляем внешний ключ на таблицу contracts
ALTER TABLE budget_plan_lines 
ADD CONSTRAINT fk_budget_plan_lines_contract 
FOREIGN KEY (contract_id) REFERENCES contracts(id) ON DELETE CASCADE;

-- 6. Создаем индекс для быстрого поиска строк по договору
CREATE INDEX idx_budget_plan_lines_contract_id ON budget_plan_lines(contract_id);