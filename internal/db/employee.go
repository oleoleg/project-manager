package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oleoleg/project-manager/internal/models"
)

const reportBatchSize = 500

type EmployeeRepo struct {
	pool *pgxpool.Pool
}

func NewEmployeeRepo(pool *pgxpool.Pool) *EmployeeRepo {
	return &EmployeeRepo{pool: pool}
}

// ReportFilter — параметры выборки табеля.
type ReportFilter struct {
	PrjID      *int64
	EmployeeID *int64
	From       *time.Time
	To         *time.Time
	Limit      int
	Offset     int
}

// buildInsert собирает строку вида ($1,$2),(...) для cols-колоночного INSERT.
func buildInsert(rows, cols int) string {
	placeholders := make([]string, 0, rows)
	n := 1
	for i := 0; i < rows; i++ {
		ph := make([]string, cols)
		for j := 0; j < cols; j++ {
			ph[j] = fmt.Sprintf("$%d", n)
			n++
		}
		placeholders = append(placeholders, "("+strings.Join(ph, ",")+")")
	}
	return strings.Join(placeholders, ",")
}

// UpsertDepartments вставляет/обновляет отделы и возвращает map[name]id.
func (r *EmployeeRepo) UpsertDepartments(ctx context.Context, deps []models.EmployeeDepartment) (map[string]int64, error) {
	res := make(map[string]int64, len(deps))
	if len(deps) == 0 {
		return res, nil
	}

	ph := buildInsert(len(deps), 1)
	args := make([]any, 0, len(deps))
	for _, d := range deps {
		args = append(args, d.Name)
	}

	rows, err := r.pool.Query(ctx, `
		INSERT INTO employee_departments (name)
		VALUES `+ph+`
		ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
		RETURNING id, name
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("upsert departments: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("scan department: %w", err)
		}
		res[name] = id
	}
	return res, rows.Err()
}

// UpsertEmployees вставляет/обновляет сотрудников и возвращает map[source_id]local_id.
func (r *EmployeeRepo) UpsertEmployees(ctx context.Context, emps []models.Employee, deptIDs map[string]int64) (map[int64]int64, error) {
	res := make(map[int64]int64, len(emps))
	if len(emps) == 0 {
		return res, nil
	}

	const cols = 9
	ph := buildInsert(len(emps), cols)
	args := make([]any, 0, len(emps)*cols)
	for _, e := range emps {
		var deptID *int64
		if e.DepartmentName != "" {
			if id, ok := deptIDs[e.DepartmentName]; ok {
				deptID = &id
			}
		}
		args = append(args, e.SourceID, e.Username, e.Longname, e.City, e.Email,
			e.IsAdmin, e.IsBlocked, deptID, time.Now())
	}

	rows, err := r.pool.Query(ctx, `
		INSERT INTO employees
			(source_id, username, longname, city, email, is_admin, is_blocked, department_id, synced_at)
		VALUES `+ph+`
		ON CONFLICT (source_id) DO UPDATE SET
			username = EXCLUDED.username,
			longname = EXCLUDED.longname,
			city = EXCLUDED.city,
			email = EXCLUDED.email,
			is_admin = EXCLUDED.is_admin,
			is_blocked = EXCLUDED.is_blocked,
			department_id = EXCLUDED.department_id,
			synced_at = now()
		RETURNING id, source_id
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("upsert employees: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, sourceID int64
		if err := rows.Scan(&id, &sourceID); err != nil {
			return nil, fmt.Errorf("scan employee: %w", err)
		}
		res[sourceID] = id
	}
	return res, rows.Err()
}

// UpsertWorkTypes вставляет/обновляет справочник видов работ.
func (r *EmployeeRepo) UpsertWorkTypes(ctx context.Context, wts []models.EmployeeWorkType) error {
	if len(wts) == 0 {
		return nil
	}

	ph := buildInsert(len(wts), 3)
	args := make([]any, 0, len(wts)*3)
	for _, w := range wts {
		args = append(args, w.SourceID, w.Name, w.CalculateHours)
	}

	_, err := r.pool.Exec(ctx, `
		INSERT INTO employee_work_types (source_id, name, calculate_hours)
		VALUES `+ph+`
		ON CONFLICT (source_id) DO UPDATE SET
			name = EXCLUDED.name,
			calculate_hours = EXCLUDED.calculate_hours
	`, args...)
	if err != nil {
		return fmt.Errorf("upsert work types: %w", err)
	}
	return nil
}

// UpsertProjects вставляет/обновляет справочник проектов.
func (r *EmployeeRepo) UpsertProjects(ctx context.Context, prjs []models.EmployeeProject) error {
	if len(prjs) == 0 {
		return nil
	}

	ph := buildInsert(len(prjs), 3)
	args := make([]any, 0, len(prjs)*3)
	for _, p := range prjs {
		args = append(args, p.PrjID, p.Description, time.Now())
	}

	_, err := r.pool.Exec(ctx, `
		INSERT INTO employee_projects (prj_id, description, synced_at)
		VALUES `+ph+`
		ON CONFLICT (prj_id) DO UPDATE SET
			description = EXCLUDED.description,
			synced_at = now()
	`, args...)
	if err != nil {
		return fmt.Errorf("upsert projects: %w", err)
	}
	return nil
}

// UpsertReports вставляет/обновляет сырые отчёты. Одна транзакция на вызов, батчи по 500.
func (r *EmployeeRepo) UpsertReports(ctx context.Context, reps []models.EmployeeReport) error {
	if len(reps) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const cols = 8
	for start := 0; start < len(reps); start += reportBatchSize {
		end := start + reportBatchSize
		if end > len(reps) {
			end = len(reps)
		}
		batch := reps[start:end]

		ph := buildInsert(len(batch), cols)
		args := make([]any, 0, len(batch)*cols)
		for _, rep := range batch {
			args = append(args,
				rep.SourceID, rep.EmployeeID, rep.PrjID, rep.WorkTypeID,
				rep.StartTime, rep.EndTime, rep.Description, rep.SourceTS,
			)
		}

		_, err := tx.Exec(ctx, `
			INSERT INTO employee_reports
				(source_id, employee_id, prj_id, work_type_id, start_time, end_time, description, source_ts)
			VALUES `+ph+`
			ON CONFLICT (source_id) DO UPDATE SET
				employee_id = EXCLUDED.employee_id,
				prj_id = EXCLUDED.prj_id,
				work_type_id = EXCLUDED.work_type_id,
				start_time = EXCLUDED.start_time,
				end_time = EXCLUDED.end_time,
				description = EXCLUDED.description,
				source_ts = EXCLUDED.source_ts,
				synced_at = now()
		`, args...)
		if err != nil {
			return fmt.Errorf("upsert reports batch [%d:%d]: %w", start, end, err)
		}
	}

	return tx.Commit(ctx)
}

// ListReports возвращает записи табеля с денормализованными полями.
func (r *EmployeeRepo) ListReports(ctx context.Context, f ReportFilter) ([]models.EmployeeReport, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 1000
	}
	if limit > 5000 {
		limit = 5000
	}

	query := `
		SELECT r.source_id, r.employee_id, r.prj_id, r.work_type_id,
		       r.start_time, r.end_time, r.description, r.source_ts, r.synced_at,
		       COALESCE(e.longname, e.username, ''),
		       COALESCE(wt.name, ''),
		       COALESCE(wt.calculate_hours, false),
		       COALESCE(d.name, ''),
		       COALESCE(r.work_kind, ''),
		       COALESCE(r.stage_number, '')
		FROM employee_reports r
		JOIN employees e ON e.id = r.employee_id
		JOIN employee_work_types wt ON wt.source_id = r.work_type_id
		LEFT JOIN employee_departments d ON d.id = e.department_id
		WHERE 1=1
	`
	args := []any{}
	i := 1

	if f.PrjID != nil {
		query += fmt.Sprintf(` AND r.prj_id = $%d`, i)
		args = append(args, *f.PrjID)
		i++
	}
	if f.EmployeeID != nil {
		query += fmt.Sprintf(` AND r.employee_id = $%d`, i)
		args = append(args, *f.EmployeeID)
		i++
	}
	if f.From != nil {
		query += fmt.Sprintf(` AND r.start_time >= $%d`, i)
		args = append(args, *f.From)
		i++
	}
	if f.To != nil {
		query += fmt.Sprintf(` AND r.start_time <= $%d`, i)
		args = append(args, *f.To)
		i++
	}

	query += fmt.Sprintf(` ORDER BY r.start_time DESC, e.longname ASC LIMIT $%d OFFSET $%d`, i, i+1)
	args = append(args, limit, f.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list reports: %w", err)
	}
	defer rows.Close()

	var result []models.EmployeeReport
	for rows.Next() {
		var rep models.EmployeeReport
		if err := rows.Scan(
			&rep.SourceID, &rep.EmployeeID, &rep.PrjID, &rep.WorkTypeID,
			&rep.StartTime, &rep.EndTime, &rep.Description, &rep.SourceTS, &rep.SyncedAt,
			&rep.EmployeeName, &rep.WorkTypeName, &rep.CalculateHours,
			&rep.DepartmentName, &rep.WorkKind, &rep.StageNumber,
		); err != nil {
			return nil, fmt.Errorf("scan report: %w", err)
		}
		result = append(result, rep)
	}
	return result, rows.Err()
}

