package handlers

import (
	"errors"
	"html/template"
	"log"
	"net/http"
	"net/url"

	"github.com/oleoleg/project-manager/internal/middleware"
	"github.com/oleoleg/project-manager/internal/models"
	"github.com/oleoleg/project-manager/internal/services"
)

type SettingsHandler struct {
	userSvc *services.UserService
	tmpl    *template.Template
}

func NewSettingsHandler(userSvc *services.UserService, tmpl *template.Template) *SettingsHandler {
	return &SettingsHandler{userSvc: userSvc, tmpl: tmpl}
}

func (h *SettingsHandler) commonData(r *http.Request, title string) map[string]any {
	u := middleware.UserFromContext(r.Context())
	data := map[string]any{"Title": title, "User": u}
	if u != nil {
		data["RoleName"] = models.RoleName(u.RoleCode)
	}
	return data
}

func (h *SettingsHandler) Index(w http.ResponseWriter, r *http.Request) {
	u := middleware.UserFromContext(r.Context())
	if u == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	data := h.commonData(r, "Настройки")
	data["Flash"] = r.URL.Query().Get("flash")
	h.render(w, "settings.html", data)
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
			data := h.commonData(r, "Настройки")
			data["Error"] = "Недопустимая тема"
			w.WriteHeader(http.StatusBadRequest)
			h.render(w, "settings.html", data)
			return
		}
		log.Printf("save theme: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/settings?flash="+url.QueryEscape("Тема сохранена"), http.StatusSeeOther)
}

func (h *SettingsHandler) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("template %s: %v", name, err)
	}
}
