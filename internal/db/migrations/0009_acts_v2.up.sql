-- =========================================================
-- 0009_acts_v2 — акты: новая структура
-- Удаляем старые таблицы и создаём новые.
-- =========================================================

DROP TABLE IF EXISTS act_stages CASCADE;
DROP TABLE IF EXISTS act_attachments CASCADE;
DROP TABLE IF EXISTS acts CASCADE;

-- Реестр главных (денежных) актов
CREATE TABLE acts (
    guid        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kind        TEXT NOT NULL,
    kind_custom TEXT NOT NULL DEFAULT '',
    name        TEXT NOT NULL,
    amount      NUMERIC(15,2) NOT NULL DEFAULT 0,  -- без НДС
    sign_date   DATE,
    file_path   TEXT NOT NULL DEFAULT '',
    note        TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_acts_sign_date ON acts(sign_date);
CREATE INDEX idx_acts_file_path ON acts(file_path);

-- Связь M:N: акт ↔ этап с распределением суммы
CREATE TABLE act_stages (
    act_guid   UUID NOT NULL REFERENCES acts(guid)   ON DELETE CASCADE,
    stage_guid UUID NOT NULL REFERENCES stages(guid) ON DELETE CASCADE,
    amount     NUMERIC(15,2) NOT NULL DEFAULT 0,     -- без НДС
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (act_guid, stage_guid)
);

CREATE INDEX idx_act_stages_stage ON act_stages(stage_guid);

-- Приложения (информационные акты) — привязаны к главному акту
CREATE TABLE act_attachments (
    guid        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    act_guid    UUID NOT NULL REFERENCES acts(guid) ON DELETE CASCADE,
    kind        TEXT NOT NULL,
    kind_custom TEXT NOT NULL DEFAULT '',
    name        TEXT NOT NULL,
    file_path   TEXT NOT NULL DEFAULT '',
    note        TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_act_attachments_act ON act_attachments(act_guid);