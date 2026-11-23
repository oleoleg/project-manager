package services

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/oleoleg/project-manager/internal/clients"
)

const reportDateTimeLayout = "2006-01-02T15:04:05"

// =====================================================
// 1. Расчёт строки табеля (на лету, не хранится)
// =====================================================

// WorklogRow — расчётная строка табеля. Соответствует логике MainExport источника.
type WorklogRow struct {
	SourceID       int64
	UserID         int64
	ReportDate     time.Time
	Weekday        string
	StartTime      time.Time
	EndTime        time.Time
	Hours          float64
	Overwork       float64
	Pause          int
	Place          string // WorkTypeName
	PrjID          string
	Description    string
	EmployeeName   string
	DepartmentName string
	WorkKind       string
	StageNumber    string
}

// BuildWorklogRow собирает WorklogRow из сырой записи и разрешённых справочников.
func BuildWorklogRow(
	it clients.ReportItem,
	start, end time.Time,
	workTypeName string,
	calculateHours bool,
	employeeName, departmentName string,
) WorklogRow {
	raw := end.Sub(start).Hours()

	var hours, overwork float64
	var pause int
	if calculateHours {
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

	workKind, stage := "", ""
	if it.WorkKind != nil {
		workKind = *it.WorkKind
	}
	if it.PrjStage != nil {
		stage = *it.PrjStage
	}

	return WorklogRow{
		SourceID:       it.ID,
		UserID:         it.UserID,
		ReportDate:     start,
		Weekday:        weekdayRU(start.Weekday()),
		StartTime:      start,
		EndTime:        end,
		Hours:          hours,
		Overwork:       overwork,
		Pause:          pause,
		Place:          workTypeName,
		PrjID:          it.PrjID,
		Description:    it.Description,
		EmployeeName:   employeeName,
		DepartmentName: departmentName,
		WorkKind:       workKind,
		StageNumber:    stage,
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
// 2. Проксирующий сервис отчётов
// =====================================================

type ReportsService struct {
	client *clients.ReportsClient
}

func NewReportsService(client *clients.ReportsClient) *ReportsService {
	return &ReportsService{client: client}
}

// ListFilter — фильтры страницы отчётов (из UI).
type ListFilter struct {
	From         *time.Time
	To           *time.Time
	PrjID        *string
	UserID       *int64
	DepartmentID *string
	PrjStage     *string
	Limit        int
	Offset       int
}

// ReportsPage — результат для хендлера: строки табеля + общее количество.
type ReportsPage struct {
	Rows   []WorklogRow
	Total  int
	Limit  int
	Offset int
}

// List делает один GET к источнику, обогащает строки справочниками и возвращает страницу.
func (s *ReportsService) List(ctx context.Context, f ListFilter) (*ReportsPage, error) {
	resp, err := s.client.FetchReports(ctx, clients.ReportsQuery{
		From:         f.From,
		To:           f.To,
		PrjID:        f.PrjID,
		UserID:       f.UserID,
		DepartmentID: f.DepartmentID,
		PrjStage:     f.PrjStage,
		Limit:        f.Limit,
		Offset:       f.Offset,
	})
	if err != nil {
		return nil, err
	}

	users, err := s.client.FetchUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch users: %w", err)
	}
	workTypes, err := s.client.FetchWorkTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch worktypes: %w", err)
	}
	departments, err := s.client.FetchDepartments(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch departments: %w", err)
	}

	usersByID := make(map[int64]clients.UserItem, len(users))
	for _, u := range users {
		usersByID[u.ID] = u
	}
	workTypesByID := make(map[int32]clients.WorktypeItem, len(workTypes))
	for _, w := range workTypes {
		workTypesByID[w.ID] = w
	}
	deptNames := make(map[int64]string, len(departments))
	for _, d := range departments {
		deptNames[d.ID] = d.Name
	}

	rows := make([]WorklogRow, 0, len(resp.Items))
	for _, it := range resp.Items {
		start, err := parseReportTime(it.StartTime)
		if err != nil {
			log.Printf("skip report id=%d: bad start_time %q: %v", it.ID, it.StartTime, err)
			continue
		}
		end, err := parseReportTime(it.EndTime)
		if err != nil {
			log.Printf("skip report id=%d: bad end_time %q: %v", it.ID, it.EndTime, err)
			continue
		}
		if !end.After(start) {
			log.Printf("skip report id=%d: end <= start", it.ID)
			continue
		}

		var employeeName, departmentName string
		if u, ok := usersByID[it.UserID]; ok {
			employeeName = u.Longname
			if employeeName == "" {
				employeeName = u.Username
			}
			departmentName = deptNames[u.DepartmentID]
		}

		var workTypeName string
		var calcHours bool
		if w, ok := workTypesByID[it.WorkType]; ok {
			workTypeName = w.Name
			calcHours = w.CalculateHours
		}

		rows = append(rows, BuildWorklogRow(it, start, end, workTypeName, calcHours, employeeName, departmentName))
	}

	return &ReportsPage{
		Rows:   rows,
		Total:  resp.Total,
		Limit:  resp.Limit,
		Offset: resp.Offset,
	}, nil
}

// Dictionaries — справочники для выпадающих списков на странице.
type Dictionaries struct {
	Users       []clients.UserItem
	Projects    []clients.ProjectItem
	Departments []clients.DepartmentItem
	WorkTypes   []clients.WorktypeItem
}

func (s *ReportsService) Dictionaries(ctx context.Context) (*Dictionaries, error) {
	users, err := s.client.FetchUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch users: %w", err)
	}
	projects, err := s.client.FetchProjects(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch projects: %w", err)
	}
	departments, err := s.client.FetchDepartments(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch departments: %w", err)
	}
	workTypes, err := s.client.FetchWorkTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch worktypes: %w", err)
	}

	sort.Slice(users, func(i, j int) bool {
		ni, nj := users[i].Longname, users[j].Longname
		if ni == "" {
			ni = users[i].Username
		}
		if nj == "" {
			nj = users[j].Username
		}
		return ni < nj
	})
	sort.Slice(departments, func(i, j int) bool {
		return departments[i].Name < departments[j].Name
	})
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].PrjID < projects[j].PrjID
	})

	return &Dictionaries{
		Users:       users,
		Projects:    projects,
		Departments: departments,
		WorkTypes:   workTypes,
	}, nil
}

// ReportUpdate — правка одной строки (заменяет старую).
type ReportUpdate = clients.UpdateItem

// UpdateMappings — пакетное обновление.
func (s *ReportsService) UpdateMappings(ctx context.Context, updates []clients.UpdateItem) (*clients.UpdateResult, error) {
	return s.client.UpdateReports(ctx, updates)
}

var reportTimeLayouts = []string{
	"2006-01-02T15:04:05", // с секундами
	"2006-01-02T15:04",    // без секунд (реальный формат источника)
}

func parseReportTime(s string) (time.Time, error) {
	var lastErr error
	for _, layout := range reportTimeLayouts {
		t, err := time.ParseInLocation(layout, s, time.Local)
		if err == nil {
			return t, nil
		}
		lastErr = err
	}
	return time.Time{}, lastErr
}
