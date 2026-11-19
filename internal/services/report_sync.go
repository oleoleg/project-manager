package services

import (
	"context"
	"fmt"

	"github.com/oleoleg/project-manager/internal/clients"
	"github.com/oleoleg/project-manager/internal/db"
	"github.com/oleoleg/project-manager/internal/models"
)

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
	client *clients.ReportsClient
	repo   *db.EmployeeRepo
}

func NewReportSyncService(client *clients.ReportsClient, repo *db.EmployeeRepo) *ReportSyncService {
	return &ReportSyncService{
		client: client,
		repo:   repo,
	}
}

func (s *ReportSyncService) Sync(ctx context.Context, p clients.ReportsParams) (*SyncResult, error) {
	res := &SyncResult{}

	// 1. Типы работ
	wts, err := s.client.FetchWorkTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch worktypes: %w", err)
	}
	if err := s.repo.UpsertWorkTypes(ctx, wts); err != nil {
		return nil, fmt.Errorf("upsert worktypes: %w", err)
	}
	res.WorkTypes = len(wts)

	// 2. Сотрудники и отделы
	emps, err := s.client.FetchUsers(ctx)
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
	prjs, err := s.client.FetchProjects(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch projects: %w", err)
	}
	if err := s.repo.UpsertProjects(ctx, prjs); err != nil {
		return nil, fmt.Errorf("upsert projects: %w", err)
	}
	res.Projects = len(prjs)

	// 4. Отчеты
	rawReps, err := s.client.FetchReports(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("fetch reports: %w", err)
	}

	var validReps []models.EmployeeReport
	for _, raw := range rawReps {
		localEmpID, ok := empMap[raw.UserID]
		if !ok {
			res.Skipped++
			res.Errors = append(res.Errors, fmt.Sprintf("Сотрудник с source_id=%d не найден в локальной БД", raw.UserID))
			continue
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
		})
	}

	if err := s.repo.UpsertReports(ctx, validReps); err != nil {
		return nil, fmt.Errorf("upsert reports: %w", err)
	}
	res.Reports = len(validReps)

	return res, nil
}
