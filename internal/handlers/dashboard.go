package handlers

import (
	"html/template"
	"net/http"

	"github.com/oleoleg/project-manager/internal/middleware"
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

	data := BaseData(r, "Дашборд")

	RenderTemplate(w, h.tmpl, "dashboard.html", data)
}
