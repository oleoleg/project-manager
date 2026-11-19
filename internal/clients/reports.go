package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/oleoleg/project-manager/internal/models"
)

const (
	reportsDateFormat = "2006-01-02T15:04:05"
	reportsPageSize   = 1000
)

// ReportsClient — клиент API сервера «Работа сотрудников по проектам».
type ReportsClient struct {
	baseURL string
	http    *http.Client
}

func NewReportsClient(baseURL string) *ReportsClient {
	return &ReportsClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// ReportsParams — параметры выборки отчётов.
type ReportsParams struct {
	From   *time.Time
	To     *time.Time
	PrjID  *int64
	UserID *int64
}

// ReportRaw — сырая запись для маппинга в сервисе.
type ReportRaw struct {
	SourceID    int64
	UserID      int64
	PrjID       int64
	WorkTypeID  int32
	StartTime   time.Time
	EndTime     time.Time
	Description string
	SourceTS    *time.Time
}

// FetchReports забирает все записи (с пагинацией).
func (c *ReportsClient) FetchReports(ctx context.Context, p ReportsParams) ([]ReportRaw, error) {
	var result []ReportRaw
	offset := 0

	for {
		u, err := url.Parse(c.baseURL + "/reports")
		if err != nil {
			return nil, fmt.Errorf("parse reports url: %w", err)
		}
		q := u.Query()
		if p.From != nil {
			q.Set("from", p.From.Format("2006-01-02"))
		}
		if p.To != nil {
			q.Set("to", p.To.Format("2006-01-02"))
		}
		if p.PrjID != nil {
			q.Set("prj_id", strconv.FormatInt(*p.PrjID, 10))
		}
		if p.UserID != nil {
			q.Set("userid", strconv.FormatInt(*p.UserID, 10))
		}
		q.Set("limit", strconv.Itoa(reportsPageSize))
		q.Set("offset", strconv.Itoa(offset))
		u.RawQuery = q.Encode()

		var resp reportsResponse
		if err := c.getJSON(ctx, u.String(), &resp); err != nil {
			return nil, fmt.Errorf("fetch reports (offset=%d): %w", offset, err)
		}

		for _, it := range resp.Items {
			rep, err := mapReportItem(it)
			if err != nil {
				log.Printf("reports client: skip report id=%d: %v", it.ID, err)
				continue
			}
			result = append(result, rep)
		}

		if len(resp.Items) == 0 || offset+len(resp.Items) >= resp.Total {
			break
		}
		offset += len(resp.Items)
	}

	return result, nil
}

// FetchWorkTypes забирает справочник видов работ.
func (c *ReportsClient) FetchWorkTypes(ctx context.Context) ([]models.EmployeeWorkType, error) {
	var resp worktypesResponse
	if err := c.getJSON(ctx, c.baseURL+"/worktypes", &resp); err != nil {
		return nil, fmt.Errorf("fetch worktypes: %w", err)
	}
	out := make([]models.EmployeeWorkType, 0, len(resp.Items))
	for _, it := range resp.Items {
		out = append(out, models.EmployeeWorkType{
			SourceID:       it.ID,
			Name:           it.Name,
			CalculateHours: it.CalculateHours,
		})
	}
	return out, nil
}

// FetchUsers забирает справочник сотрудников (без пароля).
func (c *ReportsClient) FetchUsers(ctx context.Context) ([]models.Employee, error) {
	var resp usersResponse
	if err := c.getJSON(ctx, c.baseURL+"/users", &resp); err != nil {
		return nil, fmt.Errorf("fetch users: %w", err)
	}
	out := make([]models.Employee, 0, len(resp.Items))
	for _, it := range resp.Items {
		out = append(out, models.Employee{
			SourceID:       it.ID,
			Username:       it.Username,
			Longname:       it.Longname,
			City:           it.City,
			Email:          it.Email,
			IsAdmin:        it.IsAdmin,
			IsBlocked:      it.IsBlocked,
			DepartmentName: it.Department,
		})
	}
	return out, nil
}

// FetchProjects забирает справочник проектов.
func (c *ReportsClient) FetchProjects(ctx context.Context) ([]models.EmployeeProject, error) {
	var resp projectsResponse
	if err := c.getJSON(ctx, c.baseURL+"/projects", &resp); err != nil {
		return nil, fmt.Errorf("fetch projects: %w", err)
	}
	out := make([]models.EmployeeProject, 0, len(resp.Items))
	for _, it := range resp.Items {
		out = append(out, models.EmployeeProject{
			PrjID:       it.PrjID,
			Description: it.Description,
		})
	}
	return out, nil
}

func (c *ReportsClient) getJSON(ctx context.Context, url string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", res.StatusCode)
	}
	if err := json.NewDecoder(res.Body).Decode(dst); err != nil {
		return fmt.Errorf("decode json: %w", err)
	}
	return nil
}

func mapReportItem(it reportItem) (ReportRaw, error) {
	start, err := time.ParseInLocation(reportsDateFormat, it.StartTime, time.Local)
	if err != nil {
		return ReportRaw{}, fmt.Errorf("parse start_time %q: %w", it.StartTime, err)
	}
	end, err := time.ParseInLocation(reportsDateFormat, it.EndTime, time.Local)
	if err != nil {
		return ReportRaw{}, fmt.Errorf("parse end_time %q: %w", it.EndTime, err)
	}
	if !end.After(start) {
		return ReportRaw{}, fmt.Errorf("end_time <= start_time")
	}

	var sourceTS *time.Time
	if it.Timestamp != nil && *it.Timestamp != "" {
		if ts, err := time.ParseInLocation(reportsDateFormat, *it.Timestamp, time.Local); err == nil {
			sourceTS = &ts
		}
	}

	return ReportRaw{
		SourceID:    it.ID,
		UserID:      it.UserID,
		PrjID:       it.PrjID,
		WorkTypeID:  it.WorkType,
		StartTime:   start,
		EndTime:     end,
		Description: it.Description,
		SourceTS:    sourceTS,
	}, nil
}
