package clients

// =====================================================
// GET /reports
// =====================================================

// ReportsResponse — ответ GET /reports.
type ReportsResponse struct {
	Items  []ReportItem `json:"items"`
	Total  int          `json:"total"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
}

// ReportItem — одна сырая запись Reports источника.
// Даты приходят строками в формате 2006-01-02T15:04:05 (локальное, без TZ).
type ReportItem struct {
	ID          int64   `json:"id"`
	UserID      int64   `json:"userid"`
	PrjID       string  `json:"prj_id"`
	WorkType    int32   `json:"work_type"`
	StartTime   string  `json:"start_time"`
	EndTime     string  `json:"end_time"`
	Description string  `json:"description"`
	Timestamp   *string `json:"timestamp"`
	WorkKind    *string `json:"work_kind"`
	PrjStage    *string `json:"prj_stage"`
}

// =====================================================
// GET /worktypes
// =====================================================

type WorktypesResponse struct {
	Items []WorktypeItem `json:"items"`
}

type WorktypeItem struct {
	ID             int32  `json:"id"`
	Name           string `json:"name"`
	CalculateHours bool   `json:"calculate_hours"`
}

// =====================================================
// GET /users
// =====================================================

type UsersResponse struct {
	Items []UserItem `json:"items"`
}

type UserItem struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	Longname     string `json:"longname"`
	City         string `json:"city"`
	Email        string `json:"email"`
	IsAdmin      bool   `json:"is_admin"`
	IsBlocked    bool   `json:"is_blocked"`
	DepartmentID string `json:"department_id"`
}

// =====================================================
// GET /projects
// =====================================================

type ProjectsResponse struct {
	Items []ProjectItem `json:"items"`
}

type ProjectItem struct {
	PrjID       string `json:"prj_id"`
	Description string `json:"description"`
}

// =====================================================
// GET /departments
// =====================================================

type DepartmentsResponse struct {
	Items []DepartmentItem `json:"items"`
}

type DepartmentItem struct {
	ID         string `json:"id"`
	Department string `json:"department"`
	WorkKind   string `json:"work_kind"`
}
