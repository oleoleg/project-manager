package middleware

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/oleoleg/project-manager/internal/models"
	"github.com/oleoleg/project-manager/internal/services"
)

type ctxKey string

const userCtxKey ctxKey = "user"

// Auth достаёт пользователя из сессии, если он есть, и кладёт в контекст.
// Не блокирует — просто обогащает контекст.
func Auth(sm *services.SessionManager, auth *services.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, ok := sm.GetUserID(r)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			u, err := auth.GetByID(r.Context(), id)
			if err != nil || u == nil || !u.IsActive {
				// сессия битая — чистим
				_ = sm.Clear(w, r)
				next.ServeHTTP(w, r)
				return
			}
			ctx := context.WithValue(r.Context(), userCtxKey, u)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserFromContext возвращает пользователя или nil.
func UserFromContext(ctx context.Context) *models.User {
	u, _ := ctx.Value(userCtxKey).(*models.User)
	return u
}

// RequireAuth пускает только авторизованных, иначе — редирект на /login.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if UserFromContext(r.Context()) == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireRole пускает только указанные роли, иначе 403.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := UserFromContext(r.Context())
			if u == nil {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			if _, ok := allowed[u.RoleCode]; !ok {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireContractOwnerOrRoles пропускает, если пользователь — одна из указанных ролей
// ИЛИ он назначен ответственным по договору {id}.
// Требует передать функцию, которая умеет достать contract.ResponsibleUserID по contractID.
func RequireContractOwnerOrRoles(
	roles []string,
	getOwner func(ctx context.Context, contractID string) (int64, bool),
) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := UserFromContext(r.Context())
			if u == nil {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			// роль — пускаем без вопросов
			if _, ok := allowed[u.RoleCode]; ok {
				next.ServeHTTP(w, r)
				return
			}
			// роль rp — проверяем владение
			contractID := chi.URLParam(r, "id")
			ownerID, ok := getOwner(r.Context(), contractID)
			if !ok || ownerID != u.ID {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
