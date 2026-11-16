CREATE TABLE stage_statuses (
    code SMALLINT PRIMARY KEY, 
    name TEXT NOT NULL
);
INSERT INTO stage_statuses (code, name) VALUES 
(0, 'Не активен'), 
(1, 'В работе'), 
(2, 'Подписание актов'), 
(3, 'Закрыт'), 
(4, 'Закрыт, но есть долги'), 
(5, 'Гарантия');