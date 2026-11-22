package handlers

import (
	"errors"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/oleoleg/project-manager/internal/db"
	"github.com/oleoleg/project-manager/internal/models"
	"github.com/oleoleg/project-manager/internal/services"
)

type CounterpartyHandler struct {
	svc  *services.CounterpartyService
	tmpl *template.Template
}

func NewCounterpartyHandler(svc *services.CounterpartyService, tmpl *template.Template) *CounterpartyHandler {
	return &CounterpartyHandler{svc: svc, tmpl: tmpl}
}

// ---------- List ----------

func (h *CounterpartyHandler) List(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("q"))

	list, err := h.svc.List(r.Context(), search)
	if err != nil {
		log.Printf("counterparties list: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	data := BaseData(r, "Контрагенты")
	data["List"] = list
	data["Search"] = search
	data["Flash"] = r.URL.Query().Get("flash")
	RenderTemplate(w, h.tmpl, "counterparties_list.html", data)
}

// ---------- Show ----------

func (h *CounterpartyHandler) Show(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	c, err := h.svc.GetByID(r.Context(), id)
	if errors.Is(err, db.ErrCounterpartyNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("counterparty show: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	data := BaseData(r, "Контрагент "+c.ID)
	data["Counterparty"] = c
	RenderTemplate(w, h.tmpl, "counterparties_show.html", data)
}

// ---------- New / Create ----------

func (h *CounterpartyHandler) New(w http.ResponseWriter, r *http.Request) {
	data := BaseData(r, "Новый контрагент")
	data["Counterparty"] = &models.Counterparty{}
	data["IsNew"] = true
	data["Error"] = ""
	RenderTemplate(w, h.tmpl, "counterparties_form.html", data)
}

func (h *CounterpartyHandler) Create(w http.ResponseWriter, r *http.Request) {
	c := parseCounterpartyForm(r)

	err := h.svc.Create(r.Context(), c)
	if err != nil {
		msg := humanizeCounterpartyError(err)
		data := BaseData(r, "Новый контрагент")
		data["Counterparty"] = c
		data["IsNew"] = true
		data["Error"] = msg
		w.WriteHeader(http.StatusBadRequest)
		RenderTemplate(w, h.tmpl, "counterparties_form.html", data)
		return
	}

	http.Redirect(w, r, "/counterparties?flash="+urlEncode("Контрагент создан"), http.StatusSeeOther)
}

// ---------- Edit / Update ----------

func (h *CounterpartyHandler) Edit(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	c, err := h.svc.GetByID(r.Context(), id)
	if errors.Is(err, db.ErrCounterpartyNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("counterparty edit: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	data := BaseData(r, "Редактирование "+c.ID)
	data["Counterparty"] = c
	data["IsNew"] = false
	data["Error"] = ""
	RenderTemplate(w, h.tmpl, "counterparties_form.html", data)
}

func (h *CounterpartyHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	c := parseCounterpartyForm(r)
	c.ID = id // ID из URL — его менять нельзя

	err := h.svc.Update(r.Context(), c)
	if err != nil {
		msg := humanizeCounterpartyError(err)
		data := BaseData(r, "Редактирование "+c.ID)
		data["Counterparty"] = c
		data["IsNew"] = false
		data["Error"] = msg
		w.WriteHeader(http.StatusBadRequest)
		RenderTemplate(w, h.tmpl, "counterparties_form.html", data)
		return
	}

	http.Redirect(w, r, "/counterparties?flash="+urlEncode("Изменения сохранены"), http.StatusSeeOther)
}

// ---------- Delete ----------

func (h *CounterpartyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	err := h.svc.Delete(r.Context(), id)
	if errors.Is(err, db.ErrCounterpartyNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("counterparty delete: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/counterparties?flash="+urlEncode("Контрагент удалён"), http.StatusSeeOther)
}

// ---------- helpers ----------

func parseCounterpartyForm(r *http.Request) *models.Counterparty {
	_ = r.ParseForm()
	return &models.Counterparty{
		ID:             strings.TrimSpace(r.FormValue("id")),
		Name:           strings.TrimSpace(r.FormValue("name")),
		Partner:        strings.TrimSpace(r.FormValue("partner")),
		ShortName:      strings.TrimSpace(r.FormValue("short_name")),
		AddressActual:  strings.TrimSpace(r.FormValue("address_actual")),
		AddressLegal:   strings.TrimSpace(r.FormValue("address_legal")),
		URLWorkplace:   strings.TrimSpace(r.FormValue("url_workplace")),
		URLServicework: strings.TrimSpace(r.FormValue("url_servicework")),
		URLSalesprep:   strings.TrimSpace(r.FormValue("url_salesprep")),
	}
}

func humanizeCounterpartyError(err error) string {
	switch {
	case errors.Is(err, services.ErrCounterpartyIDFormat):
		return "ID должен состоять ровно из 4 цифр (например, 0001)"
	case errors.Is(err, services.ErrCounterpartyIDRange):
		return "ID контрагента должен быть числом от 0001 до 1999"
	case errors.Is(err, services.ErrCounterpartyNameEmpty):
		return "Наименование обязательно"
	case errors.Is(err, services.ErrCounterpartyExists):
		return "Контрагент с таким ID уже существует"
	}
	return "Не удалось сохранить: " + err.Error()
}

func urlEncode(s string) string { return url.QueryEscape(s) }
