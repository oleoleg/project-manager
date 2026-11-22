package handlers

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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

	// 1. Фильтры
	f := services.ListFilter{
		Limit:  reportsPageLimit,
		Offset: 0,
	}
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

	// from/to: если не заданы — с начала предыдущего года по сегодня.
	now := time.Now()
	from, to := f.From, f.To
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

	// 2. Данные
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

	// 3. Договоры (для фильтра «Введите или выберите договор»)
	contracts, _ := h.contractSvc.List(r.Context(), db.ContractFilter{})

	// 4. Данные для шаблона
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
	for _, k := range []string{"contract_id", "employee_id", "department_id", "stage", "from", "to"} {
		if val := q.Get(k); val != "" {
			v.Set(k, val)
		}
	}
	return v.Encode()
}

// UpdateMappings — POST /reports/update-mappings
// Принимает форму с массивами report_id[] / work_kind[] / prj_stage[]
// и одним POST-ом отправляет правки на источник.
func (h *ReportsHandler) UpdateMappings(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()

	ids := r.Form["report_id[]"]
	workKinds := r.Form["work_kind[]"]
	stages := r.Form["prj_stage[]"]

	updates := make([]clients.ReportUpdate, 0, len(ids))
	for i, idStr := range ids {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			continue
		}
		var wk, st string
		if i < len(workKinds) {
			wk = workKinds[i]
		}
		if i < len(stages) {
			st = stages[i]
		}
		updates = append(updates, clients.ReportUpdate{ID: id, WorkKind: wk, PrjStage: st})
	}

	referer := r.Referer()
	if referer == "" {
		referer = "/reports"
	}

	if len(updates) == 0 {
		http.Redirect(w, r, referer+"?flash="+urlQueryEscape("Нет изменений"), http.StatusSeeOther)
		return
	}

	res, err := h.reportsSvc.UpdateMappings(r.Context(), updates)
	if err != nil {
		log.Printf("update reports mappings: %v", err)
		http.Redirect(w, r, referer+"?update_error="+urlQueryEscape(err.Error()), http.StatusSeeOther)
		return
	}

	msg := "Обновлено: " + strconv.Itoa(res.Updated)
	if len(res.Failed) > 0 {
		msg += ", не удалось: " + strconv.Itoa(len(res.Failed))
	}
	sep := "?"
	if len(referer) > 0 && containsQuery(referer) {
		sep = "&"
	}
	http.Redirect(w, r, referer+sep+"flash="+urlQueryEscape(msg), http.StatusSeeOther)
}

// --- мелкие хелперы (можно вынести в util, если уже есть) ---

func urlQueryEscape(s string) string { return url.QueryEscape(s) }

func containsQuery(u string) bool { return strings.Contains(u, "?") }

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// на случай, если json понадобится ниже — оставим импорт «живым»
var _ = json.Marshal
