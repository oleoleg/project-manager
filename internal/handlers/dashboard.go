package handlers

import (
	"html/template"
	"log"
	"net/http"

	"github.com/oleoleg/project-manager/internal/middleware"
	"github.com/oleoleg/project-manager/internal/models"
)

type DashboardHandler struct {
	tmpl *template.Template
}

func NewDashboardHandler(tmpl *template.Template) *DashboardHandler {
	return &DashboardHandler{tmpl: tmpl}
}

func (h *DashboardHandler) Index(w http.ResponseWriter, r *http.Request) {
	u := middleware.UserFromContext(r.Context())
	if u == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	data := map[string]any{
		"User":     u,
		"RoleName": models.RoleName(u.RoleCode),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "dashboard.html", data); err != nil {
		log.Printf("template dashboard: %v", err)
	}
}
