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

	"github.com/oleoleg/project-manager/internal/db"
	"github.com/oleoleg/project-manager/internal/middleware"
	"github.com/oleoleg/project-manager/internal/models"
	"github.com/oleoleg/project-manager/internal/services"
)

type AdminUserHandler struct {
	userSvc *services.UserService
	tmpl    *template.Template
}

func NewAdminUserHandler(userSvc *services.UserService, tmpl *template.Template) *AdminUserHandler {
	return &AdminUserHandler{userSvc: userSvc, tmpl: tmpl}
}

func (h *AdminUserHandler) commonData(r *http.Request, title string) map[string]any {
	u := middleware.UserFromContext(r.Context())
	data := map[string]any{
		"Title": title,
		"User":  u,
		"RoleLabel": func(code string) string {
			return models.RoleName(code)
		},
	}
	if u != nil {
		data["RoleName"] = models.RoleName(u.RoleCode)
	}
	return data
}

func (h *AdminUserHandler) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("template %s: %v", name, err)
	}
}

// ---------- List ----------

func (h *AdminUserHandler) List(w http.ResponseWriter, r *http.Request) {
	users, err := h.userSvc.List(r.Context())
	if err != nil {
		log.Printf("admin users list: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	data := h.commonData(r, "Пользователи")
	data["Users"] = users
	data["Flash"] = r.URL.Query().Get("flash")
	h.render(w, "admin_users_list.html", data)
}

// ---------- New / Create ----------

func (h *AdminUserHandler) New(w http.ResponseWriter, r *http.Request) {
	roles, _ := h.userSvc.ListRoles(r.Context())
	data := h.commonData(r, "Новый пользователь")
	data["FormUser"] = &models.User{IsActive: true, Theme: "light"}
	data["Roles"] = roles
	data["IsNew"] = true
	data["Error"] = ""
	h.render(w, "admin_users_form.html", data)
}

func (h *AdminUserHandler) Create(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	u := &models.User{
		Username: strings.TrimSpace(r.FormValue("username")),
		FullName: strings.TrimSpace(r.FormValue("full_name")),
		RoleCode: strings.TrimSpace(r.FormValue("role_code")),
		IsActive: r.FormValue("is_active") == "on",
		Theme:    "light",
	}
	password := r.FormValue("password")

	if err := h.userSvc.Create(r.Context(), u, password); err != nil {
		h.renderFormWithError(w, r, u, true, humanizeUserError(err))
		return
	}
	http.Redirect(w, r, "/admin/users?flash="+url.QueryEscape("Пользователь создан"), http.StatusSeeOther)
}

// ---------- Edit / Update ----------

func (h *AdminUserHandler) Edit(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	u, err := h.userSvc.GetByID(r.Context(), id)
	if errors.Is(err, db.ErrUserNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("admin user edit: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.renderFormWithError(w, r, u, false, "")
}

func (h *AdminUserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	_ = r.ParseForm()
	u := &models.User{
		ID:       id,
		Username: strings.TrimSpace(r.FormValue("username")),
		FullName: strings.TrimSpace(r.FormValue("full_name")),
		RoleCode: strings.TrimSpace(r.FormValue("role_code")),
		IsActive: r.FormValue("is_active") == "on",
	}
	if err := h.userSvc.Update(r.Context(), u); err != nil {
		h.renderFormWithError(w, r, u, false, humanizeUserError(err))
		return
	}
	http.Redirect(w, r, "/admin/users?flash="+url.QueryEscape("Пользователь обновлён"), http.StatusSeeOther)
}

// ---------- Password ----------

func (h *AdminUserHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	_ = r.ParseForm()
	password := r.FormValue("password")

	if err := h.userSvc.ChangePassword(r.Context(), id, password); err != nil {
		u, _ := h.userSvc.GetByID(r.Context(), id)
		h.renderFormWithError(w, r, u, false, humanizeUserError(err))
		return
	}
	http.Redirect(w, r, "/admin/users?flash="+url.QueryEscape("Пароль изменён"), http.StatusSeeOther)
}

// ---------- Toggle active ----------

func (h *AdminUserHandler) ToggleActive(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	current := middleware.UserFromContext(r.Context())
	var currentID int64
	if current != nil {
		currentID = current.ID
	}
	if err := h.userSvc.ToggleActive(r.Context(), id, currentID); err != nil {
		http.Redirect(w, r, "/admin/users?flash="+url.QueryEscape(humanizeUserError(err)), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin/users?flash="+url.QueryEscape("Статус изменён"), http.StatusSeeOther)
}

// ---------- helpers ----------

func (h *AdminUserHandler) renderFormWithError(w http.ResponseWriter, r *http.Request, u *models.User, isNew bool, errMsg string) {
	roles, _ := h.userSvc.ListRoles(r.Context())
	data := h.commonData(r, "Пользователь")
	data["FormUser"] = u
	data["Roles"] = roles
	data["IsNew"] = isNew
	data["Error"] = errMsg
	if errMsg != "" {
		w.WriteHeader(http.StatusBadRequest)
	}
	h.render(w, "admin_users_form.html", data)
}

func humanizeUserError(err error) string {
	switch {
	case errors.Is(err, services.ErrUsernameRequired):
		return "Логин обязателен"
	case errors.Is(err, services.ErrFullNameRequired):
		return "ФИО обязательно"
	case errors.Is(err, services.ErrPasswordTooShort):
		return "Пароль должен быть не короче 6 символов"
	case errors.Is(err, services.ErrUsernameExists):
		return "Пользователь с таким логином уже существует"
	case errors.Is(err, services.ErrRoleRequired):
		return "Выберите роль"
	case errors.Is(err, services.ErrCannotDeleteSelf):
		return "Нельзя деактивировать самого себя"
	}
	return "Ошибка: " + err.Error()
}
