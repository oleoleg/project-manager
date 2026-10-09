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

```bash
git clone https://github.com/oleoleg/project-manager.git
cd project-manager