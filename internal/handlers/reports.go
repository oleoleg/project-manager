package handlers

import (
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/oleoleg/project-manager/internal/clients"
	"github.com/oleoleg/project-manager/internal/db"
	"github.com/oleoleg/project-manager/internal/models"
	"github.com/oleoleg/project-manager/internal/services"
)

const (
	reportsPageLimit = 1000
	dateLayout       = "2006-01-02"
)

type ReportsHandler struct {
	reportsSvc  *services.ReportsService
	contractSvc *services.ContractService
	tmpl        *template.Template
}

func NewReportsHandler(reportsSvc *services.ReportsService, contractSvc *services.ContractService, tmpl *template.Template) *ReportsHandler {
	return &ReportsHandler{
		reportsSvc:  reportsSvc,
		contractSvc: contractSvc,
		tmpl:        tmpl,
	}
}

// Index — GET /reports
func (h *ReportsHandler) Index(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	f := services.ListFilter{Limit: reportsPageLimit, Offset: 0}

	if v := q.Get("contract_id"); v != "" {
		f.PrjID = &v
	}
	if v := q.Get("employee_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			f.UserID = &id
		}
	}
	if v := q.Get("department_id"); v != "" {
		f.DepartmentID = &v
	}
	if v := q.Get("stage"); v != "" {
		f.PrjStage = &v
	}

	now := time.Now()
	var from, to *time.Time
	if v := q.Get("from"); v != "" {
		if t, err := time.Parse(dateLayout, v); err == nil {
			from = &t
		}
	}
	if v := q.Get("to"); v != "" {
		if t, err := time.Parse(dateLayout, v); err == nil {
			to = &t
		}
	}
	if from == nil {
		t := time.Date(now.Year()-1, time.January, 1, 0, 0, 0, 0, time.Local)
		from = &t
	}
	if to == nil {
		t := now
		to = &t
	}
	f.From, f.To = from, to

	if v := q.Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			f.Offset = n
		}
	}

	page, err := h.reportsSvc.List(r.Context(), f)
	if err != nil {
		log.Printf("list reports: %v", err)
		http.Error(w, "Ошибка загрузки отчётов: "+err.Error(), http.StatusBadGateway)
		return
	}
	dict, err := h.reportsSvc.Dictionaries(r.Context())
	if err != nil {
		log.Printf("load reports dictionaries: %v", err)
		dict = &services.Dictionaries{}
	}

	contracts, _ := h.contractSvc.List(r.Context(), db.ContractFilter{})

	data := BaseData(r, "Рапорт сотрудников")
	data["Reports"] = page.Rows
	data["Total"] = page.Total
	data["Limit"] = page.Limit
	data["Offset"] = page.Offset
	data["HasNext"] = page.Offset+len(page.Rows) < page.Total
	data["HasPrev"] = page.Offset > 0
	data["PrevOffset"] = maxInt(page.Offset-page.Limit, 0)
	data["NextOffset"] = page.Offset + page.Limit
	data["QueryString"] = reportsQueryString(q)

	data["Filter"] = struct {
		PrjID        *string
		UserID       *int64
		DepartmentID *string
		PrjStage     *string
		From, To     *time.Time
	}{
		PrjID:        f.PrjID,
		UserID:       f.UserID,
		DepartmentID: f.DepartmentID,
		PrjStage:     f.PrjStage,
		From:         f.From,
		To:           f.To,
	}

	data["Contracts"] = contracts
	data["Employees"] = dict.Users
	data["Departments"] = dict.Departments
	data["WorkKinds"] = models.WorkKinds
	data["Flash"] = q.Get("flash")
	data["UpdateError"] = q.Get("update_error")

	RenderTemplate(w, h.tmpl, "reports.html", data)
}

func reportsQueryString(q url.Values) string {
	v := url.Values{}
	for _, k := range []string{"contract_id", "employee_id", "department_id", "stage", "from", "to", "offset"} {
		if val := q.Get(k); val != "" {
			v.Set(k, val)
		}
	}
	return v.Encode()
}

// UpdateMappings — POST /reports/update-work
func (h *ReportsHandler) UpdateMappings(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()

	ids := r.Form["report_id[]"]
	userIDs := r.Form["userid[]"]
	prjIDs := r.Form["prj_id[]"]
	workKinds := r.Form["work_kind[]"]
	stages := r.Form["prj_stage[]"]

	updates := make([]clients.UpdateItem, 0, len(ids))
	for i, idStr := range ids {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			continue
		}
		var uid int64
		if i < len(userIDs) {
			uid, _ = strconv.ParseInt(userIDs[i], 10, 64)
		}
		var pj, wk, st string
		if i < len(prjIDs) {
			pj = prjIDs[i]
		}
		if i < len(workKinds) {
			wk = workKinds[i]
		}
		if i < len(stages) {
			st = stages[i]
		}
		updates = append(updates, clients.UpdateItem{
			ID:       id,
			UserID:   uid,
			PrjID:    pj,
			WorkKind: wk,
			PrjStage: st,
		})
	}

	// Куда вернуться: page_query из формы (сохраняет фильтры и offset), иначе Referer.
	backURL := "/reports"
	if pq := r.FormValue("page_query"); pq != "" {
		backURL += "?" + pq
	} else if ref := r.Referer(); ref != "" {
		backURL = ref
	}

	if len(updates) == 0 {
		http.Redirect(w, r, appendQuery(backURL, "flash", "Нет изменений"), http.StatusSeeOther)
		return
	}

	res, err := h.reportsSvc.UpdateMappings(r.Context(), updates)
	if err != nil {
		log.Printf("update reports mappings: %v", err)
		http.Redirect(w, r, appendQuery(backURL, "update_error", err.Error()), http.StatusSeeOther)
		return
	}

	msg := "Обновлено: " + strconv.Itoa(res.Updated)
	if len(res.Failed) > 0 {
		msg += ", не удалось: " + strconv.Itoa(len(res.Failed))
	}
	http.Redirect(w, r, appendQuery(backURL, "flash", msg), http.StatusSeeOther)
}

// appendQuery аккуратно добавляет ?k=v или &k=v.
func appendQuery(u, k, v string) string {
	sep := "?"
	if containsQuery(u) {
		sep = "&"
	}
	return u + sep + k + "=" + url.QueryEscape(v)
}

func containsQuery(u string) bool {
	return len(u) > 0 && (indexOf(u, '?') >= 0)
}

func indexOf(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
