package services

import (
	"net/http"

	"github.com/gorilla/sessions"
)

const (
	sessionName    = "pm_session"
	sessionUserKey = "user_id"
)

type SessionManager struct {
	store *sessions.CookieStore
}

func NewSessionManager(secret string) *SessionManager {
	store := sessions.NewCookieStore([]byte(secret))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 дней
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false, // local: false; на проде — true при HTTPS
	}
	return &SessionManager{store: store}
}

// SetUserID сохраняет ID пользователя в сессию.
// Если в браузере осталась невалидная cookie (например, зашифрованная
// старым SESSION_SECRET), она сначала удаляется принудительно —
// иначе gorilla/sessions падает с "securecookie: the value is not valid".
func (m *SessionManager) SetUserID(w http.ResponseWriter, r *http.Request, userID int64) error {
	// Проверяем, что пришло в cookie
	if _, err := m.store.Get(r, sessionName); err != nil {
		// Старая cookie невалидна — удаляем её принудительно
		http.SetCookie(w, &http.Cookie{
			Name:     sessionName,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
	}

	sess, err := m.store.New(r, sessionName)
	if err != nil {
		return err
	}
	sess.Values[sessionUserKey] = userID
	return sess.Save(r, w)
}

// GetUserID читает сессию. Если cookie нет или она невалидна — возвращает (0, false).
func (m *SessionManager) GetUserID(r *http.Request) (int64, bool) {
	sess, err := m.store.Get(r, sessionName)
	if err != nil {
		return 0, false
	}
	v, ok := sess.Values[sessionUserKey]
	if !ok {
		return 0, false
	}
	id, ok := v.(int64)
	return id, ok
}

// Clear сбрасывает сессию, принудительно удаляя cookie у клиента.
func (m *SessionManager) Clear(w http.ResponseWriter, r *http.Request) error {
	// Всегда выставляем MaxAge=-1 через Set-Cookie — это самый надёжный способ
	http.SetCookie(w, &http.Cookie{
		Name:     sessionName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	// И пробуем сохранить через store, чтобы схема cookie совпадала
	sess, err := m.store.Get(r, sessionName)
	if err != nil {
		sess, _ = m.store.New(r, sessionName)
	}
	sess.Options.MaxAge = -1
	return sess.Save(r, w)
}
