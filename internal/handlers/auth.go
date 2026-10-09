package handlers

import (
	"errors"
	"html/template"
	"log"
	"net/http"

	"github.com/oleoleg/project-manager/internal/services"
)

type AuthHandler struct {
	auth     *services.AuthService
	sessions *services.SessionManager
	tmpl     *template.Template
}

func NewAuthHandler(auth *services.AuthService, sm *services.SessionManager, tmpl *template.Template) *AuthHandler {
	return &AuthHandler{auth: auth, sessions: sm, tmpl: tmpl}
}

func (h *AuthHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.sessions.GetUserID(r); ok {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}
	h.render(w, "login.html", map[string]any{"Error": ""})
}

func (h *AuthHandler) LoginSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	username := r.FormValue("username")
	password := r.FormValue("password")

	u, err := h.auth.Authenticate(r.Context(), username, password)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) || errors.Is(err, services.ErrUserInactive) {
			w.WriteHeader(http.StatusUnauthorized)
			h.render(w, "login.html", map[string]any{"Error": "Неверный логин или пароль"})
			return
		}
		log.Printf("login error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := h.sessions.SetUserID(w, r, u.ID); err != nil {
		log.Printf("session save: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	_ = h.sessions.Clear(w, r)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *AuthHandler) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("template %s: %v", name, err)
	}
}
