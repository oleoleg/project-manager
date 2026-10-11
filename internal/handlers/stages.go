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

	"github.com/go-chi/chi/v5"

	"github.com/oleoleg/project-manager/internal/db"
	"github.com/oleoleg/project-manager/internal/middleware"
	"github.com/oleoleg/project-manager/internal/models"
	"github.com/oleoleg/project-manager/internal/services"
)

type StageHandler struct {
	svc         *services.StageService
	contractSvc *services.ContractService
	cpSvc       *services.CounterpartyService
	tmpl        *template.Template
}

func NewStageHandler(svc *services.StageService, contractSvc *services.ContractService, cpSvc *services.CounterpartyService, tmpl *template.Template) *StageHandler {
	return &StageHandler{svc: svc, contractSvc: contractSvc, cpSvc: cpSvc, tmpl: tmpl}
}

func (h *StageHandler) commonData(r *http.Request, title string) map[string]any {
	u := middleware.UserFromContext(r.Context())
	data := map[string]any{"Title": title, "User": u}
	if u != nil {
		data["RoleName"] = models.RoleName(u.RoleCode)
	}
	return data
}

func (h *StageHandler) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("template %s: %v", name, err)
	}
}

// ---------- List ----------

func (h *StageHandler) List(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	contract, err := h.contractSvc.GetByID(r.Context(), contractID)
	if errors.Is(err, db.ErrContractNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("stage list contract: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// создаём карточку, если ещё нет
	if err := h.svc.EnsureCard(r.Context(), contractID); err != nil {
		log.Printf("stage ensure card: %v", err)
	}

	stages, err := h.svc.List(r.Context(), contractID)
	if err != nil {
		log.Printf("stage list: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	statuses, _ := h.svc.Statuses(r.Context())
	workKinds, _ := h.svc.WorkKinds(r.Context())
	totals, _ := h.svc.Totals(r.Context(), contractID)
	card, _ := h.svc.GetCard(r.Context(), contractID)
	cp, _ := h.cpSvc.GetByID(r.Context(), contract.CounterpartyID)

	data := h.commonData(r, "Этапы договора "+contractID)
	data["Contract"] = contract
	data["Stages"] = stages
	data["Statuses"] = statuses
	data["WorkKinds"] = workKinds
	data["Totals"] = totals
	data["Card"] = card
	data["Counterparty"] = cp
	data["StatusName"] = func(code int16) string { return stageStatusName(statuses, code) }
	data["WorkKindName"] = func(code string) string { return workKindName(workKinds, code) }
	data["Flash"] = r.URL.Query().Get("flash")
	h.render(w, "stages_list.html", data)
}

// ---------- New / Create ----------

func (h *StageHandler) New(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	contract, err := h.contractSvc.GetByID(r.Context(), contractID)
	if errors.Is(err, db.ErrContractNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	statuses, _ := h.svc.Statuses(r.Context())
	workKinds, _ := h.svc.WorkKinds(r.Context())
	nextOrder, _ := h.svc.Totals(r.Context(), contractID)

	data := h.commonData(r, "Новый этап")
	data["Contract"] = contract
	data["Stage"] = &models.Stage{
		ContractID: contractID,
		RowOrder:   nextOrder.Count + 1,
		StatusCode: 0,
	}
	data["Statuses"] = statuses
	data["WorkKinds"] = workKinds
	data["IsNew"] = true
	data["Error"] = ""
	h.render(w, "stages_form.html", data)
}

func (h *StageHandler) Create(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	st, err := parseStageForm(r)
	if err != nil {
		// положим в stage-заглушку, чтобы шаблон отрисовал значения
		empty := &models.Stage{ContractID: contractID}
		h.renderFormWithError(w, r, empty, true, err.Error())
		return
	}
	st.ContractID = contractID

	if err := h.svc.Create(r.Context(), st); err != nil {
		h.renderFormWithError(w, r, st, true, humanizeStageError(err))
		return
	}
	http.Redirect(w, r, "/contracts/"+contractID+"/stages?flash="+url.QueryEscape("Этап создан"), http.StatusSeeOther)
}

// ---------- Edit / Update ----------

func (h *StageHandler) Edit(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	guid := chi.URLParam(r, "guid")

	contract, err := h.contractSvc.GetByID(r.Context(), contractID)
	if errors.Is(err, db.ErrContractNotFound) {
		http.NotFound(w, r)
		return
	}

	st, err := h.svc.GetByID(r.Context(), guid)
	if errors.Is(err, db.ErrStageNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("stage edit: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	h.renderFormWithError(w, r, st, false, "")
	_ = contract
}

func (h *StageHandler) Update(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	guid := chi.URLParam(r, "guid")

	st, err := parseStageForm(r)
	if err != nil {
		empty := &models.Stage{GUID: guid, ContractID: contractID}
		h.renderFormWithError(w, r, empty, false, err.Error())
		return
	}
	st.GUID = guid
	st.ContractID = contractID

	if err := h.svc.Update(r.Context(), st); err != nil {
		h.renderFormWithError(w, r, st, false, humanizeStageError(err))
		return
	}
	http.Redirect(w, r, "/contracts/"+contractID+"/stages?flash="+url.QueryEscape("Изменения сохранены"), http.StatusSeeOther)
}

func (h *StageHandler) Move(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	guid := chi.URLParam(r, "guid")

	dirStr := r.URL.Query().Get("dir")
	dir := 0
	switch dirStr {
	case "up":
		dir = -1
	case "down":
		dir = 1
	default:
		http.Error(w, "bad direction", http.StatusBadRequest)
		return
	}

	if err := h.svc.Move(r.Context(), contractID, guid, dir); err != nil {
		log.Printf("stage move: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/contracts/"+contractID+"/stages", http.StatusSeeOther)
}

// ---------- Delete ----------

func (h *StageHandler) Delete(w http.ResponseWriter, r *http.Request) {
	contractID := chi.URLParam(r, "id")
	guid := chi.URLParam(r, "guid")

	err := h.svc.Delete(r.Context(), guid)
	if errors.Is(err, db.ErrStageNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("stage delete: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/contracts/"+contractID+"/stages?flash="+url.QueryEscape("Этап удалён"), http.StatusSeeOther)
}

// ---------- helpers ----------

func (h *StageHandler) renderFormWithError(w http.ResponseWriter, r *http.Request, st *models.Stage, isNew bool, errMsg string) {
	contractID := chi.URLParam(r, "id")
	contract, _ := h.contractSvc.GetByID(r.Context(), contractID)
	statuses, _ := h.svc.Statuses(r.Context())
	workKinds, _ := h.svc.WorkKinds(r.Context())

	data := h.commonData(r, "Этап")
	data["Contract"] = contract
	data["Stage"] = st
	data["Statuses"] = statuses
	data["WorkKinds"] = workKinds
	data["IsNew"] = isNew
	data["Error"] = errMsg
	if errMsg != "" {
		w.WriteHeader(http.StatusBadRequest)
	}
	h.render(w, "stages_form.html", data)
}

func parseStageForm(r *http.Request) (*models.Stage, error) {
	_ = r.ParseForm()

	rowOrder, _ := strconv.Atoi(r.FormValue("row_order"))
	statusCode := int16(0)
	if v, err := strconv.Atoi(r.FormValue("status_code")); err == nil {
		statusCode = int16(v)
	}

	cost, err := services.ParseMoney(r.FormValue("cost"))
	if err != nil {
		return nil, fmt.Errorf("Стоимость: %w", err)
	}
	closed, err := services.ParseMoney(r.FormValue("closed_by_acts"))
	if err != nil {
		return nil, fmt.Errorf("Закрыто по актам: %w", err)
	}
	vat, err := parsePercent(r.FormValue("vat_percent"))
	if err != nil {
		return nil, fmt.Errorf("НДС: %w", err)
	}
	adv, err := parsePercent(r.FormValue("advance_percent"))
	if err != nil {
		return nil, fmt.Errorf("Аванс: %w", err)
	}

	return &models.Stage{
		RowOrder:          rowOrder,
		StageNumber:       strings.TrimSpace(r.FormValue("stage_number")),
		Name:              strings.TrimSpace(r.FormValue("name")),
		Cost:              cost,
		ClosedByActs:      closed,
		VATPercent:        vat,
		AdvancePercent:    adv,
		StartConditions:   strings.TrimSpace(r.FormValue("start_conditions")),
		EndConditions:     strings.TrimSpace(r.FormValue("end_conditions")),
		CloseDatePlan:     services.ParseDate(r.FormValue("close_date_plan")),
		ResponsiblePerson: strings.TrimSpace(r.FormValue("responsible_person")),
		StatusCode:        statusCode,
		WorkKindCode:      strings.TrimSpace(r.FormValue("work_kind_code")),
		Note:              strings.TrimSpace(r.FormValue("note")),
	}, nil
}

// parsePercent — как ParseMoney, но для процентов (0..100).
func parsePercent(s string) (float64, error) {
	v, err := services.ParseMoney(s)
	if err != nil {
		return 0, err
	}
	return v, nil
}

func humanizeStageError(err error) string {
	switch {
	case errors.Is(err, services.ErrStageNameRequired):
		return "Наименование этапа обязательно"
	case errors.Is(err, services.ErrStageWorkKindNeeded):
		return "Выберите вид работы"
	case errors.Is(err, services.ErrStageCostNegative):
		return "Стоимость и закрытая сумма не могут быть отрицательными"
	}
	return "Не удалось сохранить: " + err.Error()
}

func stageStatusName(list []models.StageStatusRef, code int16) string {
	for _, s := range list {
		if s.Code == code {
			return s.Name
		}
	}
	return ""
}

func workKindName(list []models.WorkKindRef, code string) string {
	for _, w := range list {
		if w.Code == code {
			return w.Name
		}
	}
	return code
}
