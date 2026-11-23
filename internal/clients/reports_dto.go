package clients

// =====================================================
// GET /reports
// =====================================================

type ReportsResponse struct {
	Items  []ReportItem `json:"items"`
	Total  int          `json:"total"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
}

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
	DepartmentID int64  `json:"department_id"`
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
	ID              int64  `json:"id"`
	ShortName       string `json:"short_name"`
	Name            string `json:"name"`
	WorkKindDefault string `json:"work_kind_default"`
}

// =====================================================
// POST /reports/update-work
// =====================================================

// UpdateResult — ответ POST /reports/update-work.
type UpdateResult struct {
	Updated int             `json:"updated"`
	Failed  []UpdateFailure `json:"failed"`
}

type UpdateFailure struct {
	ID    int64  `json:"id"`
	Error string `json:"error"`
}

// UpdatePayload — корневой объект запроса.
type UpdatePayload struct {
	Items []UpdateItem `json:"items"`
}

// UpdateItem — одна строка для обновления.
type UpdateItem struct {
	ID       int64  `json:"id"`
	UserID   int64  `json:"userid"`
	PrjID    string `json:"prj_id"`
	WorkKind string `json:"work_kind"`
	PrjStage string `json:"prj_stage"`
}
