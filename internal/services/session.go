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
		Secure:   false, // local: false; на проде — true
	}
	return &SessionManager{store: store}
}

func (m *SessionManager) SetUserID(w http.ResponseWriter, r *http.Request, userID int64) error {
	sess, err := m.store.Get(r, sessionName)
	if err != nil {
		return err
	}
	sess.Values[sessionUserKey] = userID
	return sess.Save(r, w)
}

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

func (m *SessionManager) Clear(w http.ResponseWriter, r *http.Request) error {
	sess, err := m.store.Get(r, sessionName)
	if err != nil {
		return err
	}
	sess.Options.MaxAge = -1
	return sess.Save(r, w)
}
