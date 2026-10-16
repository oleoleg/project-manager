-- =========================================================
-- 0008_acts — акты, связь с этапами, режим ручного ввода
-- =========================================================

-- Реестр актов
CREATE TABLE IF NOT EXISTS acts (
    guid          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kind          TEXT NOT NULL,
    kind_custom   TEXT NOT NULL DEFAULT '',
    name          TEXT NOT NULL,
    amount        NUMERIC(15,2) NOT NULL DEFAULT 0,      -- без НДС
    vat_amount    NUMERIC(15,2) NOT NULL DEFAULT 0,      -- сумма НДС
    sign_date     DATE,
    file_path     TEXT NOT NULL DEFAULT '',
    is_main       BOOLEAN NOT NULL DEFAULT TRUE,
    parent_guid   UUID REFERENCES acts(guid) ON DELETE CASCADE,
    note          TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_acts_parent   ON acts(parent_guid);
CREATE INDEX IF NOT EXISTS idx_acts_sign_date ON acts(sign_date);

-- Связь M:N: акт ↔ этап, с распределением суммы
CREATE TABLE IF NOT EXISTS act_stages (
    act_guid   UUID NOT NULL REFERENCES acts(guid)   ON DELETE CASCADE,
    stage_guid UUID NOT NULL REFERENCES stages(guid) ON DELETE CASCADE,
    amount     NUMERIC(15,2) NOT NULL DEFAULT 0,     -- без НДС
    vat_amount NUMERIC(15,2) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (act_guid, stage_guid)
);

CREATE INDEX IF NOT EXISTS idx_act_stages_stage ON act_stages(stage_guid);

-- Режим ввода «Закрыто по актам»: ручной = TRUE, автоматический из актов = FALSE
ALTER TABLE stages
    ADD COLUMN IF NOT EXISTS closed_by_acts_manual BOOLEAN NOT NULL DEFAULT TRUE;

-- Старая таблица stage_acts больше не используется, но оставляем
-- на случай, если в ней есть данные — удалим позже отдельной миграцией.