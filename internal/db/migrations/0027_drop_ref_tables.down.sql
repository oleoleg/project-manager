-- Восстанавливаем contract_types
CREATE TABLE contract_types (code TEXT PRIMARY KEY, name TEXT NOT NULL);
INSERT INTO contract_types (code, name) VALUES 
('1', 'Подряд'), ('2', 'Поставка'), ('3', 'Сервис'), ('4', 'Остальное');

-- Восстанавливаем procurement_card_statuses
CREATE TABLE procurement_card_statuses (code TEXT PRIMARY KEY, name TEXT NOT NULL);
INSERT INTO procurement_card_statuses (code, name) VALUES 
('active', 'Ведется'), ('closed', 'Закрыта'), ('none', 'Не ведется');

-- Восстанавливаем tracking_statuses
CREATE TABLE tracking_statuses (code TEXT PRIMARY KEY, name TEXT NOT NULL);
INSERT INTO tracking_statuses (code, name) VALUES 
('none', 'Не отслеживать'), ('active', 'Отслеживать');

-- contract_cards (пустая заготовка, так как структуру мы не знаем, 
-- но если откат не нужен, можно закомментировать)
-- CREATE TABLE contract_cards (...); 