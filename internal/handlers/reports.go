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
	"github.com/oleoleg/project-manager/internal/middleware"
	"github.com/oleoleg/project-manager/internal/models"
	"github.com/oleoleg/project-manager/internal/services"
)

type ReportsHandler struct {
	syncSvc      *services.ReportSyncService
	repo         *db.EmployeeRepo
	contractSvc  *services.ContractService
	settingsRepo *db.SettingsRepo
	tmpl         *template.Template
}

func NewReportsHandler(syncSvc *services.ReportSyncService, repo *db.EmployeeRepo, contractSvc *services.ContractService, settingsRepo *db.SettingsRepo, tmpl *template.Template) *ReportsHandler {
	return &ReportsHandler{
		syncSvc:      syncSvc,
		repo:         repo,
		contractSvc:  contractSvc,
		settingsRepo: settingsRepo,
		tmpl:         tmpl,
	}
}

func (h *ReportsHandler) Index(w http.ResponseWriter, r *http.Request) {
	filter := db.ReportFilter{}

	if cid := r.URL.Query().Get("contract_id"); cid != "" {
		filter.PrjID = &cid
	}
	if eid := r.URL.Query().Get("employee_id"); eid != "" {
		if id, err := strconv.ParseInt(eid, 10, 64); err == nil {
			filter.EmployeeID = &id
		}
	}
	if did := r.URL.Query().Get("department_id"); did != "" {
		if id, err := strconv.ParseInt(did, 10, 64); err == nil {
			filter.DepartmentID = &id
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

	currentURL, _ := h.settingsRepo.Get(r.Context(), "reports_api_url")
	if currentURL == "" {
		currentURL = "http://192.168.0.200:8000/reports/api/database" // дефолт
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

	// Получаем отделы и маппинг для формы
	departments, _ := h.repo.GetDepartments(r.Context())
	deptMappings, _ := h.repo.GetDepartmentMappings(r.Context())

	// 4. Собираем данные для шаблона
	data := h.commonData(r, "Рапорт сотрудников")
	data["Reports"] = rows
	data["Filter"] = filter
	data["Contracts"] = contracts
	data["Employees"] = employees
	data["WorkKinds"] = workKinds
	data["Flash"] = r.URL.Query().Get("flash")
	data["CurrentAPIURL"] = currentURL
	data["Departments"] = departments
	data["DepartmentMappings"] = deptMappings

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "reports.html", data); err != nil {
		log.Printf("!!! ОШИБКА РЕНДЕРА reports.html: %v", err)
		// Временно выводим ошибку прямо в браузер, чтобы увидеть, что не так
		http.Error(w, "Ошибка рендера: "+err.Error(), http.StatusInternalServerError)
	}
}

// Sync — POST /reports/sync
func (h *ReportsHandler) Sync(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()

	params := clients.ReportsParams{}
	if prjStr := r.FormValue("contract_id"); prjStr != "" {
		params.PrjID = &prjStr
	}

	res, err := h.syncSvc.Sync(r.Context(), params)

	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		log.Printf("sync reports: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{"error": err.Error()})
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

	// Возвращаемся на ту же страницу, откуда пришли (с фильтрами)
	referer := r.Referer()
	if referer == "" || !strings.Contains(referer, "/reports") {
		referer = "/reports"
	}

	// Добавляем flash-сообщение
	sep := "?"
	if strings.Contains(referer, "?") {
		sep = "&"
	}
	redirectURL := referer + sep + "flash=Изменения+сохранены"

	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (h *ReportsHandler) SaveSettings(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	url := r.FormValue("api_url")

	if err := h.settingsRepo.Set(r.Context(), "reports_api_url", url); err != nil {
		log.Printf("save settings: %v", err)
	}

	http.Redirect(w, r, "/reports?flash=URL+API+сохранен", http.StatusSeeOther)
}

func (h *ReportsHandler) DeleteAll(w http.ResponseWriter, r *http.Request) {
	if err := h.repo.DeleteAllReports(r.Context()); err != nil {
		log.Printf("delete all reports: %v", err)
		http.Redirect(w, r, "/reports?flash="+url.QueryEscape("Ошибка удаления: "+err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/reports?flash=Все+записи+удалены", http.StatusSeeOther)
}

func (h *ReportsHandler) SaveDepartmentMapping(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()

	// Проходим по всем полям формы (они имеют вид dept_1, dept_2 и т.д.)
	for key, values := range r.Form {
		if strings.HasPrefix(key, "dept_") && len(values) > 0 {
			deptIDStr := strings.TrimPrefix(key, "dept_")
			deptID, err := strconv.ParseInt(deptIDStr, 10, 64)
			if err != nil {
				continue
			}

			workKind := values[0]
			if workKind != "" {
				if err := h.repo.SaveDepartmentMapping(r.Context(), deptID, workKind); err != nil {
					log.Printf("save dept mapping %d: %v", deptID, err)
				}
			}
		}
	}

	http.Redirect(w, r, "/reports?flash=Настройки+отделов+сохранены", http.StatusSeeOther)
}
