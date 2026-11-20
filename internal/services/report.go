package services

import (
	"context"
	"fmt"
	"time"

	"github.com/oleoleg/project-manager/internal/clients"
	"github.com/oleoleg/project-manager/internal/db"
	"github.com/oleoleg/project-manager/internal/models"
)

// =====================================================
// 1. Логика расчета табеля (Worklog)
// =====================================================

// BuildWorklogRow рассчитывает строку табеля по сырой записи.
func BuildWorklogRow(r models.EmployeeReport) models.WorklogRow {
	raw := r.EndTime.Sub(r.StartTime).Hours()

	var hours, overwork float64
	var pause int
	if r.CalculateHours {
		if raw > 8 {
			hours = raw - 1
		} else {
			hours = raw
		}
		if raw-9 > 0 {
			overwork = raw - 9
		}
		if raw >= 9 {
			pause = 1
		}
	}

	return models.WorklogRow{
		ReportDate:     r.StartTime,
		SourceID:       r.SourceID,
		Weekday:        weekdayRU(r.StartTime.Weekday()),
		StartTime:      r.StartTime,
		EndTime:        r.EndTime,
		Hours:          hours,
		Overwork:       overwork,
		Pause:          pause,
		Place:          r.WorkTypeName,
		PrjID:          r.PrjID,
		Description:    r.Description,
		EmployeeName:   r.EmployeeName,
		DepartmentName: r.DepartmentName,
		WorkKind:       r.WorkKind,
		StageNumber:    r.StageNumber,
	}
}

func weekdayRU(d time.Weekday) string {
	switch d {
	case time.Monday:
		return "ПН"
	case time.Tuesday:
		return "ВТ"
	case time.Wednesday:
		return "СР"
	case time.Thursday:
		return "ЧТ"
	case time.Friday:
		return "ПТ"
	case time.Saturday:
		return "СБ"
	default:
		return "ВС"
	}
}

// =====================================================
// 2. Синхронизация (ReportSyncService)
// =====================================================

type SyncResult struct {
	Departments int
	Employees   int
	WorkTypes   int
	Projects    int
	Reports     int
	Skipped     int
	Errors      []string
}

type ReportSyncService struct {
	repo         *db.EmployeeRepo
	settingsRepo *db.SettingsRepo
	defaultURL   string
}

func NewReportSyncService(repo *db.EmployeeRepo, settingsRepo *db.SettingsRepo, defaultURL string) *ReportSyncService {
	return &ReportSyncService{repo: repo, settingsRepo: settingsRepo, defaultURL: defaultURL}
}

func (s *ReportSyncService) Sync(ctx context.Context, p clients.ReportsParams) (*SyncResult, error) {
	apiURL, err := s.settingsRepo.Get(ctx, "reports_api_url")
	if err != nil || apiURL == "" {
		apiURL = s.defaultURL
	}

	client := clients.NewReportsClient(apiURL)
	res := &SyncResult{}

	// 1. Типы работ
	wts, err := client.FetchWorkTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch worktypes: %w", err)
	}
	if err := s.repo.UpsertWorkTypes(ctx, wts); err != nil {
		return nil, fmt.Errorf("upsert worktypes: %w", err)
	}
	res.WorkTypes = len(wts)

	// 2. Сотрудники и отделы
	emps, err := client.FetchUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch users: %w", err)
	}

	deptSet := make(map[string]bool)
	for _, e := range emps {
		if e.DepartmentName != "" {
			deptSet[e.DepartmentName] = true
		}
	}
	var deps []models.EmployeeDepartment
	for name := range deptSet {
		deps = append(deps, models.EmployeeDepartment{Name: name})
	}
	deptMap, err := s.repo.UpsertDepartments(ctx, deps)
	if err != nil {
		return nil, fmt.Errorf("upsert departments: %w", err)
	}
	res.Departments = len(deptMap)

	empMap, err := s.repo.UpsertEmployees(ctx, emps, deptMap)
	if err != nil {
		return nil, fmt.Errorf("upsert employees: %w", err)
	}
	res.Employees = len(empMap)

	// 3. Проекты
	prjs, err := client.FetchProjects(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch projects: %w", err)
	}
	if err := s.repo.UpsertProjects(ctx, prjs); err != nil {
		return nil, fmt.Errorf("upsert projects: %w", err)
	}
	res.Projects = len(prjs)

	// 4. Отчеты
	rawReps, err := client.FetchReports(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("fetch reports: %w", err)
	}

	// Получаем маппинг отделов
	deptMappings, _ := s.repo.GetDepartmentMappings(ctx)

	var validReps []models.EmployeeReport
	for _, raw := range rawReps {
		localEmpID, ok := empMap[raw.UserID]
		if !ok {
			res.Skipped++
			continue
		}

		// Ищем отдел сотрудника, чтобы проставить work_kind
		// (в идеале нужно сохранить deptID при загрузке сотрудников,
		// но для простоты найдем его здесь через маппинг)
		workKind := ""
		for _, emp := range emps {
			if emp.SourceID == raw.UserID {
				// Находим ID отдела по его названию
				if deptID, ok := deptMap[emp.DepartmentName]; ok {
					// Берем вид работ из нашего маппинга
					if wk, ok := deptMappings[deptID]; ok {
						workKind = wk
					}
				}
				break
			}
		}

		validReps = append(validReps, models.EmployeeReport{
			SourceID:    raw.SourceID,
			EmployeeID:  localEmpID,
			PrjID:       raw.PrjID,
			WorkTypeID:  raw.WorkTypeID,
			StartTime:   raw.StartTime,
			EndTime:     raw.EndTime,
			Description: raw.Description,
			SourceTS:    raw.SourceTS,
			WorkKind:    workKind,
		})
	}

	if err := s.repo.UpsertReports(ctx, validReps); err != nil {
		return nil, fmt.Errorf("upsert reports: %w", err)
	}
	res.Reports = len(validReps)

	return res, nil
}
