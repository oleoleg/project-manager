-- 1. Папка приложений для конкретной версии документа
ALTER TABLE document_versions
ADD COLUMN attachments_rel_path TEXT NOT NULL DEFAULT '';

-- 2. Запрет на создание второго Основного договора в одном проекте
CREATE UNIQUE INDEX idx_one_main_doc_per_contract
ON contract_documents (contract_id)
WHERE doc_type = 'main';