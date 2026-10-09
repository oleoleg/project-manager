-- =========================================================
-- 0001_init — базовая схема Project Manager
-- =========================================================

-- ---------------------------------------------------------
-- Пользователи и роли
-- ---------------------------------------------------------
CREATE TABLE IF NOT EXISTS roles (
    code        TEXT PRIMARY KEY,           -- admin, director, rp_chief, rp, accountant
    name        TEXT NOT NULL,
    description TEXT
);

INSERT INTO roles (code, name, description) VALUES
    ('admin',      'Администратор',      'Создаёт пользователей, полный доступ'),
    ('rp_chief',   'Начальник РП',       'Полный доступ ко всему'),
    ('director',   'Директор',           'Строит графики, видит статистику по фирме'),
    ('rp',         'Руководитель проектов', 'Управляет своими проектами и персоналом'),
    ('accountant', 'Бухгалтер',          'Доступ на чтение ко всему')
ON CONFLICT (code) DO NOTHING;

CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    full_name     TEXT NOT NULL,
    password_hash TEXT NOT NULL,            -- bcrypt, заполним на шаге 2
    role_code     TEXT NOT NULL REFERENCES roles(code),
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------
-- Справочники (статусы, виды работ, типы договоров)
-- ---------------------------------------------------------
CREATE TABLE IF NOT EXISTS contract_statuses (
    code SMALLINT PRIMARY KEY,
    name TEXT NOT NULL
);

INSERT INTO contract_statuses (code, name) VALUES
    (0, 'Заключение договора'),
    (1, 'В работе'),
    (2, 'Закрытие договора'),
    (3, 'Закрыт'),
    (4, 'Закрыт, но есть долги'),
    (5, 'Не пошло в реализацию')
ON CONFLICT (code) DO NOTHING;

CREATE TABLE IF NOT EXISTS contract_types (
    code TEXT PRIMARY KEY,
    name TEXT NOT NULL
);

INSERT INTO contract_types (code, name) VALUES
    ('contract', 'Подряд'),
    ('service',  'Сервис'),
    ('supply',   'Поставка'),
    ('other',    'Остальное')
ON CONFLICT (code) DO NOTHING;

CREATE TABLE IF NOT EXISTS procurement_card_statuses (
    code TEXT PRIMARY KEY,
    name TEXT NOT NULL
);

INSERT INTO procurement_card_statuses (code, name) VALUES
    ('active',  'Ведется'),
    ('closed',  'Закрыта'),
    ('none',    'Не ведется')
ON CONFLICT (code) DO NOTHING;

CREATE TABLE IF NOT EXISTS tracking_statuses (
    code TEXT PRIMARY KEY,
    name TEXT NOT NULL
);

INSERT INTO tracking_statuses (code, name) VALUES
    ('track',    'Отслеживать'),
    ('no_track', 'Не отслеживать')
ON CONFLICT (code) DO NOTHING;

CREATE TABLE IF NOT EXISTS stage_statuses (
    code SMALLINT PRIMARY KEY,
    name TEXT NOT NULL
);

INSERT INTO stage_statuses (code, name) VALUES
    (0, 'Не активен'),
    (1, 'В работе'),
    (2, 'Подписание актов'),
    (3, 'Закрыт'),
    (4, 'Закрыт, но есть долги'),
    (5, 'Гарантия')
ON CONFLICT (code) DO NOTHING;

CREATE TABLE IF NOT EXISTS work_kinds (
    code TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    is_complex BOOLEAN NOT NULL DEFAULT FALSE
);

INSERT INTO work_kinds (code, name, is_complex) VALUES
    ('PIR',       'ПИР',           FALSE),
    ('PPO',       'ППО',           FALSE),
    ('SMR',       'СМР',           FALSE),
    ('PNR',       'ПНР',           FALSE),
    ('assembly',  'Сборка',        FALSE),
    ('TO',        'ТО',            FALSE),
    ('repair',    'Ремонт',        FALSE),
    ('sale',      'Продажа',       FALSE),
    ('complex',   'Комплексный',   TRUE)
ON CONFLICT (code) DO NOTHING;

-- ---------------------------------------------------------
-- Контрагенты
-- ---------------------------------------------------------
CREATE TABLE IF NOT EXISTS counterparties (
    id           TEXT PRIMARY KEY,          -- формат NNNN
    name         TEXT NOT NULL,
    partner      TEXT,
    short_name   TEXT,
    address_actual   TEXT,
    address_legal    TEXT,
    url_workplace    TEXT,
    url_servicework  TEXT,
    url_salesprep    TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT counterparties_id_format CHECK (id ~ '^[0-9]{4}$')
);

