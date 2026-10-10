package handlers

import (
	"errors"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/oleoleg/project-manager/internal/db"
	"github.com/oleoleg/project-manager/internal/middleware"
	"github.com/oleoleg/project-manager/internal/models"
	"github.com/oleoleg/project-manager/internal/services"
)

type ContractHandler struct {
	svc   *services.ContractService
	cpSvc *services.CounterpartyService
	tmpl  *template.Template
}

func NewContractHandler(svc *services.ContractService, cpSvc *services.CounterpartyService, tmpl *template.Template) *ContractHandler {
	return &ContractHandler{svc: svc, cpSvc: cpSvc, tmpl: tmpl}
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

	data := h.commonData(r, "Договор "+c.ID)
	data["Contract"] = c
	data["Statuses"] = statuses
	data["Types"] = types
	data["Counterparty"] = cp
	data["StatusName"] = findStatusName(statuses, c.StatusCode)
	data["TypeName"] = findTypeName(types, c.ContractType)
	h.render(w, "contracts_show.html", data)
}

// ---------- New / Create ----------

func (h *ContractHandler) New(w http.ResponseWriter, r *http.Request) {
	statuses, _ := h.svc.Statuses(r.Context())
	types, _ := h.svc.Types(r.Context())
	cp, _ := h.cpSvc.List(r.Context(), "")
	pcs, _ := h.svc.ProcurementCardStatuses(r.Context())
	ts, _ := h.svc.TrackingStatuses(r.Context())

	data := h.commonData(r, "Новый договор")
	data["Contract"] = &models.Contract{}
	data["Statuses"] = statuses
	data["Types"] = types
	data["Counterparties"] = cp
	data["ProcurementStatuses"] = pcs
	data["TrackingStatuses"] = ts
	data["IsNew"] = true
	data["Error"] = ""
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

// ---------- helpers ----------

func (h *ContractHandler) renderFormWithError(w http.ResponseWriter, r *http.Request, c *models.Contract, isNew bool, errMsg string) {
	statuses, _ := h.svc.Statuses(r.Context())
	types, _ := h.svc.Types(r.Context())
	cp, _ := h.cpSvc.List(r.Context(), "")
	pcs, _ := h.svc.ProcurementCardStatuses(r.Context())
	ts, _ := h.svc.TrackingStatuses(r.Context())

	data := h.commonData(r, "Договор")
	data["Contract"] = c
	data["Statuses"] = statuses
	data["Types"] = types
	data["Counterparties"] = cp
	data["ProcurementStatuses"] = pcs
	data["TrackingStatuses"] = ts
	data["IsNew"] = isNew
	data["Error"] = errMsg

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
	return &models.Contract{
		ID:                    strings.TrimSpace(r.FormValue("id")),
		ContractNumber:        strings.TrimSpace(r.FormValue("contract_number")),
		ContractDate:          services.ParseDate(r.FormValue("contract_date")),
		WorkName:              strings.TrimSpace(r.FormValue("work_name")),
		ResponsiblePerson:     strings.TrimSpace(r.FormValue("responsible_person")),
		StatusCode:            statusCode,
		CounterpartyID:        strings.TrimSpace(r.FormValue("counterparty_id")),
		Workshop:              strings.TrimSpace(r.FormValue("workshop")),
		ProjectURL:            strings.TrimSpace(r.FormValue("project_url")),
		ProcurementCardStatus: nullIfEmpty(r.FormValue("procurement_card_status")),
		ExtraDetails:          strings.TrimSpace(r.FormValue("extra_details")), // это просто TEXT, можно оставить как есть
		FinplanStatus:         nullIfEmpty(r.FormValue("finplan_status")),
		PlanScheduleStatus:    nullIfEmpty(r.FormValue("plan_schedule_status")),
		ShortName:             strings.TrimSpace(r.FormValue("short_name")),
		ContractType:          strings.TrimSpace(r.FormValue("contract_type")),
	}
}

func humanizeContractError(err error) string {
	switch {
	case errors.Is(err, services.ErrContractIDFormat):
		return "ID должен быть в формате NNNN-AA, например 6666-П2 (Н — монтаж, П — поставка, Р — проектирование)"
	case errors.Is(err, services.ErrContractNumberRequired):
		return "Номер договора обязателен"
	case errors.Is(err, services.ErrContractCounterparty):
		return "Выберите контрагента"
	case errors.Is(err, services.ErrContractType):
		return "Выберите тип договора"
	case errors.Is(err, services.ErrContractExists):
		return "Договор с таким ID уже существует"
	}
	return "Не удалось сохранить: " + err.Error()

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