// компилируем пул, чтобы pgx не ругался на неиспользуемый импорт при отсутствии QueryRow
var _ = pgx.ErrNoRows

// ListEmployees возвращает всех сотрудников для выпадающего списка.
func (r *EmployeeRepo) ListEmployees(ctx context.Context) ([]models.Employee, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, source_id, username, COALESCE(longname, ''), COALESCE(city, ''), COALESCE(email, ''), is_admin, is_blocked, department_id
		FROM employees 
		ORDER BY longname
	`)
	if err != nil {
		return nil, fmt.Errorf("list employees: %w", err)
	}
	defer rows.Close()

	var res []models.Employee
	for rows.Next() {
		var e models.Employee
		if err := rows.Scan(
			&e.ID, &e.SourceID, &e.Username, &e.Longname, &e.City, &e.Email,
			&e.IsAdmin, &e.IsBlocked, &e.DepartmentID,
		); err != nil {
			return nil, err
		}
		res = append(res, e)
	}
	return res, rows.Err()
}

// UpdateReportMapping обновляет вид работ и этап для конкретной записи.
func (r *EmployeeRepo) UpdateReportMapping(ctx context.Context, sourceID int64, workKind, stageNumber string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE employee_reports 
		SET work_kind = $2, stage_number = $3, synced_at = now()
		WHERE source_id = $1
	`, sourceID, workKind, stageNumber)
	if err != nil {
		return fmt.Errorf("update report mapping: %w", err)
	}
	return nil
}
