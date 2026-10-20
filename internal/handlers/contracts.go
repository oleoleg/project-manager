package handlers

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/oleoleg/project-manager/internal/db"
	"github.com/oleoleg/project-manager/internal/middleware"
	"github.com/oleoleg/project-manager/internal/models"
	"github.com/oleoleg/project-manager/internal/services"
)

type ContractHandler struct {
	svc      *services.ContractService
	cpSvc    *services.CounterpartyService
	stageSvc *services.StageService
	tmpl     *template.Template
}

func NewContractHandler(svc *services.ContractService, cpSvc *services.CounterpartyService, stageSvc *services.StageService, tmpl *template.Template) *ContractHandler {
	return &ContractHandler{svc: svc, cpSvc: cpSvc, stageSvc: stageSvc, tmpl: tmpl}
}

func (h *ContractHandler) commonData(r *http.Request, title string) map[string]any {
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

func (h *ContractHandler) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("template %s: %v", name, err)
	}
}

// ---------- List ----------

func (h *ContractHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	f := db.ContractFilter{
		Search:         strings.TrimSpace(q.Get("q")),
		CounterpartyID: strings.TrimSpace(q.Get("counterparty")),
	}
	if sc := q.Get("status"); sc != "" {
		if n, err := strconv.Atoi(sc); err == nil {
			v := int16(n)
			f.StatusCode = &v
		}
	}

	list, err := h.svc.List(r.Context(), f)
	if err != nil {
		log.Printf("contracts list: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	statuses, _ := h.svc.Statuses(r.Context())
	counterparties, _ := h.cpSvc.List(r.Context(), "")

	data := h.commonData(r, "Договоры")
	data["List"] = list
	data["Statuses"] = statuses
	data["Counterparties"] = counterparties
	data["Search"] = f.Search
	data["FilterCounterparty"] = f.CounterpartyID
	if f.StatusCode != nil {
		data["FilterStatus"] = *f.StatusCode
	} else {
		data["FilterStatus"] = -1
	}
	data["Flash"] = q.Get("flash")
	h.render(w, "contracts_list.html", data)
}

// ---------- Show ----------

func (h *ContractHandler) Show(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	c, err := h.svc.GetByID(r.Context(), id)
	if errors.Is(err, db.ErrContractNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("contract show: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	statuses, _ := h.svc.Statuses(r.Context())
	types, _ := h.svc.Types(r.Context())
	cp, _ := h.cpSvc.GetByID(r.Context(), c.CounterpartyID)
	totals, _ := h.stageSvc.Totals(r.Context(), c.ID)
	stageStatuses, _ := h.stageSvc.Statuses(r.Context())
	assignees, _ := h.svc.ListAssignees(r.Context(), c.ID)
	costTotals, _ := h.stageSvc.StageCostTotals(r.Context(), c.ID)

	data := h.commonData(r, "Договор "+c.ID)
	data["Contract"] = c
	data["Statuses"] = statuses
	data["Types"] = types
	data["Counterparty"] = cp
	data["StatusName"] = findStatusName(statuses, c.StatusCode)
	data["TypeName"] = findTypeName(types, c.ContractType)
	data["Totals"] = totals
	data["StageStatuses"] = stageStatuses
	data["StageStatusName"] = func(code int16) string {
		for _, s := range stageStatuses {
			if s.Code == code {
				return s.Name
			}
		}
		return ""
	}
	data["Today"] = time.Now()
	data["CostTotals"] = costTotals

	var responsible *models.User
	if c.ResponsibleUserID != nil {
		responsible, _ = h.svc.GetResponsible(r.Context(), *c.ResponsibleUserID)
	}
	data["Assignees"] = assignees
	data["Responsible"] = responsible

	// Параметры отображения
	vatMode := r.URL.Query().Get("vat")
	if vatMode != "with" {
		vatMode = "without"
	}
	data["VATMode"] = vatMode

	showAll := r.URL.Query().Get("stages") == "all"
	showCosts := r.URL.Query().Get("costs") == "1"
	data["ShowAll"] = showAll
	data["ShowCosts"] = showCosts

	// Полный список этапов — используется и для VatSingle/VatMixed,
	// и для подготовки problems/allStages.
	allStagesForVAT, _ := h.stageSvc.List(r.Context(), c.ID)

	// Единый список этапов для отображения, с учётом showAll.
	var srcStages []models.Stage
	if showAll {
		srcStages = allStagesForVAT
	} else {
		// только «в работе» (все, кроме закрытых и гарантии)
		for _, s := range allStagesForVAT {
			if s.StatusCode == 3 || s.StatusCode == 5 {
				continue
			}
			srcStages = append(srcStages, s)
		}
	}

	// problems — для компактной таблицы
	var problems []db.ProblemStage
	for _, s := range srcStages {
		problems = append(problems, db.ProblemStage{
			GUID:              s.GUID,
			StageNumber:       s.StageNumber,
			Name:              s.Name,
			StatusCode:        s.StatusCode,
			CloseDatePlan:     s.CloseDatePlan,
			ResponsiblePerson: s.ResponsiblePerson,
			Note:              s.Note,
			Remaining:         s.Cost - s.ClosedByActs,
		})
	}

	// allStages — для расширенной таблицы (только если showCosts)
	type stageRow struct {
		GUID           string
		StageNumber    string
		Name           string
		StatusCode     int16
		Cost           float64
		Closed         float64
		Expenses       float64
		VATPercent     float64
		AdvancePercent float64
	}
	var allStages []stageRow
	if showCosts {
		for _, s := range srcStages {
			allStages = append(allStages, stageRow{
				GUID:           s.GUID,
				StageNumber:    s.StageNumber,
				Name:           s.Name,
				StatusCode:     s.StatusCode,
				Cost:           s.Cost,
				Closed:         s.ClosedByActs,
				Expenses:       costTotals[s.GUID],
				VATPercent:     s.VATPercent,
				AdvancePercent: s.AdvancePercent,
			})
		}
	}

	data["Problems"] = problems
	data["AllStages"] = allStages

	// Определяем единую ставку НДС по всем этапам договора.
	// Если все этапы имеют одинаковую ставку — показываем её в карточке-итоге.
	// Если разные — «смешанный НДС». Если этапов нет — НДС не начисляется.
	var vatSingle float64
	var vatMixed bool
	seen := map[float64]bool{}
	for _, s := range allStagesForVAT {
		seen[s.VATPercent] = true
	}
	switch len(seen) {
	case 0:
		vatSingle = 0
	case 1:
		for k := range seen {
			vatSingle = k
		}
	default:
		vatMixed = true
	}
	data["VatSingle"] = vatSingle
	data["VatMixed"] = vatMixed

	h.render(w, "contracts_show.html", data)
}

// ---------- New / Create ----------

func (h *ContractHandler) New(w http.ResponseWriter, r *http.Request) {
	statuses, _ := h.svc.Statuses(r.Context())
	types, _ := h.svc.Types(r.Context())
	cp, _ := h.cpSvc.List(r.Context(), "")
	pcs, _ := h.svc.ProcurementCardStatuses(r.Context())
	ts, _ := h.svc.TrackingStatuses(r.Context())
	users, _ := h.svc.ListActiveUsers(r.Context())

	data := h.commonData(r, "Новый договор")
	data["Contract"] = &models.Contract{}
	data["Statuses"] = statuses
	data["Types"] = types
	data["Counterparties"] = cp
	data["ProcurementStatuses"] = pcs
	data["TrackingStatuses"] = ts
	data["IsNew"] = true
	data["Error"] = ""
	data["ActiveUsers"] = users
	h.render(w, "contracts_form.html", data)
}

func (h *ContractHandler) Create(w http.ResponseWriter, r *http.Request) {
	c := parseContractForm(r)

	if err := h.svc.Create(r.Context(), c); err != nil {
		h.renderFormWithError(w, r, c, true, humanizeContractError(err))
		return
	}

	http.Redirect(w, r, "/contracts?flash="+url.QueryEscape("Договор создан"), http.StatusSeeOther)
}

// ---------- Edit / Update ----------

func (h *ContractHandler) Edit(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	c, err := h.svc.GetByID(r.Context(), id)
	if errors.Is(err, db.ErrContractNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("contract edit: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.renderFormWithError(w, r, c, false, "")
}

func (h *ContractHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	c := parseContractForm(r)
	c.ID = id

	if err := h.svc.Update(r.Context(), c); err != nil {
		h.renderFormWithError(w, r, c, false, humanizeContractError(err))
		return
	}

	http.Redirect(w, r, "/contracts?flash="+url.QueryEscape("Изменения сохранены"), http.StatusSeeOther)
}

// ---------- Delete ----------

func (h *ContractHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	err := h.svc.Delete(r.Context(), id)
	if errors.Is(err, db.ErrContractNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("contract delete: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/contracts?flash="+url.QueryEscape("Договор удалён"), http.StatusSeeOther)
}

// OpenFolder отдаёт страницу, которая открывает папку проекта
// через проводник Windows. Работает по URL-протоколу pm://,
// зарегистрированному установщиком.
func (h *ContractHandler) OpenFolder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	c, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if c.ProjectURL == "" {
		http.Error(w, "Путь к корню проекта не задан", http.StatusNotFound)
		return
	}

	target := "pm://open?path=" + url.QueryEscape(c.ProjectURL)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="ru"><head><meta charset="utf-8"><title>Открытие папки…</title></head>
<body>
<script>
    window.location.href = %q;
    setTimeout(function(){ window.close(); }, 500);
</script>
<p>Открываем папку… Если ничего не произошло, установите Project Manager на этом компьютере.</p>
</body></html>`, target)
}

// ---------- helpers ----------

func (h *ContractHandler) renderFormWithError(w http.ResponseWriter, r *http.Request, c *models.Contract, isNew bool, errMsg string) {
	statuses, _ := h.svc.Statuses(r.Context())
	types, _ := h.svc.Types(r.Context())
	cp, _ := h.cpSvc.List(r.Context(), "")
	pcs, _ := h.svc.ProcurementCardStatuses(r.Context())
	ts, _ := h.svc.TrackingStatuses(r.Context())
	users, _ := h.svc.ListActiveUsers(r.Context())

	data := h.commonData(r, "Договор")
	data["Contract"] = c
	data["Statuses"] = statuses
	data["Types"] = types
	data["Counterparties"] = cp
	data["ProcurementStatuses"] = pcs
	data["TrackingStatuses"] = ts
	data["IsNew"] = isNew
	data["Error"] = errMsg
	data["ActiveUsers"] = users
	if errMsg != "" {
		w.WriteHeader(http.StatusBadRequest)
	}
	h.render(w, "contracts_form.html", data)
}

func parseContractForm(r *http.Request) *models.Contract {
	_ = r.ParseForm()
	statusCode := int16(0)
	if v, err := strconv.Atoi(r.FormValue("status_code")); err == nil {
		statusCode = int16(v)
	}

	var responsibleID *int64
	if v := r.FormValue("responsible_user_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			responsibleID = &n
		}
	}

	return &models.Contract{
		ID:                    strings.TrimSpace(r.FormValue("id")),
		ContractNumber:        strings.TrimSpace(r.FormValue("contract_number")),
		ContractDate:          services.ParseDate(r.FormValue("contract_date")),
		WorkName:              strings.TrimSpace(r.FormValue("work_name")),
		ResponsiblePerson:     strings.TrimSpace(r.FormValue("responsible_person")),
		ResponsibleUserID:     responsibleID,
		StatusCode:            statusCode,
		CounterpartyID:        strings.TrimSpace(r.FormValue("counterparty_id")),
		Workshop:              strings.TrimSpace(r.FormValue("workshop")),
		ProjectURL:            normalizeServerPath(r.FormValue("project_url")),
		ProcurementCardStatus: nullIfEmpty(r.FormValue("procurement_card_status")),
		ExtraDetails:          strings.TrimSpace(r.FormValue("extra_details")),
		FinplanStatus:         nullIfEmpty(r.FormValue("finplan_status")),
		PlanScheduleStatus:    nullIfEmpty(r.FormValue("plan_schedule_status")),
		ShortName:             strings.TrimSpace(r.FormValue("short_name")),
		ContractType:          strings.TrimSpace(r.FormValue("contract_type")),
	}
}

func humanizeContractError(err error) string {
	switch {
	case errors.Is(err, services.ErrContractIDFormat):
		return "ID должен начинаться с 4 цифр номера проекта. Примеры: 6666, 6666-P, 6666-P2. Буквы — латинские: W (монтаж), S (поставка), D (проектирование)"
	case errors.Is(err, services.ErrContractCounterparty):
		return "Выберите контрагента"
	case errors.Is(err, services.ErrContractType):
		return "Выберите тип договора"
	case errors.Is(err, services.ErrContractExists):
		return "Договор с таким ID уже существует"
	case errors.Is(err, services.ErrContractProjectURLRequired):
		return "Укажите путь к корню проекта"
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return "Одно из полей-справочников содержит недопустимое значение: " + pgErr.ConstraintName
	}
	return "Не удалось сохранить: " + err.Error()
}

func findStatusName(list []models.ContractStatusRef, code int16) string {
	for _, s := range list {
		if s.Code == code {
			return s.Name
		}
	}
	return ""
}

// nullIfEmpty возвращает указатель на строку, либо nil, если строка пустая.
// Нужно для полей, у которых FK: пустая строка ≠ NULL, и БД её отвергает.
func nullIfEmpty(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

func findTypeName(list []models.ContractTypeRef, code string) string {
	for _, t := range list {
		if t.Code == code {
			return t.Name
		}
	}
	return ""
}

// normalizeServerPath приводит путь к рабочему виду:
//   - убирает пробелы по краям
//   - убирает обрамляющие кавычки "..." (частая ошибка при копировании пути)
//   - заменяет прямые слэши на обратные (чтобы работал explorer)
//   - убирает trailing-слэш, кроме случаев, когда это корень диска (C:\) или UNC (\\server\share\)
func normalizeServerPath(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	s = strings.ReplaceAll(s, "/", `\`)

	// trailing-слэш оставляем только если это:
	//   - корень диска: C:\
	//   - UNC-корень:   \\server\share\
	if strings.HasSuffix(s, `\`) {
		trimmed := strings.TrimRight(s, `\`)
		// C:\  или  \\server\share\  — оставляем
		if len(trimmed) == 2 && trimmed[1] == ':' {
			return trimmed + `\`
		}
		if strings.HasPrefix(trimmed, `\\`) && strings.Count(trimmed, `\`) == 3 {
			return trimmed + `\`
		}
		s = trimmed
	}
	return s
}
