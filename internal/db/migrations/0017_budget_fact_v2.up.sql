DROP TABLE IF EXISTS budget_fact_lines;

CREATE TABLE budget_fact_lines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contract_id VARCHAR(16) NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    plan_line_id UUID REFERENCES budget_plan_lines(id) ON DELETE CASCADE,
    stage_guid UUID REFERENCES stages(guid) ON DELETE SET NULL,
    work_kind_code VARCHAR(50),
    name TEXT NOT NULL,
    operation_date DATE NOT NULL,
    payment_date DATE NOT NULL,
    operation_type VARCHAR(100) NOT NULL,
    plan_income NUMERIC(15,2) NOT NULL DEFAULT 0,
    plan_expense NUMERIC(15,2) NOT NULL DEFAULT 0,
    fact_income NUMERIC(15,2) NOT NULL DEFAULT 0,
    fact_expense NUMERIC(15,2) NOT NULL DEFAULT 0,
    is_confirmed BOOLEAN NOT NULL DEFAULT FALSE,
    is_manual_edit BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);