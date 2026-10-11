-- 1. Главный ответственный — ссылка на users
ALTER TABLE contracts
    ADD COLUMN IF NOT EXISTS responsible_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_contracts_responsible ON contracts(responsible_user_id);

-- 2. Мигрируем старые строки: пытаемся найти пользователя по ФИО
UPDATE contracts c
SET responsible_user_id = u.id
FROM users u
WHERE c.responsible_user_id IS NULL
  AND c.responsible_person IS NOT NULL
  AND c.responsible_person = u.full_name;

-- 3. Дополнительные ответственные (M2M)
CREATE TABLE IF NOT EXISTS contract_assignees (
    contract_id TEXT NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (contract_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_contract_assignees_user ON contract_assignees(user_id);