CREATE TABLE IF NOT EXISTS stage_acts (
    guid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    stage_guid UUID NOT NULL REFERENCES stages(guid) ON DELETE CASCADE,
    name TEXT NOT NULL,
    url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_stage_acts_stage ON stage_acts(stage_guid);