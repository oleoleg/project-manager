ALTER TABLE employee_reports
    ADD COLUMN work_kind   TEXT NOT NULL DEFAULT '',
    ADD COLUMN stage_number TEXT NOT NULL DEFAULT '';

CREATE TABLE department_work_kinds (
    department_id BIGINT PRIMARY KEY REFERENCES employee_departments(id) ON DELETE CASCADE,
    work_kind     TEXT NOT NULL
);

CREATE TABLE employee_rates (
    id           BIGSERIAL PRIMARY KEY,
    employee_id  BIGINT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    rate_per_hour NUMERIC(12,2) NOT NULL DEFAULT 0,
    valid_from   DATE NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (employee_id, valid_from)
);