CREATE TABLE contract_statuses (
    code SMALLINT PRIMARY KEY, 
    name TEXT NOT NULL
);
INSERT INTO contract_statuses (code, name) VALUES 
(0, 'Заключение договора'), 
(1, 'В работе'), 
(2, 'Закрытие договора'), 
(3, 'Закрыт'), 
(4, 'Закрыт, но есть долги'), 
(5, 'Не пошло в реализацию');