-- ---------------------------------------------------------
-- Реестр договоров
-- ---------------------------------------------------------
CREATE TABLE IF NOT EXISTS contracts (
    id                  TEXT PRIMARY KEY,   -- формат NNNN-AA
    contract_number     TEXT NOT NULL,
    contract_date       DATE,
    work_name           TEXT,
    responsible_person  TEXT,
    status_code         SMALLINT NOT NULL REFERENCES contract_statuses(code),
    counterparty_id     TEXT NOT NULL REFERENCES counterparties(id),
    workshop            TEXT,               -- Цех
    project_url         TEXT,               -- Гиперссылка на проект
    procurement_card_status TEXT REFERENCES procurement_card_statuses(code),
    extra_details       TEXT,
    finplan_status      TEXT REFERENCES tracking_statuses(code),
    plan_schedule_status TEXT REFERENCES tracking_statuses(code),
    short_name          TEXT,
    contract_type       TEXT NOT NULL REFERENCES contract_types(code),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT contracts_id_format CHECK (id ~ '^[0-9]{4}-[A-Za-z0-9]{1,3}$')
);

CREATE INDEX IF NOT EXISTS idx_contracts_counterparty ON contracts(counterparty_id);
CREATE INDEX IF NOT EXISTS idx_contracts_status       ON contracts(status_code);

-- ---------------------------------------------------------
-- Карточка договора
-- ---------------------------------------------------------
CREATE TABLE IF NOT EXISTS contract_cards (
    contract_id         TEXT PRIMARY KEY REFERENCES contracts(id) ON DELETE CASCADE,
    overhead_percent    NUMERIC(5,2) NOT NULL DEFAULT 0,   -- % накладные расходы
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT contract_cards_overhead_range CHECK (overhead_percent BETWEEN 0 AND 100)
);

-- ---------------------------------------------------------
-- Коллекция этапов
-- ---------------------------------------------------------
CREATE TABLE IF NOT EXISTS stages (
    guid                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contract_id         TEXT NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    row_order           INTEGER NOT NULL DEFAULT 0,        -- последовательность отображения
    stage_number        TEXT,                              -- № этапа (строка, может быть "1.1")
    name                TEXT NOT NULL,
    cost                NUMERIC(15,2) NOT NULL DEFAULT 0,  -- стоимость этапа, руб без НДС
    closed_by_acts      NUMERIC(15,2) NOT NULL DEFAULT 0,  -- закрыто по актам
    vat_percent         NUMERIC(5,2)  NOT NULL DEFAULT 0,
    advance_percent     NUMERIC(5,2)  NOT NULL DEFAULT 0,
    start_conditions    TEXT,
    end_conditions      TEXT,
    close_date_plan     DATE,
    responsible_person  TEXT,
    status_code         SMALLINT NOT NULL REFERENCES stage_statuses(code),
    work_kind_code      TEXT NOT NULL REFERENCES work_kinds(code),
    note                TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT stages_vat_range     CHECK (vat_percent     BETWEEN 0 AND 100),
    CONSTRAINT stages_advance_range CHECK (advance_percent BETWEEN 0 AND 100)
);

CREATE INDEX IF NOT EXISTS idx_stages_contract ON stages(contract_id);
CREATE INDEX IF NOT EXISTS idx_stages_order    ON stages(contract_id, row_order);

-- ---------------------------------------------------------
-- Затраты этапа
-- ---------------------------------------------------------
CREATE TABLE IF NOT EXISTS stage_costs (
    guid                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    stage_guid          UUID NOT NULL REFERENCES stages(guid) ON DELETE CASCADE,
    work_kind_code      TEXT NOT NULL REFERENCES work_kinds(code),   -- без 'complex'
    cost                NUMERIC(15,2) NOT NULL DEFAULT 0,   -- стоимость работ
    fot_plan            NUMERIC(15,2) NOT NULL DEFAULT 0,   -- ФОТп
    fot_ratio           NUMERIC(6,4)  NOT NULL DEFAULT 0,   -- коэф = ФОТп / Стоимость работ
    labor_days          NUMERIC(10,2) NOT NULL DEFAULT 0,   -- т/з в днях
    rate_per_day        NUMERIC(15,2) NOT NULL DEFAULT 0,   -- ставка чел/день
    travel_expenses     NUMERIC(15,2) NOT NULL DEFAULT 0,   -- командировочные
    equipment_sale      NUMERIC(15,2) NOT NULL DEFAULT 0,   -- оборудование (продажа)
    equipment_cost      NUMERIC(15,2) NOT NULL DEFAULT 0,   -- оборудование (себестоимость)
    subcontract         NUMERIC(15,2) NOT NULL DEFAULT 0,   -- субподряд
    commission          NUMERIC(15,2) NOT NULL DEFAULT 0,   -- комиссия
    other_expenses      NUMERIC(15,2) NOT NULL DEFAULT 0,   -- другое
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT stage_costs_work_kind_not_complex CHECK (work_kind_code <> 'complex'),
    CONSTRAINT stage_costs_ratio_range           CHECK (fot_ratio BETWEEN 0 AND 1)
);

CREATE INDEX IF NOT EXISTS idx_stage_costs_stage ON stage_costs(stage_guid);

-- ---------------------------------------------------------
-- Коллекция актов
-- ---------------------------------------------------------
CREATE TABLE IF NOT EXISTS stage_acts (
    guid          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    stage_guid    UUID NOT NULL REFERENCES stages(guid) ON DELETE CASCADE,
    name          TEXT NOT NULL,        -- номер акта и дата
    url           TEXT,                 -- ссылка на акт
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_stage_acts_stage ON stage_acts(stage_guid);