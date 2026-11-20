package clients

// reportsResponse — ответ GET /reports.
type reportsResponse struct {
	Items  []reportItem `json:"items"`
	Total  int          `json:"total"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
}

type reportItem struct {
	ID          int64   `json:"id"`
	UserID      int64   `json:"userid"`
	PrjID       string  `json:"prj_id"`
	WorkType    int32   `json:"work_type"`
	StartTime   string  `json:"start_time"`
	EndTime     string  `json:"end_time"`
	Description string  `json:"description"`
	Timestamp   *string `json:"timestamp"`
}

// worktypesResponse — ответ GET /worktypes.
type worktypesResponse struct {
	Items []worktypeItem `json:"items"`
}

type worktypeItem struct {
	ID             int32  `json:"id"`
	Name           string `json:"name"`
	CalculateHours bool   `json:"calculate_hours"`
}

// usersResponse — ответ GET /users.
type usersResponse struct {
	Items []userItem `json:"items"`
}

type userItem struct {
	ID         int64  `json:"id"`
	Username   string `json:"username"`
	Longname   string `json:"longname"`
	City       string `json:"city"`
	Email      string `json:"email"`
	IsAdmin    bool   `json:"is_admin"`
	IsBlocked  bool   `json:"is_blocked"`
	Department string `json:"department"`
}

// projectsResponse — ответ GET /projects.
type projectsResponse struct {
	Items []projectItem `json:"items"`
}

type projectItem struct {
	PrjID       string `json:"prj_id"`
	Description string `json:"description"`
}
