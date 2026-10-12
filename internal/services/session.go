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

// SetUserID всегда создаёт новую сессию, не читая существующую cookie.
// Это устраняет проблему «securecookie: the value is not valid»,
// когда cookie зашифрована старым SESSION_SECRET.
func (m *SessionManager) SetUserID(w http.ResponseWriter, r *http.Request, userID int64) error {
	sess, err := m.store.New(r, sessionName)
	if err != nil {
		return err
	}
	sess.Values[sessionUserKey] = userID
	return sess.Save(r, w)
}

// GetUserID читает сессию. Если cookie нет или она невалидна —
// просто возвращает (0, false), не падая.
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

// Clear сбрасывает сессию. Если cookie битая — создаёт новую с MaxAge=-1,
// чтобы браузер её удалил.
func (m *SessionManager) Clear(w http.ResponseWriter, r *http.Request) error {
	sess, err := m.store.Get(r, sessionName)
	if err != nil {
		sess, _ = m.store.New(r, sessionName)
	}
	sess.Options.MaxAge = -1
	return sess.Save(r, w)
}
