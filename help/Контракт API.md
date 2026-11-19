# Контракт API: сервер «Работа сотрудников по проектам»

Базовый URL: `http://192.168.0.200:8000/reports/api/database`

Общие требования:
- Формат ответа: JSON, `Content-Type: application/json`.
- Коды: `200` — успех, `400` — неверные параметры, `500` — ошибка сервера.
- Даты/время в формате `YYYY-MM-DDTHH:MM:SS` (локальное время источника, без таймзоны), например `2024-01-16T09:00:00`.
- Пагинация: `limit` (по умолчанию `1000`, максимум `5000`), `offset` (по умолчанию `0`).
- Сортировка `Reports` — по `id ASC` (для стабильной пагинации).
- API только на чтение (GET), без аутентификации.

Устаревшие поля НЕ отдавать: `disable`, `mission_on`, `mission_place`, `mission_road`, `password`.

---

## 1) GET /reports

Сырые записи таблицы `Reports`.

Параметры (все опциональны):
- `from` — `YYYY-MM-DD`, фильтр по локальной дате `start_time`, включительно.
- `to` — `YYYY-MM-DD`, фильтр по локальной дате `start_time`, включительно.
- `prj_id` — integer. Если не задан — все проекты (включая `prj_id = 0`).
- `userid` — integer. На будущее.
- `limit` — integer, по умолчанию `1000`, максимум `5000`.
- `offset` — integer, по умолчанию `0`.

Ответ:
```
{
  "items": [
    {
      "id": 12345,
      "userid": 7,
      "prj_id": 1024,
      "work_type": 1,
      "start_time": "2024-01-16T09:00:00",
      "end_time": "2024-01-16T18:00:00",
      "description": "Разработка модуля X",
      "timestamp": "2024-01-16T18:05:12"
    }
  ],
  "total": 54321,
  "limit": 1000,
  "offset": 0
}
```

Поля:
- `id` — `Reports.id`.
- `userid` — `Reports.userid`.
- `prj_id` — `Reports.prj_id` (`0` означает «Не учтено»).
- `work_type` — `Reports.work_type`.
- `start_time`, `end_time` — `Reports.start_time`, `Reports.end_time`.
- `description` — `Reports.description`.
- `timestamp` — `Reports.timestamp`.

---

## 2) GET /worktypes

Справочник видов работ.

Ответ:
```
{
  "items": [
    { "id": 1, "name": "Командировка", "calculate_hours": true }
  ]
}
```

Поля:
- `id` — `WorkTypes.id`.
- `name` — `WorkTypes.name`.
- `calculate_hours` — `WorkTypes.calculate_hours` (boolean).

---

## 3) GET /users

Справочник сотрудников (без пароля).

Ответ:
```
{
  "items": [
    {
      "id": 7,
      "username": "ivanov",
      "longname": "Иванов И.И.",
      "city": "Москва",
      "email": "ivanov@example.com",
      "is_admin": false,
      "is_blocked": false,
      "department": "Отдел разработки"
    }
  ]
}
```

Поля:
- `id` — `Users.id`.
- `username` — `Users.username`.
- `longname` — `Users.longname`.
- `city` — `Users.city`.
- `email` — `Users.email`.
- `is_admin` — `Users.is_admin` (boolean).
- `is_blocked` — `Users.is_blocked` (boolean).
- `department` — новое поле отдела сотрудника (текст), добавляется в сторонней БД.

---

## 4) GET /projects

Справочник проектов.

Ответ:
```
{
  "items": [
    { "prj_id": 1024, "description": "Договор 1024-AA ..." }
  ]
}
```

Поля:
- `prj_id` — `Projects.prj_id`.
- `description` — `Projects.description`.