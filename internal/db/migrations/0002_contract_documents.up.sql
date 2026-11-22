-- 1. Логические документы проекта (Основной договор, ДС, Приложения)
CREATE TABLE contract_documents (
    id              BIGSERIAL PRIMARY KEY,
    contract_id     TEXT NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    doc_type        VARCHAR(20) NOT NULL, -- 'main', 'addendum', 'appendix'
    number          VARCHAR(100) NOT NULL DEFAULT '',
    title           TEXT NOT NULL,
    current_stage   VARCHAR(30) NOT NULL DEFAULT 'draft', -- 'draft', 'internal_review', 'client_review', 'rework', 'signing', 'signed'
    current_form    VARCHAR(20) NOT NULL DEFAULT 'draft', -- 'draft', 'scan', 'copy', 'original', 'edo'
    holder_info     TEXT NOT NULL DEFAULT '',             -- У кого сейчас (ФИО / Отдел / Контрагент)
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_contract_documents_contract ON contract_documents(contract_id);

-- 2. Версии файлов на сервере
CREATE TABLE document_versions (
    id              BIGSERIAL PRIMARY KEY,
    document_id     BIGINT NOT NULL REFERENCES contract_documents(id) ON DELETE CASCADE,
    version_num     INT NOT NULL,
    rel_path        TEXT NOT NULL,        -- Относительный путь: 2) Договор\Основной\v1_2026-11-22_Проект.docx
    file_name       TEXT NOT NULL,
    file_ext        VARCHAR(10) NOT NULL, -- 'docx', 'pdf'
    form_type       VARCHAR(20) NOT NULL, -- 'draft', 'scan', 'copy', 'original', 'edo'
    comment         TEXT NOT NULL DEFAULT '',
    uploaded_by     BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(document_id, version_num)
);

-- 3. История согласования и отправки
CREATE TABLE document_history (
    id              BIGSERIAL PRIMARY KEY,
    document_id     BIGINT NOT NULL REFERENCES contract_documents(id) ON DELETE CASCADE,
    version_id      BIGINT REFERENCES document_versions(id) ON DELETE SET NULL,
    stage           VARCHAR(30) NOT NULL,
    form_type       VARCHAR(20) NOT NULL, -- В каком виде передано/получено ('edo', 'original', 'scan', 'copy')
    recipient_info  TEXT NOT NULL,        -- Кому отправлено / от кого получено
    note            TEXT NOT NULL DEFAULT '',
    created_by      BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_document_history_doc ON document_history(document_id, created_at DESC);