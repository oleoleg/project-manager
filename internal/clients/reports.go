package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	reportsDateTimeFormat = "2006-01-02T15:04:05"
	reportsDateFormat     = "2006-01-02"
)

// ReportsClient — проксирующий клиент API «Работа сотрудников по проектам».
// Ничего не кэширует и не сохраняет: каждый вызов — один HTTP-запрос.
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

// ReportsQuery — параметры выборки отчётов (все опциональны).
type ReportsQuery struct {
	From         *time.Time // по дате start_time, включительно
	To           *time.Time
	PrjID        *string
	UserID       *int64
	DepartmentID *string
	PrjStage     *string
	Limit        int // 0 — дефолт источника (1000)
	Offset       int
}

// ReportUpdate — правка одной строки отчёта.
type ReportUpdate struct {
	ID       int64  `json:"id"`
	WorkKind string `json:"work_kind"`
	PrjStage string `json:"prj_stage"`
}

// UpdateResult — ответ POST /reports/update.
type UpdateResult struct {
	Updated int             `json:"updated"`
	Failed  []UpdateFailure `json:"failed"`
}

type UpdateFailure struct {
	ID    int64  `json:"id"`
	Error string `json:"error"`
}

// FetchReports — один GET /reports с параметрами. Возвращает страницу как есть.
func (c *ReportsClient) FetchReports(ctx context.Context, q ReportsQuery) (*ReportsResponse, error) {
	u, err := url.Parse(c.baseURL + "/reports")
	if err != nil {
		return nil, fmt.Errorf("parse reports url: %w", err)
	}
	v := u.Query()
	if q.From != nil {
		v.Set("from", q.From.Format(reportsDateFormat))
	}
	if q.To != nil {
		v.Set("to", q.To.Format(reportsDateFormat))
	}
	if q.PrjID != nil {
		v.Set("prj_id", *q.PrjID)
	}
	if q.UserID != nil {
		v.Set("userid", strconv.FormatInt(*q.UserID, 10))
	}
	if q.DepartmentID != nil {
		v.Set("department_id", *q.DepartmentID)
	}
	if q.PrjStage != nil {
		v.Set("prj_stage", *q.PrjStage)
	}
	if q.Limit > 0 {
		v.Set("limit", strconv.Itoa(q.Limit))
	}
	if q.Offset > 0 {
		v.Set("offset", strconv.Itoa(q.Offset))
	}
	u.RawQuery = v.Encode()

	var resp ReportsResponse
	if err := c.getJSON(ctx, u.String(), &resp); err != nil {
		return nil, fmt.Errorf("fetch reports: %w", err)
	}
	return &resp, nil
}

func (c *ReportsClient) FetchWorkTypes(ctx context.Context) ([]WorktypeItem, error) {
	var resp WorktypesResponse
	if err := c.getJSON(ctx, c.baseURL+"/worktypes", &resp); err != nil {
		return nil, fmt.Errorf("fetch worktypes: %w", err)
	}
	return resp.Items, nil
}

func (c *ReportsClient) FetchUsers(ctx context.Context) ([]UserItem, error) {
	var resp UsersResponse
	if err := c.getJSON(ctx, c.baseURL+"/users", &resp); err != nil {
		return nil, fmt.Errorf("fetch users: %w", err)
	}
	return resp.Items, nil
}

func (c *ReportsClient) FetchProjects(ctx context.Context) ([]ProjectItem, error) {
	var resp ProjectsResponse
	if err := c.getJSON(ctx, c.baseURL+"/projects", &resp); err != nil {
		return nil, fmt.Errorf("fetch projects: %w", err)
	}
	return resp.Items, nil
}

func (c *ReportsClient) FetchDepartments(ctx context.Context) ([]DepartmentItem, error) {
	var resp DepartmentsResponse
	if err := c.getJSON(ctx, c.baseURL+"/departments", &resp); err != nil {
		return nil, fmt.Errorf("fetch departments: %w", err)
	}
	return resp.Items, nil
}

// UpdateReports — один POST /reports/update с массивом правок.
func (c *ReportsClient) UpdateReports(ctx context.Context, updates []ReportUpdate) (*UpdateResult, error) {
	if len(updates) == 0 {
		return &UpdateResult{}, nil
	}

	body, err := json.Marshal(updates)
	if err != nil {
		return nil, fmt.Errorf("marshal updates: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/reports/update", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("post reports/update: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reports/update: unexpected status %d", res.StatusCode)
	}

	var out UpdateResult
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode reports/update: %w", err)
	}
	return &out, nil
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
	return json.NewDecoder(res.Body).Decode(dst)
}
