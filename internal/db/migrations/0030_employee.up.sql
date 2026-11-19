CREATE TABLE employee_departments (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE employees (
    id           BIGSERIAL PRIMARY KEY,
    source_id    BIGINT NOT NULL UNIQUE,          -- Users.id на источнике
    username     TEXT NOT NULL,
    longname     TEXT,
    city         TEXT,
    email        TEXT,
    is_admin     BOOLEAN NOT NULL DEFAULT FALSE,
    is_blocked   BOOLEAN NOT NULL DEFAULT FALSE,
    department_id BIGINT REFERENCES employee_departments(id) ON DELETE SET NULL,
    synced_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_employees_username ON employees (username);

CREATE TABLE employee_work_types (
    source_id        INTEGER PRIMARY KEY,          -- WorkTypes.id на источнике
    name             TEXT NOT NULL,
    calculate_hours  BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE employee_projects (
    prj_id       BIGINT PRIMARY KEY,               -- Projects.prj_id (0 = «Не учтено»)
    description  TEXT,
    synced_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE employee_reports (
    source_id    BIGINT PRIMARY KEY,               -- Reports.id на источнике
    employee_id  BIGINT NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    prj_id       BIGINT NOT NULL,                  -- 0 = «Не учтено»
    work_type_id INTEGER NOT NULL REFERENCES employee_work_types(source_id),
    start_time   TIMESTAMP NOT NULL,               -- локальное время источника
    end_time     TIMESTAMP NOT NULL,
    description  TEXT,
    source_ts    TIMESTAMP,                        -- Reports.timestamp
    synced_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_employee_reports_prj      ON employee_reports (prj_id);
CREATE INDEX idx_employee_reports_employee ON employee_reports (employee_id);
CREATE INDEX idx_employee_reports_start    ON employee_reports (start_time);
