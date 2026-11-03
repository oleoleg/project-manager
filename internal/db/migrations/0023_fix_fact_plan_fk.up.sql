-- 1. Удаляем старый внешний ключ, который удаляет факт
ALTER TABLE budget_fact_lines 
DROP CONSTRAINT IF EXISTS budget_fact_lines_plan_line_id_fkey;

-- 2. Создаем новый внешний ключ, который при удалении плана просто делает plan_line_id = NULL
ALTER TABLE budget_fact_lines 
ADD CONSTRAINT budget_fact_lines_plan_line_id_fkey 
FOREIGN KEY (plan_line_id) REFERENCES budget_plan_lines(id) ON DELETE SET NULL;

-- 3. Также добавим связь с договором для факта (на всякий случай), если её нет
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_budget_fact_lines_contract'
    ) THEN
        ALTER TABLE budget_fact_lines 
        ADD CONSTRAINT fk_budget_fact_lines_contract 
        FOREIGN KEY (contract_id) REFERENCES contracts(id) ON DELETE CASCADE;
    END IF;
END $$;