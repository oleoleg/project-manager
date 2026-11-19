package handlers

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/oleoleg/project-manager/internal/clients"
	"github.com/oleoleg/project-manager/internal/db"
	"github.com/oleoleg/project-manager/internal/middleware"
	"github.com/oleoleg/project-manager/internal/models"
	"github.com/oleoleg/project-manager/internal/services"
)

type ReportsHandler struct {
	syncSvc     *services.ReportSyncService
	repo        *db.EmployeeRepo
	contractSvc *services.ContractService
	tmpl        *template.Template
}

func NewReportsHandler(syncSvc *services.ReportSyncService, repo *db.EmployeeRepo, contractSvc *services.ContractService, tmpl *template.Template) *ReportsHandler {
	return &ReportsHandler{
		syncSvc:     syncSvc,
		repo:        repo,
		contractSvc: contractSvc,
		tmpl:        tmpl,
	}
}

func (h *ReportsHandler) Index(w http.ResponseWriter, r *http.Request) {
	filter := db.ReportFilter{}

	if cid := r.URL.Query().Get("contract_id"); cid != "" {
		if id, err := strconv.ParseInt(cid, 10, 64); err == nil {
			filter.PrjID = &id
		}
	}
	if eid := r.URL.Query().Get("employee_id"); eid != "" {
		if id, err := strconv.ParseInt(eid, 10, 64); err == nil {
			filter.EmployeeID = &id
		}
	}
	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		if t, err := time.Parse("2006-01-02", fromStr); err == nil {
			filter.From = &t
		}
	}
	if toStr := r.URL.Query().Get("to"); toStr != "" {
		if t, err := time.Parse("2006-01-02", toStr); err == nil {
			filter.To = &t
		}
	}

	// 2. Получаем данные
	reports, err := h.repo.ListReports(r.Context(), filter)
	if err != nil {
		log.Printf("list reports: %v", err)
		http.Error(w, "Ошибка загрузки отчётов", http.StatusInternalServerError)
		return
	}

	// 3. Считаем WorklogRow
	var rows []models.WorklogRow
	for _, rep := range reports {
		rows = append(rows, services.BuildWorklogRow(rep))
	}

	// Получаем данные для фильтров (договоры и сотрудники)
	contracts, _ := h.contractSvc.List(r.Context(), db.ContractFilter{})
	employees, _ := h.repo.ListEmployees(r.Context())
	workKinds := models.WorkKinds // Берем из models/stage.go

	// 4. Собираем данные для шаблона
	data := h.commonData(r, "Рапорт сотрудников")
	data["Reports"] = rows
	data["Filter"] = filter
	data["Contracts"] = contracts
	data["Employees"] = employees
	data["WorkKinds"] = workKinds
	data["Flash"] = r.URL.Query().Get("flash")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "reports.html", data); err != nil {
		log.Printf("render reports.html: %v", err)
	}
}

// Sync — POST /reports/sync
func (h *ReportsHandler) Sync(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()

	params := clients.ReportsParams{}
	if prjStr := r.FormValue("contract_id"); prjStr != "" {
		if id, err := strconv.ParseInt(prjStr, 10, 64); err == nil {
			params.PrjID = &id
		}
	}

	res, err := h.syncSvc.Sync(r.Context(), params)

	// Всегда отдаем JSON
	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		log.Printf("sync reports: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"error": err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]any{
		"success":     true,
		"departments": res.Departments,
		"employees":   res.Employees,
		"worktypes":   res.WorkTypes,
		"projects":    res.Projects,
		"reports":     res.Reports,
		"skipped":     res.Skipped,
		"errors":      res.Errors,
	})
}

func (h *ReportsHandler) commonData(r *http.Request, title string) map[string]any {
	u := middleware.UserFromContext(r.Context())
	data := map[string]any{
		"Title": title,
		"User":  u,
	}
	if u != nil {
		data["RoleName"] = models.RoleName(u.RoleCode)
	}
	return data
}

func (h *ReportsHandler) SaveMapping(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()

	ids := r.Form["report_id[]"]
	workKinds := r.Form["work_kind[]"]
	stageNumbers := r.Form["stage_number[]"]

	if len(ids) == 0 {
		http.Redirect(w, r, "/reports", http.StatusSeeOther)
		return
	}

	for i, idStr := range ids {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			continue
		}

		wk := ""
		if i < len(workKinds) {
			wk = workKinds[i]
		}

		sn := ""
		if i < len(stageNumbers) {
			sn = stageNumbers[i]
		}

		if err := h.repo.UpdateReportMapping(r.Context(), id, wk, sn); err != nil {
			log.Printf("save mapping for report %d: %v", id, err)
		}
	}

	http.Redirect(w, r, "/reports?flash=Изменения+сохранены", http.StatusSeeOther)
}
