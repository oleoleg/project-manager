-- 1. Настройки бюджета для договора
CREATE TABLE IF NOT EXISTS budget_settings (
    contract_id VARCHAR(16) PRIMARY KEY REFERENCES contracts(id) ON DELETE CASCADE,
    project_start_date DATE NOT NULL,
    project_end_date DATE,
    payment_delay_days INT NOT NULL DEFAULT 20,
    equip_payment_percent NUMERIC(5,2) DEFAULT 0,
    equip_payment_delay INT DEFAULT 2,
    overhead_rate NUMERIC(10,4) NOT NULL DEFAULT 1.2, -- коэффициент накладных
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Снимки (версии) сгенерированного ФинПлана
CREATE TABLE IF NOT EXISTS budget_plan_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contract_id VARCHAR(16) NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT REFERENCES users(id)
);

-- 3. Строки сгенерированного ФинПлана (привязаны к снимку)
CREATE TABLE IF NOT EXISTS budget_plan_lines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    snapshot_id UUID NOT NULL REFERENCES budget_plan_snapshots(id) ON DELETE CASCADE,
    stage_guid UUID REFERENCES stages(guid) ON DELETE SET NULL,
    work_kind_code VARCHAR(50),
    name TEXT NOT NULL,
    operation_date DATE NOT NULL,
    delay_days INT NOT NULL DEFAULT 0,
    payment_date DATE NOT NULL,
    income NUMERIC(15,2) NOT NULL DEFAULT 0,
    expense NUMERIC(15,2) NOT NULL DEFAULT 0,
    operation_type VARCHAR(100) NOT NULL,
    note TEXT
);

-- 4. Фактические операции (вводятся службами)
CREATE TABLE IF NOT EXISTS budget_fact_lines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contract_id VARCHAR(16) NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    stage_guid UUID REFERENCES stages(guid) ON DELETE SET NULL,
    work_kind_code VARCHAR(50) NOT NULL, -- Вид работы (ПИР, ПНР и т.д.)
    name TEXT NOT NULL, -- Наименование операции
    operation_date DATE NOT NULL,
    delay_days INT NOT NULL DEFAULT 0, -- Отсрочка
    payment_date DATE NOT NULL, -- Вычисляется как operation_date + delay_days
    income NUMERIC(15,2) NOT NULL DEFAULT 0,
    expense NUMERIC(15,2) NOT NULL DEFAULT 0,
    operation_type VARCHAR(100) NOT NULL,
    note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Индексы для быстрого поиска
CREATE INDEX IF NOT EXISTS idx_budget_plan_snapshots_contract ON budget_plan_snapshots(contract_id);
CREATE INDEX IF NOT EXISTS idx_budget_plan_lines_snapshot ON budget_plan_lines(snapshot_id);
CREATE INDEX IF NOT EXISTS idx_budget_fact_lines_contract ON budget_fact_lines(contract_id);