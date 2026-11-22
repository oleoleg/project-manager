package handlers

import (
	"errors"
	"html/template"
	"log"
	"net/http"
	"net/url"

	"github.com/oleoleg/project-manager/internal/middleware"
	"github.com/oleoleg/project-manager/internal/services"
)

type SettingsHandler struct {
	userSvc *services.UserService
	tmpl    *template.Template
}

func NewSettingsHandler(userSvc *services.UserService, tmpl *template.Template) *SettingsHandler {
	return &SettingsHandler{userSvc: userSvc, tmpl: tmpl}
}

func (h *SettingsHandler) Index(w http.ResponseWriter, r *http.Request) {
	u := middleware.UserFromContext(r.Context())
	if u == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	data := BaseData(r, "Настройки")
	data["Flash"] = r.URL.Query().Get("flash")
	RenderTemplate(w, h.tmpl, "settings.html", data)
}

func (h *SettingsHandler) SaveTheme(w http.ResponseWriter, r *http.Request) {
	u := middleware.UserFromContext(r.Context())
	if u == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	_ = r.ParseForm()
	theme := r.FormValue("theme")

	if err := h.userSvc.SetTheme(r.Context(), u.ID, theme); err != nil {
		if errors.Is(err, services.ErrInvalidTheme) {
			data := BaseData(r, "Настройки")
			data["Error"] = "Недопустимая тема"
			w.WriteHeader(http.StatusBadRequest)
			RenderTemplate(w, h.tmpl, "settings.html", data)
			return
		}
		log.Printf("save theme: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/settings?flash="+url.QueryEscape("Тема сохранена"), http.StatusSeeOther)
}

func (h *SettingsHandler) SavePrivacy(w http.ResponseWriter, r *http.Request) {
	u := middleware.UserFromContext(r.Context())
	if u == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	_ = r.ParseForm()
	hide := r.FormValue("hide_financials") == "on"
	if err := h.userSvc.SetHideFinancials(r.Context(), u.ID, hide); err != nil {
		log.Printf("save privacy: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/settings?flash="+url.QueryEscape("Сохранено"), http.StatusSeeOther)
}

func (h *SettingsHandler) SaveCheckPath(w http.ResponseWriter, r *http.Request) {
	u := middleware.UserFromContext(r.Context())
	if u == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	_ = r.ParseForm()
	check := r.FormValue("check_contract_path") == "on"
	if err := h.userSvc.SetCheckContractPath(r.Context(), u.ID, check); err != nil {
		log.Printf("save check path: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/settings?flash="+url.QueryEscape("Сохранено"), http.StatusSeeOther)
}
