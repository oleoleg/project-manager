# Project Manager

Внутренний веб-инструмент для управления проектами: договоры, этапы, бюджеты,
табели сотрудников и финансы. Замена Excel/VBA-наработкам.

## Стек

- **Backend:** Go 1.22+, [chi](https://github.com/go-chi/chi) router
- **DB:** PostgreSQL 14+
- **Migrations:** [golang-migrate](https://github.com/golang-migrate/migrate) (встроены в бинарник через `embed`)
- **Auth:** bcrypt + cookie-сессии ([gorilla/sessions](https://github.com/gorilla/sessions))
- **Frontend:** HTML-шаблоны + собственный CSS (без внешних фреймворков)

## Требования

- Go 1.22 или новее
- PostgreSQL 14+
- Git

## Быстрый старт

### 1. Клонировать репозиторий

Открой терминал и выполни:

```
git clone https://github.com/oleoleg/project-manager.git
cd project-manager
```

### 2. Настроить окружение

Скопируй пример и заполни реальными значениями:

```
cp .env.example .env
```

Минимальный набор переменных:

| Переменная | Назначение |
|---|---|
| `HTTP_PORT` | Порт HTTP-сервера (по умолчанию `8080`) |
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` | Подключение к PostgreSQL |
| `SESSION_SECRET` | Секрет для cookie-сессий, **минимум 32 символа** |
| `BOOTSTRAP_ADMIN_USERNAME` / `BOOTSTRAP_ADMIN_PASSWORD` / `BOOTSTRAP_ADMIN_FULLNAME` | Первый админ, создаётся один раз, если таблица `users` пуста |

Пример генерации `SESSION_SECRET` (PowerShell):

```
-join ((1..48) | ForEach-Object { [char](Get-Random -Min 33 -Max 126) })
```

### 3. Запустить

```
go mod tidy
go run ./cmd/server
```

При первом запуске сервер:

1. Подключится к служебной БД `postgres` и создаст `DB_NAME`, если её нет.
2. Накатит миграции из `internal/db/migrations/`.
3. Создаст первого админа из `BOOTSTRAP_ADMIN_*`, если таблица `users` пуста.

Открыть: http://localhost:8080 → редирект на `/login`.

## Структура проекта

```
project-manager/
├── cmd/server/            # точка входа
├── internal/
│   ├── config/            # загрузка .env и env
│   ├── db/                # пул pgx, миграции, репозитории
│   │   └── migrations/    # SQL-миграции (embed)
│   ├── handlers/          # HTTP-обработчики
│   ├── middleware/        # auth, роли, контекст
│   ├── models/            # доменные типы
│   └── services/          # бизнес-логика, сессии
├── web/
│   ├── templates/         # html/template
│   └── static/            # css
├── .env.example
└── README.md
```

## Роли и права

| Роль | Код | Права |
|---|---|---|
| Администратор | `admin` | Создание пользователей, полный доступ |
| Начальник РП | `rp_chief` | Полный доступ |
| Директор | `director` | Графики, статистика по фирме |
| Руководитель проектов | `rp` | Свои проекты, любой персонал |
| Бухгалтер | `accountant` | Чтение всего |

## Прогресс разработки

- [x] **Шаг 0** — скелет проекта (chi, конфиг, статика, шаблоны)
- [x] **Шаг 1** — PostgreSQL, автосоздание БД, миграции, полная схема (13 таблиц + справочники)
- [x] **Шаг 2** — аутентификация (bcrypt), cookie-сессии, роли, `/login`, `/logout`, `/dashboard`
- [ ] **Шаг 3** — CRUD «Контрагент»
- [ ] **Шаг 4** — CRUD «Реестр договоров» + валидация `NNNN-AA`
- [ ] **Шаг 5** — «Карточка договора» + «Этапы»
- [ ] **Шаг 6** — «Затраты этапа» (ФОТп, коэф)
- [ ] **Шаг 7** — «Акты»
- [ ] **Шаг 8** — Табели сотрудников
- [ ] **Шаг 9** — Финплан / факт
- [ ] **Шаг 10** — Дашборд директора (графики)

## Схема БД (актуальная)

Справочники: `roles`, `contract_statuses`, `contract_types`, `procurement_card_statuses`,
`tracking_statuses`, `stage_statuses`, `work_kinds`.

Рабочие таблицы: `users`, `counterparties`, `contracts`, `contract_cards`,
`stages`, `stage_costs`, `stage_acts`.

Подробности — в `internal/db/migrations/0001_init.up.sql`.

## Разработка

Проверить сборку и статический анализ:

```
go build ./...
go vet ./...
```

Проверить состояние БД через pgAdmin или:

```sql
SELECT * FROM users;
SELECT * FROM roles;
```

## Лицензия

Внутренний проект.